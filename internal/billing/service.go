package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Pototoooo/lorelattice/internal/logger"
	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const trialCredit = 0.36

type Service struct {
	db           *gorm.DB
	config       Config
	client       *Client
	catalogMu    sync.Mutex
	catalogOK    bool
	features     map[FeatureKey]featureResource
	plans        map[string]planResource
	kick         chan struct{}
	pricing      *PriceCatalog
	accessMu     sync.Mutex
	entitlements map[string]cachedEntitlement
	credits      map[string]cachedCredit
}

type cachedEntitlement struct {
	value     entitlementValue
	expiresAt time.Time
}

type cachedCredit struct {
	value     float64
	expiresAt time.Time
}

func NewService(db *gorm.DB) *Service {
	cfg := LoadConfigFromEnv()
	pricing, err := NewPriceCatalog(cfg.ModelPricingJSON)
	if err != nil {
		logger.Warnf(context.Background(), "[Billing] %v; using built-in prices", err)
		pricing, _ = NewPriceCatalog("")
	}
	return &Service{
		db: db, config: cfg, client: NewClient(cfg),
		features:     make(map[FeatureKey]featureResource),
		plans:        make(map[string]planResource),
		kick:         make(chan struct{}, 1),
		pricing:      pricing,
		entitlements: make(map[string]cachedEntitlement),
		credits:      make(map[string]cachedCredit),
	}
}

func (s *Service) entitlementValue(ctx context.Context, customerKey string, feature FeatureKey) (entitlementValue, error) {
	key := customerKey + "\x00" + string(feature)
	now := time.Now()
	s.accessMu.Lock()
	if cached, ok := s.entitlements[key]; ok && now.Before(cached.expiresAt) {
		s.accessMu.Unlock()
		return cached.value, nil
	}
	s.accessMu.Unlock()
	value, err := s.client.EntitlementValue(ctx, customerKey, feature)
	if err == nil && s.config.AccessCacheTTL > 0 {
		s.accessMu.Lock()
		s.entitlements[key] = cachedEntitlement{value: value, expiresAt: now.Add(s.config.AccessCacheTTL)}
		s.accessMu.Unlock()
	}
	return value, err
}

func (s *Service) creditBalance(ctx context.Context, customerID string) (float64, error) {
	now := time.Now()
	s.accessMu.Lock()
	if cached, ok := s.credits[customerID]; ok && now.Before(cached.expiresAt) {
		s.accessMu.Unlock()
		return cached.value, nil
	}
	s.accessMu.Unlock()
	value, err := s.client.CreditBalance(ctx, customerID)
	if err == nil && s.config.AccessCacheTTL > 0 {
		s.accessMu.Lock()
		s.credits[customerID] = cachedCredit{value: value, expiresAt: now.Add(s.config.AccessCacheTTL)}
		s.accessMu.Unlock()
	}
	return value, err
}

func (s *Service) invalidateAccessCache() {
	s.accessMu.Lock()
	defer s.accessMu.Unlock()
	clear(s.entitlements)
	clear(s.credits)
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
		logger.Infof(context.Background(), "[Billing] MeterForge integration disabled")
		return
	}
	go s.runBootstrapAndBackfill()
	go s.runOutbox()
}

func (s *Service) runBootstrapAndBackfill() {
	ctx := context.Background()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if err := s.provisioningPass(ctx); err != nil {
			logger.Errorf(ctx, "[Billing] provisioning pass failed: %v", err)
		}
		<-ticker.C
	}
}

func (s *Service) provisioningPass(ctx context.Context) error {
	if err := s.BootstrapCatalog(ctx); err != nil {
		return fmt.Errorf("catalog bootstrap: %w", err)
	}
	var tenants []types.Tenant
	if err := s.db.WithContext(ctx).Find(&tenants).Error; err != nil {
		return fmt.Errorf("tenant backfill list: %w", err)
	}
	for _, tenant := range tenants {
		if err := s.ProvisionTenant(ctx, tenant.ID, tenant.Name); err != nil {
			logger.Errorf(ctx, "[Billing] tenant %d backfill failed: %v", tenant.ID, err)
		}
	}
	if err := s.ReconcileAccounts(ctx); err != nil {
		return fmt.Errorf("billing reconciliation: %w", err)
	}
	return nil
}

