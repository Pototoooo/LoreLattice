package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Pototoooo/lorelattice/internal/logger"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db           *gorm.DB
	config       Config
	client       *Client
	kick         chan struct{}
	pricing      *PriceCatalog
	payment      *AlipayClient
	paymentError error
	settingsMu   sync.Mutex
}

func NewService(db *gorm.DB) *Service {
	cfg := LoadConfigFromEnv()
	pricing, err := NewPriceCatalog(cfg.ModelPricingJSON)
	// Bad pricing must not silently use an unintended sale price.
	if err != nil {
		logger.Errorf(context.Background(), "[Billing] invalid price catalog: %v", err)
	}
	payment, paymentErr := NewAlipayClientFromEnv()
	if paymentErr != nil {
		logger.Errorf(context.Background(), "[Billing] payment configuration invalid: %v", paymentErr)
	}
	return &Service{db: db, config: cfg, client: NewClient(cfg), kick: make(chan struct{}, 1), pricing: pricing, payment: payment, paymentError: paymentErr}
}

func (s *Service) Enabled() bool { return s != nil && s.config.Enabled }
func (s *Service) DefaultCompletionReserve() int {
	if s == nil || s.config.DefaultCompletionReserve <= 0 {
		return 256
	}
	return s.config.DefaultCompletionReserve
}
func StartService(s *Service) {
	SetDefault(s)
	if !s.Enabled() {
		return
	}
	go s.runMaintenance()
	if s.config.MeterForgeEnabled {
		go s.runOutbox()
	}
}
func (s *Service) runMaintenance() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.provisioningPass(context.Background()); err != nil {
			logger.Errorf(context.Background(), "[Billing] maintenance: %v", err)
		}
		<-ticker.C
	}
}
func (s *Service) provisioningPass(ctx context.Context) error {
	var tenants []types.Tenant
	if err := s.db.WithContext(ctx).Find(&tenants).Error; err != nil {
		return err
	}
	for _, tenant := range tenants {
		if err := s.ProvisionTenant(ctx, tenant.ID, tenant.Name); err != nil {
			return err
		}
	}
	if err := s.RecoverReservations(ctx); err != nil {
		return err
	}
	return s.ReconcilePayments(ctx)
}
func (s *Service) ProvisionTenantAsync(id uint64, name string) {
	if !s.Enabled() || id == 0 {
		return
	}
	go func() {
		if err := s.ProvisionTenant(context.Background(), id, name); err != nil {
			logger.Errorf(context.Background(), "[Billing] provision tenant %d: %v", id, err)
		}
	}()
}
func (s *Service) ProvisionTenant(ctx context.Context, id uint64, _ string) error {
	if !s.Enabled() {
		return nil
	}
	if id == 0 {
		return fmt.Errorf("missing workspace")
	}
	// Existing accounts and their money are preserved; no remote subscriptions or grants.
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&Account{
		TenantID: id, MeterForgeCustomerKey: fmt.Sprintf("lorelattice_tenant_%d", id),
		Status: "active", PlanKey: "wallet", LocalCreditGranted: s.config.WelcomeCreditUSD,
	}).Error
}

func (s *Service) Reserve(ctx context.Context, tenantID uint64, feature FeatureKey, quantity float64, meta UsageMetadata) (*Reservation, error) {
	if !s.Enabled() {
		return nil, nil
	}
	def, ok := FeatureDefinitions[feature]
	if !ok || tenantID == 0 || quantity <= 0 || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return nil, fmt.Errorf("invalid billing reservation")
	}
	if err := s.ProvisionTenant(ctx, tenantID, ""); err != nil {
		return nil, err
	}
	mode := meta.Mode
	if !mode.Valid() {
		mode = BillingModePlatform
	}
	// Legacy included configuration becomes paid platform usage; historical rows stay intact.
	if mode == BillingModeIncluded {
		mode = BillingModePlatform
	}
	if meta.JobID == "" {
		meta.JobID = meta.RequestID
	}
	if meta.JobID == "" {
		meta.JobID = uuid.NewString()
	}
	if meta.Category == "" {
		meta.Category = BusinessCategoryForOperation(meta.Operation)
	}
	pricing := s.pricing
	if pricing == nil {
		if mode.Chargeable() {
			return nil, billingError("billing_not_ready", "模型价格配置无效", 503, feature, nil)
		}
		pricing = &PriceCatalog{}
	}
	unitPrice, priceVersion := pricing.Resolve(feature, meta.Provider, meta.ModelName, mode)
	cost := math.Ceil(quantity*unitPrice*1e8) / 1e8
	if math.IsInf(cost, 0) || cost > 1e10 {
		return nil, fmt.Errorf("invalid reservation cost")
	}
	var result *Reservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
			return err
		}
		if mode.Chargeable() {
			if account.Status == "suspended" {
				return billingError("billing_not_ready", "账户已暂停，请联系管理员", 403, feature, nil)
			}
			var spent float64
			if err := tx.Model(&Reservation{}).Select("COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost ELSE actual_cost END), 0)").Where("tenant_id = ? AND chargeable = ? AND status IN ?", tenantID, true, []string{"reserved", "completed", "recovered"}).Scan(&spent).Error; err != nil {
				return err
			}
			if cost > account.LocalCreditGranted-spent+1e-9 {
				return billingError("insufficient_credit", "余额不足，请充值或使用自备模型", 402, feature, nil)
			}
		}
		now := time.Now().UTC()
		result = &Reservation{ID: uuid.NewString(), TenantID: tenantID, FeatureKey: string(feature), EventType: def.EventType, MeterUnit: def.Unit,
			ModelID: meta.ModelID, ModelName: meta.ModelName, Provider: meta.Provider, Operation: meta.Operation,
			RequestID: meta.RequestID, JobID: meta.JobID, BusinessCategory: meta.Category, BillingMode: string(mode),
			Chargeable: mode.Chargeable(), PriceVersion: priceVersion, ReservedQuantity: quantity, UnitPrice: unitPrice,
			ReservedCost: cost, Estimated: meta.Estimated, Status: "reserved", PeriodStartedAt: now}
		return tx.Create(result).Error
	})
	return result, err
}

