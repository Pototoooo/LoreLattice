package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FeatureOverview struct {
	Key       FeatureKey `json:"key"`
	Name      string     `json:"name"`
	Unit      string     `json:"unit"`
	Limit     float64    `json:"limit"`
	Used      float64    `json:"used"`
	Remaining float64    `json:"remaining"`
	UnitPrice float64    `json:"unit_price_usd"`
}

type Overview struct {
	Enabled               bool              `json:"enabled"`
	Status                string            `json:"status"`
	PlanKey               string            `json:"plan_key"`
	PlanName              string            `json:"plan_name"`
	TrialEndsAt           *time.Time        `json:"trial_ends_at,omitempty"`
	PeriodStartedAt       *time.Time        `json:"period_started_at,omitempty"`
	PeriodEndsAt          *time.Time        `json:"period_ends_at,omitempty"`
	Features              []FeatureOverview `json:"features"`
	CreditBalanceUSD      *float64          `json:"credit_balance_usd,omitempty"`
	CancelScheduled       bool              `json:"cancel_scheduled"`
	MeterForgeCustomerID  string            `json:"-"`
	MeterForgeCustomerKey string            `json:"-"`
}

func (s *Service) Overview(ctx context.Context, tenantID uint64, includeFinancial bool) (*Overview, error) {
	if !s.Enabled() {
		return &Overview{Enabled: false, Status: "disabled"}, nil
	}
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, billingError("billing_not_ready", "计费账户尚未开通", http.StatusServiceUnavailable, "", nil)
	}
	if account.MeterForgeCustomerID == "" {
		return nil, billingError("billing_not_ready", "计费账户正在开通", http.StatusServiceUnavailable, "", nil)
	}
	result := &Overview{
		Enabled: true, Status: account.Status, PlanKey: account.PlanKey,
		PlanName:    map[string]string{PlanTrial: "Trial", PlanPro: "Pro"}[account.PlanKey],
		TrialEndsAt: account.TrialEndsAt, PeriodStartedAt: account.PeriodStartedAt,
		PeriodEndsAt: account.PeriodEndsAt, CancelScheduled: account.Status == "cancel_scheduled",
		MeterForgeCustomerID:  account.MeterForgeCustomerID,
		MeterForgeCustomerKey: account.MeterForgeCustomerKey,
	}
	trialExpired := account.PlanKey == PlanTrial && account.TrialEndsAt != nil &&
		!time.Now().UTC().Before(*account.TrialEndsAt)
	if trialExpired {
		result.Status = "expired"
	}
	for _, def := range orderedFeatureDefinitions() {
		entitlement, err := s.client.EntitlementValue(ctx, account.MeterForgeCustomerKey, def.Key)
		if err != nil {
			return nil, billingError("billing_unavailable", "计费服务暂时不可用", http.StatusServiceUnavailable, def.Key, account.PeriodEndsAt)
		}
		limit := def.TrialLimit
		if account.PlanKey == PlanPro {
			limit = def.ProLimit
		}
		var localUsed float64
		query := s.db.WithContext(ctx).Model(&Reservation{}).
			Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_quantity WHEN status = 'completed' THEN actual_quantity ELSE 0 END), 0)`).
			Where("tenant_id = ? AND feature_key = ? AND status IN ?", tenantID, def.Key, []string{"reserved", "completed"})
		if account.PeriodStartedAt != nil {
			query = query.Where("period_started_at = ?", *account.PeriodStartedAt)
		}
		if err := query.Scan(&localUsed).Error; err != nil {
			return nil, err
		}
		remaining := math.Min(math.Max(0, limit-localUsed), math.Max(0, entitlement.Balance))
		if trialExpired {
			remaining = 0
		}
		result.Features = append(result.Features, FeatureOverview{
			Key: def.Key, Name: def.Name, Unit: def.Unit, Limit: limit,
			Used: math.Max(localUsed, entitlement.Usage), Remaining: remaining, UnitPrice: def.UnitPrice,
		})
	}
	if includeFinancial {
		balance, err := s.client.CreditBalance(ctx, account.MeterForgeCustomerID)
		if err != nil {
			return nil, billingError("billing_unavailable", "计费服务暂时不可用", http.StatusServiceUnavailable, "", account.PeriodEndsAt)
		}
		localBalance, err := s.localAvailableCredit(ctx, tenantID, account.LocalCreditGranted)
		if err != nil {
			return nil, err
		}
		balance = math.Min(balance, localBalance)
		result.CreditBalanceUSD = &balance
	}
	return result, nil
}

func (s *Service) localAvailableCredit(ctx context.Context, tenantID uint64, granted float64) (float64, error) {
	var localCost float64
	if err := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost WHEN status = 'completed' THEN actual_cost ELSE 0 END), 0)`).
		Where("tenant_id = ? AND status IN ?", tenantID, []string{"reserved", "completed"}).
		Scan(&localCost).Error; err != nil {
		return 0, err
	}
	return math.Max(0, granted-localCost), nil
}