func (s *Service) BootstrapCatalog(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	if s.catalogOK {
		return nil
	}

	meters, err := s.client.ListMeters(ctx)
	if err != nil {
		return fmt.Errorf("list meters: %w", err)
	}
	meterBySlug := make(map[string]meterResource, len(meters))
	for _, meter := range meters {
		meterBySlug[meter.Slug] = meter
	}
	for _, def := range orderedFeatureDefinitions() {
		if _, ok := meterBySlug[def.MeterSlug]; !ok {
			meter, err := s.client.CreateMeter(ctx, def)
			if err != nil {
				if !isHTTPStatus(err, http.StatusConflict) {
					return fmt.Errorf("create meter %s: %w", def.MeterSlug, err)
				}
				meters, err = s.client.ListMeters(ctx)
				if err != nil {
					return err
				}
				for _, current := range meters {
					meterBySlug[current.Slug] = current
				}
			} else {
				meterBySlug[def.MeterSlug] = meter
			}
		}
	}

	features, err := s.client.ListFeatures(ctx)
	if err != nil {
		return fmt.Errorf("list features: %w", err)
	}
	featureByKey := make(map[string]featureResource, len(features))
	for _, feature := range features {
		featureByKey[feature.Key] = feature
	}
	for _, def := range orderedFeatureDefinitions() {
		feature, ok := featureByKey[string(def.Key)]
		if !ok {
			feature, err = s.client.CreateFeature(ctx, def, meterBySlug[def.MeterSlug].ID)
			if err != nil {
				if !isHTTPStatus(err, http.StatusConflict) {
					return fmt.Errorf("create feature %s: %w", def.Key, err)
				}
				refreshed, listErr := s.client.ListFeatures(ctx)
				if listErr != nil {
					return listErr
				}
				for _, current := range refreshed {
					featureByKey[current.Key] = current
				}
				feature = featureByKey[string(def.Key)]
			}
		}
		if feature.ID == "" {
			return fmt.Errorf("feature %s has no MeterForge ID", def.Key)
		}
		s.features[def.Key] = feature
	}

	plans, err := s.client.ListPlans(ctx)
	if err != nil {
		return fmt.Errorf("list plans: %w", err)
	}
	activePlan := func(key string) (planResource, bool) {
		var latest planResource
		found := false
		for _, plan := range plans {
			if plan.Key == key && plan.Status == "active" && (!found || plan.Version > latest.Version) {
				latest = plan
				found = true
			}
		}
		return latest, found
	}
	createPlan := func(key, name string, pro bool) (planResource, error) {
		limits := make(map[FeatureKey]float64)
		for _, def := range orderedFeatureDefinitions() {
			if pro {
				limits[def.Key] = def.ProLimit
			} else {
				limits[def.Key] = def.TrialLimit
			}
		}
		plan, err := s.client.CreatePlan(ctx, key, name, limits, s.features)
		if err != nil {
			return plan, err
		}
		if err := s.client.PublishPlan(ctx, plan.ID); err != nil {
			return plan, err
		}
		plan.Status = "active"
		return plan, nil
	}
	for _, spec := range []struct {
		Key, Name string
		Pro       bool
	}{{PlanTrial, "LoreLattice Trial", false}, {PlanPro, "LoreLattice Pro", true}} {
		plan, ok := activePlan(spec.Key)
		if !ok {
			plan, err = createPlan(spec.Key, spec.Name, spec.Pro)
			if err != nil {
				return fmt.Errorf("create plan %s: %w", spec.Key, err)
			}
		}
		s.plans[spec.Key] = plan
	}
	s.catalogOK = true
	logger.Infof(ctx, "[Billing] LoreLattice MeterForge catalog is ready")
	return nil
}