func (s *Service) Complete(ctx context.Context, reservation *Reservation, actual float64, estimated bool) error {
	if !s.Enabled() || reservation == nil {
		return nil
	}
	if math.IsNaN(actual) || math.IsInf(actual, 0) {
		return fmt.Errorf("invalid actual usage")
	}
	if actual <= 0 {
		return s.Release(ctx, reservation, "provider returned no billable usage")
	}
	now := time.Now().UTC()
	actualCost := math.Min(actual*reservation.UnitPrice, reservation.ReservedCost)
	actualCost = math.Round(actualCost*1e8) / 1e8
	event := map[string]any{
		"specversion": "1.0", "type": "lorelattice.wallet.usage.v1", "id": reservation.ID,
		"time": now, "source": s.config.Source,
		"subject": s.config.SubjectPrefix + "wallet:" + fmt.Sprint(reservation.TenantID),
		"data": map[string]any{
			"quantity": actual, "unit": reservation.MeterUnit,
			"feature_key": reservation.FeatureKey, "model_id": reservation.ModelID,
			"model": reservation.ModelName, "provider": reservation.Provider,
			"operation": reservation.Operation, "request_id": reservation.RequestID,
			"job_id": reservation.JobID, "business_category": reservation.BusinessCategory,
			"billing_mode": reservation.BillingMode, "chargeable": reservation.Chargeable,
			"estimated": estimated, "unit_price_usd": reservation.UnitPrice,
			"cost_usd": actualCost, "price_version": reservation.PriceVersion,
		},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&Reservation{}).Where("id = ? AND status IN ('reserved', 'recovered')", reservation.ID).
			Updates(map[string]any{
				"actual_quantity": actual, "actual_cost": actualCost, "estimated": estimated,
				"status": "completed", "completed_at": now, "updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		// All modes use a new telemetry-only event/subject namespace.
		// No remote customer, priced rate card or balance participates.
		if !s.config.MeterForgeEnabled {
			return nil
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&OutboxRow{
			EventID: reservation.ID, TenantID: reservation.TenantID,
			ReservationID: reservation.ID, Payload: payload,
			Status: "pending", NextAttemptAt: now,
		}).Error
	})
	if err == nil {
		select {
		case s.kick <- struct{}{}:
		default:
		}
	}
	return err
}

func (s *Service) Release(ctx context.Context, reservation *Reservation, reason string) error {
	if !s.Enabled() || reservation == nil {
		return nil
	}
	return s.db.WithContext(ctx).Model(&Reservation{}).Where("id = ? AND status IN ?", reservation.ID, []string{"reserved", "recovered"}).Updates(map[string]any{"status": "released", "error_message": reason, "updated_at": time.Now().UTC()}).Error
}

// A crashed call cannot be proven free. Settle at its authorized estimate after
// 24h, mark it for review, and allow a late definitive completion/release to fix it.
func (s *Service) RecoverReservations(ctx context.Context) error {
	return s.db.WithContext(ctx).Model(&Reservation{}).Where("status = ? AND created_at < ?", "reserved", time.Now().UTC().Add(-s.config.ReservationTTL)).Updates(map[string]any{
		"status": "recovered", "actual_quantity": gorm.Expr("reserved_quantity"), "actual_cost": gorm.Expr("reserved_cost"),
		"estimated": true, "error_message": "调用中断，按预留金额暂结；待核查", "completed_at": time.Now().UTC(), "updated_at": time.Now().UTC(),
	}).Error
}
func (s *Service) runOutbox() {
	ticker := time.NewTicker(s.config.OutboxInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-s.kick:
		}
		if err := s.FlushOutbox(context.Background(), 50); err != nil {
			logger.Warnf(context.Background(), "[Billing] telemetry: %v", err)
		}
	}
}
func (s *Service) FlushOutbox(ctx context.Context, limit int) error {
	// Claim rows under database locks. Stable event IDs remain safe after a crash.
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []OutboxRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status IN ? AND next_attempt_at <= ?", []string{"pending", "failed"}, time.Now().UTC()).Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			// Never resend legacy priced events into old subscriptions.
			var event struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(row.Payload, &event); err != nil {
				return err
			}
			if event.Type != "lorelattice.wallet.usage.v1" {
				if err := tx.Model(&row).Updates(map[string]any{"status": "legacy", "last_error": "legacy priced event retained for manual audit"}).Error; err != nil {
					return err
				}
				continue
			}
			err := s.client.SendEvent(ctx, row.EventID, row.Payload)
			now := time.Now().UTC()
			updates := map[string]any{"status": "sent", "sent_at": now, "last_error": "", "updated_at": now}
			if err != nil {
				attempts := row.Attempts + 1
				updates = map[string]any{"status": "failed", "attempts": attempts, "last_error": err.Error(), "next_attempt_at": now.Add(time.Duration(1<<min(attempts, 6)) * time.Second), "updated_at": now}
			}
			if err := tx.Model(&row).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func billingError(code, message string, status int, feature FeatureKey, resetAt *time.Time) *BillingError {
	return &BillingError{Code: code, Message: message, HTTPStatus: status, Feature: feature, ResetAt: resetAt}
}