type UsageRow struct {
	ID          string     `json:"id"`
	FeatureKey  string     `json:"feature_key"`
	ModelName   string     `json:"model"`
	Provider    string     `json:"provider"`
	Operation   string     `json:"operation"`
	Quantity    *float64   `json:"quantity"`
	Unit        string     `json:"unit"`
	CostUSD     *float64   `json:"cost_usd"`
	Estimated   bool       `json:"estimated"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (s *Service) Usage(
	ctx context.Context,
	tenantID uint64,
	from, to *time.Time,
	feature FeatureKey,
	model, provider string,
	limit int,
) ([]UsageRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`id, feature_key, model_name, provider, operation, actual_quantity AS quantity,
		        meter_unit AS unit, actual_cost AS cost_usd, estimated, status, created_at, completed_at`).
		Where("tenant_id = ?", tenantID)
	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at < ?", *to)
	}
	if feature != "" {
		query = query.Where("feature_key = ?", feature)
	}
	if model = strings.TrimSpace(model); model != "" {
		query = query.Where("model_name = ?", model)
	}
	if provider = strings.TrimSpace(provider); provider != "" {
		query = query.Where("provider = ?", provider)
	}
	var rows []UsageRow
	err := query.Order("created_at DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (s *Service) TopUp(ctx context.Context, tenantID uint64, amount float64, idempotencyKey string) (*Overview, error) {
	if amount != 1 && amount != 5 && amount != 10 {
		return nil, billingError("invalid_top_up", "充值金额只能是 $1、$5 或 $10", http.StatusBadRequest, "", nil)
	}
	if idempotencyKey == "" {
		return nil, billingError("invalid_top_up", "缺少 idempotency_key", http.StatusBadRequest, "", nil)
	}
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, billingError("billing_not_ready", "计费账户尚未就绪", http.StatusServiceUnavailable, "", nil)
	}
	op := CreditOperation{
		TenantID: tenantID, IdempotencyKey: idempotencyKey, Amount: amount,
		Kind: "sandbox_topup", Status: "pending",
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&op).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&op, "tenant_id = ? AND idempotency_key = ?", tenantID, idempotencyKey).Error; err != nil {
		return nil, err
	}
	if op.Amount != amount {
		return nil, billingError("subscription_conflict", "同一 idempotency_key 不能用于不同金额", http.StatusConflict, "", nil)
	}
	if op.Status != "succeeded" {
		grant, err := s.client.CreateCreditGrant(ctx, account.MeterForgeCustomerID, idempotencyKey,
			fmt.Sprintf("LoreLattice Sandbox top-up $%.0f", amount), amount, "", nil)
		if err != nil && !isHTTPStatus(err, http.StatusConflict) {
			_ = s.db.WithContext(ctx).Model(&CreditOperation{}).Where("id = ?", op.ID).
				Updates(map[string]any{"status": "failed", "last_error": err.Error(), "updated_at": time.Now().UTC()}).Error
			return nil, billingError("billing_unavailable", "模拟充值失败，请稍后重试", http.StatusServiceUnavailable, "", nil)
		}
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var locked CreditOperation
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, op.ID).Error; err != nil {
				return err
			}
			if locked.Status == "succeeded" {
				return nil
			}
			if err := tx.Model(&Account{}).Where("tenant_id = ?", tenantID).
				UpdateColumn("local_credit_granted", gorm.Expr("local_credit_granted + ?", amount)).Error; err != nil {
				return err
			}
			return tx.Model(&CreditOperation{}).Where("id = ?", locked.ID).Updates(map[string]any{
				"status": "succeeded", "meterforge_grant_id": grant.ID,
				"last_error": "", "updated_at": time.Now().UTC(),
			}).Error
		})
		if err != nil {
			return nil, err
		}
	}
	return s.Overview(ctx, tenantID, true)
}

func (s *Service) ChangeToPro(ctx context.Context, tenantID uint64) (*Overview, error) {
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, billingError("billing_not_ready", "计费账户尚未就绪", http.StatusServiceUnavailable, "", nil)
	}
	if account.PlanKey == PlanPro {
		return nil, billingError("subscription_conflict", "当前已经是 Pro 套餐", http.StatusConflict, "", nil)
	}
	// Trial subscriptions are created with an automatic end-of-trial
	// cancellation. A plan change cannot be applied to a subscription that
	// already has a pending cancellation, so restore it before the immediate
	// Trial -> Pro transition.
	if _, err := s.client.UnscheduleCancel(ctx, account.MeterForgeSubscriptionID); err != nil &&
		!isHTTPStatus(err, http.StatusConflict) {
		return nil, billingError("billing_unavailable", "套餐升级准备失败", http.StatusServiceUnavailable, "", nil)
	}
	subscription, err := s.client.ChangeSubscription(ctx, account.MeterForgeSubscriptionID, account.MeterForgeCustomerKey, PlanPro)
	if err != nil {
		status := http.StatusServiceUnavailable
		code := "billing_unavailable"
		if isHTTPStatus(err, http.StatusConflict) {
			status, code = http.StatusConflict, "subscription_conflict"
		}
		return nil, billingError(code, "套餐升级失败", status, "", nil)
	}
	now, end := time.Now().UTC(), time.Now().UTC().AddDate(0, 1, 0)
	if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{
		"meterforge_subscription_id": subscription.ID, "plan_key": PlanPro,
		"status": "active", "period_started_at": now, "period_ends_at": end,
		"last_synced_at": now, "last_error": "", "updated_at": now,
	}).Error; err != nil {
		return nil, err
	}
	return s.Overview(ctx, tenantID, true)
}

func (s *Service) CancelAtPeriodEnd(ctx context.Context, tenantID uint64) (*Overview, error) {
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, err
	}
	if account.PlanKey != PlanPro {
		return nil, billingError("subscription_conflict", "Trial 试用已经自动安排到期", http.StatusConflict, "", nil)
	}
	if _, err := s.client.CancelSubscription(ctx, account.MeterForgeSubscriptionID, "next_billing_cycle"); err != nil {
		return nil, billingError("billing_unavailable", "取消套餐失败", http.StatusServiceUnavailable, "", nil)
	}
	if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", tenantID).
		Updates(map[string]any{"status": "cancel_scheduled", "updated_at": time.Now().UTC()}).Error; err != nil {
		return nil, err
	}
	return s.Overview(ctx, tenantID, true)
}

func (s *Service) UnscheduleCancel(ctx context.Context, tenantID uint64) (*Overview, error) {
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, err
	}
	if account.Status != "cancel_scheduled" {
		return nil, billingError("subscription_conflict", "当前套餐没有待生效的取消操作", http.StatusConflict, "", nil)
	}
	if _, err := s.client.UnscheduleCancel(ctx, account.MeterForgeSubscriptionID); err != nil {
		return nil, billingError("billing_unavailable", "撤销取消失败", http.StatusServiceUnavailable, "", nil)
	}
	if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", tenantID).
		Updates(map[string]any{"status": "active", "updated_at": time.Now().UTC()}).Error; err != nil {
		return nil, err
	}
	return s.Overview(ctx, tenantID, true)
}

func (s *Service) Invoices(ctx context.Context, tenantID uint64) (json.RawMessage, error) {
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, err
	}
	return s.client.ListInvoices(ctx, account.MeterForgeCustomerID)
}