func (s *Service) ProvisionTenantAsync(tenantID uint64, name string) {
	if !s.Enabled() || tenantID == 0 {
		return
	}
	go func() {
		if err := s.ProvisionTenant(context.Background(), tenantID, name); err != nil {
			logger.Errorf(context.Background(), "[Billing] tenant %d provisioning failed: %v", tenantID, err)
		}
	}()
}

func (s *Service) ProvisionTenant(ctx context.Context, tenantID uint64, name string) error {
	if !s.Enabled() {
		return nil
	}
	if err := s.BootstrapCatalog(ctx); err != nil {
		return err
	}
	customerKey := fmt.Sprintf("lorelattice_tenant_%d", tenantID)
	now := time.Now().UTC()
	account := Account{TenantID: tenantID, MeterForgeCustomerKey: customerKey, Status: "pending"}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"meterforge_customer_key": customerKey, "updated_at": now,
		}),
	}).Create(&account).Error; err != nil {
		return fmt.Errorf("upsert billing account: %w", err)
	}
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return err
	}
	isProvisionedStatus := account.Status == "active" ||
		account.Status == "cancel_scheduled" ||
		account.Status == "reconciling_active" ||
		account.Status == "reconciling_cancel_scheduled"
	if account.PlanKey == PlanPro && account.MeterForgeSubscriptionID != "" && isProvisionedStatus {
		return nil
	}
	if account.PlanKey == PlanTrial && account.MeterForgeCustomerID != "" &&
		account.MeterForgeSubscriptionID != "" && account.TrialStartedAt != nil && isProvisionedStatus {
		return nil
	}

	customer, err := s.client.FindCustomerByKey(ctx, customerKey)
	if err != nil {
		return s.markProvisionError(ctx, tenantID, err)
	}
	if customer == nil {
		created, createErr := s.client.CreateCustomer(ctx, customerKey,
			fmt.Sprintf("LoreLattice Workspace %d — %s", tenantID, name),
			s.config.SubjectPrefix+fmt.Sprint(tenantID))
		if createErr != nil {
			return s.markProvisionError(ctx, tenantID, createErr)
		}
		customer = &created
	}

	trialStart := now
	trialEnd := trialStart.Add(30 * 24 * time.Hour)
	subscriptionID := account.MeterForgeSubscriptionID
	needsTrialCancel := account.TrialStartedAt == nil
	if subscriptionID == "" {
		subscriptions, listErr := s.client.ListSubscriptions(ctx, customer.ID)
		if listErr != nil {
			return s.markProvisionError(ctx, tenantID, listErr)
		}
		trialPlan := s.plans[PlanTrial]
		for _, current := range subscriptions {
			if current.PlanID == trialPlan.ID || (trialPlan.PlanID != "" && current.PlanID == trialPlan.PlanID) {
				subscriptionID = current.ID
				break
			}
		}
		if subscriptionID == "" {
			subscription, createErr := s.client.CreateSubscription(ctx, customerKey, PlanTrial)
			if createErr != nil {
				return s.markProvisionError(ctx, tenantID, createErr)
			}
			subscriptionID = subscription.ID
		}
	}
	// MeterForge aligned subscriptions only accept an aligned cancel timing.
	// The Trial plan cadence is P1M, so the next billing boundary is the
	// authoritative remote expiry while LoreLattice also displays the fixed
	// 30-day product deadline from trialEnd.
	if needsTrialCancel {
		if _, cancelErr := s.client.CancelSubscription(ctx, subscriptionID, "next_billing_cycle"); cancelErr != nil {
			return s.markProvisionError(ctx, tenantID, cancelErr)
		}
	}

	grantKey := fmt.Sprintf("lorelattice_trial_%d", tenantID)
	featureKeys := []string{
		string(FeatureLLMTokens), string(FeatureEmbeddingTokens),
		string(FeatureRerankTokens), string(FeatureASRSeconds),
	}
	grant, grantErr := s.client.CreateCreditGrant(ctx, customer.ID, grantKey,
		"LoreLattice 30-day trial credit", trialCredit, "P30D", featureKeys)
	if grantErr != nil && !isHTTPStatus(grantErr, http.StatusConflict) {
		return s.markProvisionError(ctx, tenantID, grantErr)
	}
	_ = grant

	synced := time.Now().UTC()
	updates := map[string]any{
		"meterforge_customer_id": customer.ID, "meterforge_subscription_id": subscriptionID,
		"plan_key": PlanTrial, "status": "active", "trial_started_at": trialStart,
		"trial_ends_at": trialEnd, "period_started_at": trialStart, "period_ends_at": trialEnd,
		"local_credit_granted": trialCredit, "last_synced_at": synced,
		"last_error": "", "updated_at": synced,
	}
	if account.TrialStartedAt != nil {
		delete(updates, "trial_started_at")
		delete(updates, "trial_ends_at")
		delete(updates, "period_started_at")
		delete(updates, "period_ends_at")
		// Preserve all previously granted Sandbox top-ups. Re-provisioning is
		// idempotent and must never reset the local prepaid ledger to the
		// original Trial promotional credit.
		delete(updates, "local_credit_granted")
	}
	if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", tenantID).Updates(updates).Error; err != nil {
		return err
	}
	logger.Infof(ctx, "[Billing] tenant %d provisioned customer=%s subscription=%s", tenantID, customer.ID, subscriptionID)
	return nil
}

func (s *Service) markProvisionError(ctx context.Context, tenantID uint64, err error) error {
	_ = s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", tenantID).
		Updates(map[string]any{"status": "error", "last_error": err.Error(), "updated_at": time.Now().UTC()}).Error
	return err
}

func (s *Service) CheckAccess(ctx context.Context, tenantID uint64, feature FeatureKey, quantity float64) error {
	if !s.Enabled() {
		return nil
	}
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return billingError("billing_not_ready", "计费账户尚未开通，请稍后重试", http.StatusServiceUnavailable, feature, nil)
	}
	if account.Status != "active" && account.Status != "cancel_scheduled" {
		return billingError("billing_not_ready", "计费账户尚未就绪，请稍后重试", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
	}
	if account.PlanKey == PlanTrial && account.TrialEndsAt != nil && !time.Now().UTC().Before(*account.TrialEndsAt) {
		return billingError("token_limit_reached", "Trial 试用已到期，请升级 Pro", http.StatusPaymentRequired, feature, account.TrialEndsAt)
	}
	entitlement, err := s.entitlementValue(ctx, account.MeterForgeCustomerKey, feature)
	if err != nil {
		return billingError("billing_unavailable", "计费服务暂时不可用，请稍后重试", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
	}
	if !entitlement.HasAccess || entitlement.Balance+1e-9 < quantity {
		return billingError("token_limit_reached", "本周期用量额度不足", http.StatusPaymentRequired, feature, account.PeriodEndsAt)
	}
	credit, err := s.creditBalance(ctx, account.MeterForgeCustomerID)
	if err != nil {
		return billingError("billing_unavailable", "计费服务暂时不可用，请稍后重试", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
	}
	localCredit, err := s.localAvailableCredit(ctx, tenantID, account.LocalCreditGranted)
	if err != nil {
		return err
	}
	cost := quantity * FeatureDefinitions[feature].UnitPrice
	if math.Min(credit, localCredit)+1e-9 < cost {
		return billingError("insufficient_credit", "预付余额不足，请充值后继续使用", http.StatusPaymentRequired, feature, account.PeriodEndsAt)
	}
	return nil
}

func (s *Service) Reserve(ctx context.Context, tenantID uint64, feature FeatureKey, quantity float64, meta UsageMetadata) (*Reservation, error) {
	if !s.Enabled() {
		return nil, nil
	}
	def, ok := FeatureDefinitions[feature]
	if !ok || quantity <= 0 {
		return nil, fmt.Errorf("invalid billing reservation")
	}
	mode := meta.Mode
	if !mode.Valid() {
		// Backward-compatible default for callers compiled before billing modes.
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
	unitPrice, priceVersion := s.pricing.Resolve(feature, meta.Provider, meta.ModelName, mode)

	// BYOK and local models must never be blocked by LoreLattice's prepaid
	// wallet. They still create a durable local usage row for analytics.
	if !mode.EnforcesQuota() {
		now := time.Now().UTC()
		reservation := &Reservation{
			ID: uuid.NewString(), TenantID: tenantID, FeatureKey: string(feature),
			EventType: def.EventType, MeterUnit: def.Unit,
			ModelID: meta.ModelID, ModelName: meta.ModelName, Provider: meta.Provider,
			Operation: meta.Operation, RequestID: meta.RequestID, JobID: meta.JobID,
			BusinessCategory: meta.Category, BillingMode: string(mode),
			Chargeable: false, PriceVersion: priceVersion,
			ReservedQuantity: quantity, UnitPrice: unitPrice, ReservedCost: 0,
			Estimated: meta.Estimated, Status: "reserved", PeriodStartedAt: now,
		}
		if err := s.db.WithContext(ctx).Create(reservation).Error; err != nil {
			return nil, err
		}
		return reservation, nil
	}
	var result *Reservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&account, "tenant_id = ?", tenantID).Error; err != nil {
			return billingError("billing_not_ready", "计费账户尚未开通，请稍后重试", http.StatusServiceUnavailable, feature, nil)
		}
		if account.Status != "active" && account.Status != "cancel_scheduled" {
			return billingError("billing_not_ready", "计费账户尚未就绪", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
		}
		if account.PlanKey == PlanTrial && account.TrialEndsAt != nil && !time.Now().UTC().Before(*account.TrialEndsAt) {
			return billingError("token_limit_reached", "Trial 试用已到期，请升级 Pro", http.StatusPaymentRequired, feature, account.TrialEndsAt)
		}
		s.advancePeriod(&account)
		if err := tx.Model(&Account{}).Where("tenant_id = ?", tenantID).Updates(map[string]any{
			"period_started_at": account.PeriodStartedAt, "period_ends_at": account.PeriodEndsAt,
		}).Error; err != nil {
			return err
		}

		entitlement, err := s.entitlementValue(ctx, account.MeterForgeCustomerKey, feature)
		if err != nil {
			return billingError("billing_unavailable", "计费服务暂时不可用，请稍后重试", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
		}
		remoteCredit := math.MaxFloat64
		if mode.Chargeable() {
			remoteCredit, err = s.creditBalance(ctx, account.MeterForgeCustomerID)
			if err != nil {
				return billingError("billing_unavailable", "计费服务暂时不可用，请稍后重试", http.StatusServiceUnavailable, feature, account.PeriodEndsAt)
			}
		}
		if !entitlement.HasAccess {
			return billingError("token_limit_reached", "本周期用量额度已耗尽", http.StatusPaymentRequired, feature, account.PeriodEndsAt)
		}

		var localQuantity float64
		if err := tx.Model(&Reservation{}).
			Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_quantity WHEN status = 'completed' THEN actual_quantity ELSE 0 END), 0)`).
			Where("tenant_id = ? AND feature_key = ? AND period_started_at = ? AND billing_mode IN ? AND status IN ?",
				tenantID, feature, *account.PeriodStartedAt,
				[]string{string(BillingModePlatform), string(BillingModeIncluded)}, []string{"reserved", "completed"}).
			Scan(&localQuantity).Error; err != nil {
			return err
		}
		var localCost float64
		if err := tx.Model(&Reservation{}).
			Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost WHEN status = 'completed' THEN actual_cost ELSE 0 END), 0)`).
			Where("tenant_id = ? AND chargeable = ? AND status IN ?", tenantID, true, []string{"reserved", "completed"}).
			Scan(&localCost).Error; err != nil {
			return err
		}
		limit := def.TrialLimit
		if account.PlanKey == PlanPro {
			limit = def.ProLimit
		}
		if quantity > math.Min(math.Max(0, limit-localQuantity), entitlement.Balance)+1e-9 {
			return billingError("token_limit_reached", "本周期用量额度不足", http.StatusPaymentRequired, feature, account.PeriodEndsAt)
		}
		cost := quantity * unitPrice
		localCredit := math.Max(0, account.LocalCreditGranted-localCost)
		if mode.Chargeable() && cost > math.Min(localCredit, remoteCredit)+1e-9 {
			return billingError("insufficient_credit", "预付余额不足，请充值后继续使用", http.StatusPaymentRequired, feature, account.PeriodEndsAt)
		}
		reservation := &Reservation{
			ID: uuid.NewString(), TenantID: tenantID, FeatureKey: string(feature),
			EventType: def.EventType, MeterUnit: def.Unit,
			ModelID: meta.ModelID, ModelName: meta.ModelName, Provider: meta.Provider,
			Operation: meta.Operation, RequestID: meta.RequestID, JobID: meta.JobID,
			BusinessCategory: meta.Category, BillingMode: string(mode),
			Chargeable: mode.Chargeable(), PriceVersion: priceVersion,
			ReservedQuantity: quantity, UnitPrice: unitPrice,
			ReservedCost: cost, Estimated: meta.Estimated,
			Status: "reserved", PeriodStartedAt: *account.PeriodStartedAt,
		}
		if err := tx.Create(reservation).Error; err != nil {
			return err
		}
		result = reservation
		return nil
	})
	return result, err
}

func (s *Service) advancePeriod(account *Account) {
	now := time.Now().UTC()
	if account.PeriodStartedAt == nil || account.PeriodEndsAt == nil {
		start, end := now, now.AddDate(0, 1, 0)
		account.PeriodStartedAt, account.PeriodEndsAt = &start, &end
		return
	}
	if account.PlanKey != PlanPro {
		return
	}
	start, end := *account.PeriodStartedAt, *account.PeriodEndsAt
	for !now.Before(end) {
		start = end
		end = end.AddDate(0, 1, 0)
	}
	account.PeriodStartedAt, account.PeriodEndsAt = &start, &end
}

func (s *Service) Complete(ctx context.Context, reservation *Reservation, actual float64, estimated bool) error {
	if !s.Enabled() || reservation == nil {
		return nil
	}
	if actual <= 0 {
		return s.Release(ctx, reservation, "provider returned no billable usage")
	}
	now := time.Now().UTC()
	actualCost := actual * reservation.UnitPrice
	event := map[string]any{
		"specversion": "1.0", "type": reservation.EventType, "id": reservation.ID,
		"time": now, "source": s.config.Source,
		"subject": s.config.SubjectPrefix + fmt.Sprint(reservation.TenantID),
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
		result := tx.Model(&Reservation{}).Where("id = ? AND status = 'reserved'", reservation.ID).
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
		// BYOK/local/included usage is intentionally kept in LoreLattice's
		// analytics ledger only. Sending it through a priced MeterForge rate
		// card would incorrectly debit the platform wallet.
		if !reservation.Chargeable {
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
	return s.db.WithContext(ctx).Model(&Reservation{}).
		Where("id = ? AND status = 'reserved'", reservation.ID).
		Updates(map[string]any{"status": "released", "error_message": reason, "updated_at": time.Now().UTC()}).Error
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
			logger.Warnf(context.Background(), "[Billing] outbox flush failed: %v", err)
		}
	}
}

func (s *Service) FlushOutbox(ctx context.Context, limit int) error {
	var rows []OutboxRow
	if err := s.db.WithContext(ctx).
		Where("status IN ? AND next_attempt_at <= ?", []string{"pending", "failed"}, time.Now().UTC()).
		Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		err := s.client.SendEvent(ctx, row.EventID, row.Payload)
		now := time.Now().UTC()
		if err == nil {
			_ = s.db.WithContext(ctx).Model(&OutboxRow{}).Where("id = ?", row.ID).
				Updates(map[string]any{"status": "sent", "sent_at": now, "last_error": "", "updated_at": now}).Error
			s.invalidateAccessCache()
			continue
		}
		attempts := row.Attempts + 1
		delay := time.Duration(1<<minInt(attempts, 6)) * time.Second
		_ = s.db.WithContext(ctx).Model(&OutboxRow{}).Where("id = ?", row.ID).
			Updates(map[string]any{
				"status": "failed", "attempts": attempts, "last_error": err.Error(),
				"next_attempt_at": now.Add(delay), "updated_at": now,
			}).Error
	}
	return nil
}

// ReconcileAccounts compares the durable local ledger with MeterForge's
// current-period entitlement usage. A local-ahead mismatch is repaired by
// replaying the stable Event IDs from the outbox. A remote-ahead mismatch is
// fail-closed because deleting an already accepted usage fact would be unsafe;
// the account remains in a reconciling state for operator review.
func (s *Service) ReconcileAccounts(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	var accounts []Account
	if err := s.db.WithContext(ctx).Where("status IN ?", []string{
		"active", "cancel_scheduled", "reconciling_active", "reconciling_cancel_scheduled",
	}).Find(&accounts).Error; err != nil {
		return err
	}
	for _, account := range accounts {
		var pending int64
		if err := s.db.WithContext(ctx).Model(&OutboxRow{}).
			Where("tenant_id = ? AND status IN ?", account.TenantID, []string{"pending", "failed"}).
			Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 || account.PeriodStartedAt == nil {
			continue
		}

		mismatched := false
		repairQueued := false
		remoteUnavailable := false
		var mismatchDetails string
		for _, def := range orderedFeatureDefinitions() {
			remote, err := s.entitlementValue(ctx, account.MeterForgeCustomerKey, def.Key)
			if err != nil {
				// Availability failures are handled fail-closed on every access
				// check. Do not turn a transient outage into a ledger mismatch.
				remoteUnavailable = true
				continue
			}
			var local float64
			if err := s.db.WithContext(ctx).Model(&Reservation{}).
				Select("COALESCE(SUM(actual_quantity), 0)").
				Where("tenant_id = ? AND feature_key = ? AND period_started_at = ? AND status = 'completed'",
					account.TenantID, def.Key, *account.PeriodStartedAt).
				Scan(&local).Error; err != nil {
				return err
			}
			delta := local - remote.Usage
			if math.Abs(delta) <= 1e-6 {
				continue
			}
			mismatched = true
			mismatchDetails += fmt.Sprintf("%s local=%.6f remote=%.6f; ", def.Key, local, remote.Usage)
			if delta > 0 {
				result := s.db.WithContext(ctx).Exec(`
					UPDATE billing_outbox AS o
					SET status = 'pending', attempts = 0, next_attempt_at = NOW(),
					    sent_at = NULL, last_error = '', updated_at = NOW()
					FROM billing_reservations AS r
					WHERE o.reservation_id = r.id
					  AND r.tenant_id = ?
					  AND r.feature_key = ?
					  AND r.period_started_at = ?
					  AND r.status = 'completed'
					  AND o.status = 'sent'`,
					account.TenantID, def.Key, *account.PeriodStartedAt)
				if result.Error != nil {
					return result.Error
				}
				repairQueued = repairQueued || result.RowsAffected > 0
			}
		}
		if remoteUnavailable {
			continue
		}

		if mismatched {
			status := "reconciling_active"
			if account.Status == "cancel_scheduled" || account.Status == "reconciling_cancel_scheduled" {
				status = "reconciling_cancel_scheduled"
			}
			if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", account.TenantID).
				Updates(map[string]any{
					"status": status, "last_error": "usage reconciliation: " + mismatchDetails,
					"updated_at": time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
			if repairQueued {
				select {
				case s.kick <- struct{}{}:
				default:
				}
			}
			continue
		}

		if strings.HasPrefix(account.Status, "reconciling_") {
			status := "active"
			if account.Status == "reconciling_cancel_scheduled" {
				status = "cancel_scheduled"
			}
			now := time.Now().UTC()
			if err := s.db.WithContext(ctx).Model(&Account{}).Where("tenant_id = ?", account.TenantID).
				Updates(map[string]any{
					"status": status, "last_error": "", "last_synced_at": now, "updated_at": now,
				}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func billingError(code, message string, status int, feature FeatureKey, resetAt *time.Time) *BillingError {
	return &BillingError{Code: code, Message: message, HTTPStatus: status, Feature: feature, ResetAt: resetAt}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
