package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
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

type AICreditOverview struct {
	GrantedUSD   float64  `json:"granted_usd"`
	UsedUSD      float64  `json:"used_usd"`
	RemainingUSD *float64 `json:"remaining_usd,omitempty"`
	Currency     string   `json:"currency"`
}

type BillingModeOverview struct {
	Mode    BillingMode `json:"mode"`
	Calls   int64       `json:"calls"`
	CostUSD float64     `json:"cost_usd"`
}

type Overview struct {
	Enabled               bool                  `json:"enabled"`
	Status                string                `json:"status"`
	PlanKey               string                `json:"plan_key"`
	PlanName              string                `json:"plan_name"`
	TrialEndsAt           *time.Time            `json:"trial_ends_at,omitempty"`
	PeriodStartedAt       *time.Time            `json:"period_started_at,omitempty"`
	PeriodEndsAt          *time.Time            `json:"period_ends_at,omitempty"`
	Features              []FeatureOverview     `json:"features"`
	CreditBalanceUSD      *float64              `json:"credit_balance_usd,omitempty"`
	AICredits             AICreditOverview      `json:"ai_credits"`
	BillingModes          []BillingModeOverview `json:"billing_modes"`
	CancelScheduled       bool                  `json:"cancel_scheduled"`
	MeterForgeCustomerID  string                `json:"-"`
	MeterForgeCustomerKey string                `json:"-"`
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
		AICredits:             AICreditOverview{GrantedUSD: account.LocalCreditGranted, Currency: "USD"},
	}
	trialExpired := account.PlanKey == PlanTrial && account.TrialEndsAt != nil &&
		!time.Now().UTC().Before(*account.TrialEndsAt)
	if trialExpired {
		result.Status = "expired"
	}
	for _, def := range orderedFeatureDefinitions() {
		limit := def.TrialLimit
		if account.PlanKey == PlanPro {
			limit = def.ProLimit
		}
		var localUsed float64
		query := s.db.WithContext(ctx).Model(&Reservation{}).
			Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_quantity WHEN status = 'completed' THEN actual_quantity ELSE 0 END), 0)`).
			Where("tenant_id = ? AND feature_key = ? AND status IN ?", tenantID, def.Key, []string{"reserved", "completed"})
		if account.PeriodStartedAt != nil {
			query = query.Where("created_at >= ?", *account.PeriodStartedAt)
		}
		if account.PeriodEndsAt != nil {
			query = query.Where("created_at < ?", *account.PeriodEndsAt)
		}
		if err := query.Scan(&localUsed).Error; err != nil {
			return nil, err
		}
		remaining := math.Max(0, limit-localUsed)
		if trialExpired {
			remaining = 0
		}
		result.Features = append(result.Features, FeatureOverview{
			Key: def.Key, Name: def.Name, Unit: def.Unit, Limit: limit,
			Used: localUsed, Remaining: remaining, UnitPrice: def.UnitPrice,
		})
	}
	var usedCost float64
	if err := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost WHEN status = 'completed' THEN actual_cost ELSE 0 END), 0)`).
		Where("tenant_id = ? AND chargeable = ? AND status IN ?", tenantID, true, []string{"reserved", "completed"}).
		Scan(&usedCost).Error; err != nil {
		return nil, err
	}
	result.AICredits.UsedUSD = usedCost
	var modes []struct {
		Mode  string
		Calls int64
		Cost  float64
	}
	modeQuery := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`billing_mode AS mode, COUNT(*) AS calls, COALESCE(SUM(CASE WHEN status = 'completed' THEN actual_cost ELSE 0 END), 0) AS cost`).
		Where("tenant_id = ?", tenantID)
	if account.PeriodStartedAt != nil {
		modeQuery = modeQuery.Where("created_at >= ?", *account.PeriodStartedAt)
	}
	if err := modeQuery.Group("billing_mode").Scan(&modes).Error; err != nil {
		return nil, err
	}
	for _, row := range modes {
		result.BillingModes = append(result.BillingModes, BillingModeOverview{
			Mode: BillingMode(row.Mode), Calls: row.Calls, CostUSD: row.Cost,
		})
	}
	if includeFinancial {
		balance, err := s.creditBalance(ctx, account.MeterForgeCustomerID)
		if err != nil {
			return nil, billingError("billing_unavailable", "计费服务暂时不可用", http.StatusServiceUnavailable, "", account.PeriodEndsAt)
		}
		localBalance, err := s.localAvailableCredit(ctx, tenantID, account.LocalCreditGranted)
		if err != nil {
			return nil, err
		}
		balance = math.Min(balance, localBalance)
		result.CreditBalanceUSD = &balance
		result.AICredits.RemainingUSD = &balance
	}
	return result, nil
}

func (s *Service) localAvailableCredit(ctx context.Context, tenantID uint64, granted float64) (float64, error) {
	var localCost float64
	if err := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost WHEN status = 'completed' THEN actual_cost ELSE 0 END), 0)`).
		Where("tenant_id = ? AND chargeable = ? AND status IN ?", tenantID, true, []string{"reserved", "completed"}).
		Scan(&localCost).Error; err != nil {
		return 0, err
	}
	return math.Max(0, granted-localCost), nil
}

type UsageRow struct {
	ID           string     `json:"id"`
	FeatureKey   string     `json:"feature_key"`
	ModelName    string     `json:"model"`
	Provider     string     `json:"provider"`
	Operation    string     `json:"operation"`
	JobID        string     `json:"job_id"`
	Category     string     `json:"business_category"`
	BillingMode  string     `json:"billing_mode"`
	Chargeable   bool       `json:"chargeable"`
	PriceVersion string     `json:"price_version"`
	Quantity     *float64   `json:"quantity"`
	Unit         string     `json:"unit"`
	CostUSD      *float64   `json:"cost_usd"`
	Estimated    bool       `json:"estimated"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
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
		Select(`id, feature_key, model_name, provider, operation, job_id, business_category,
		        billing_mode, chargeable, price_version, actual_quantity AS quantity,
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

type UsageJobRow struct {
	ID               string     `json:"id"`
	BusinessCategory string     `json:"business_category"`
	BillingMode      string     `json:"billing_mode"`
	CallCount        int        `json:"call_count"`
	ChargeableCalls  int        `json:"chargeable_calls"`
	CostUSD          float64    `json:"cost_usd"`
	Estimated        bool       `json:"estimated"`
	Status           string     `json:"status"`
	Models           []string   `json:"models"`
	Providers        []string   `json:"providers"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// UsageJobs folds provider-level reservations into one user-visible action.
// Raw rows remain available through Usage for audits and model diagnostics.
func (s *Service) UsageJobs(ctx context.Context, tenantID uint64, from, to *time.Time, limit int) ([]UsageJobRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.Usage(ctx, tenantID, from, to, "", "", "", 500)
	if err != nil {
		return nil, err
	}
	type accumulator struct {
		row       UsageJobRow
		models    map[string]struct{}
		providers map[string]struct{}
		completed int
	}
	ordered := make([]string, 0, limit)
	jobs := make(map[string]*accumulator)
	for _, call := range rows {
		jobID := call.JobID
		if jobID == "" {
			jobID = call.ID
		}
		job := jobs[jobID]
		if job == nil {
			if len(ordered) >= limit {
				continue
			}
			job = &accumulator{
				row: UsageJobRow{ID: jobID, BusinessCategory: call.Category, BillingMode: call.BillingMode,
					Estimated: call.Estimated, Status: call.Status, StartedAt: call.CreatedAt},
				models: map[string]struct{}{}, providers: map[string]struct{}{},
			}
			jobs[jobID] = job
			ordered = append(ordered, jobID)
		}
		job.row.CallCount++
		if call.Chargeable {
			job.row.ChargeableCalls++
		}
		if call.CostUSD != nil {
			job.row.CostUSD += *call.CostUSD
		}
		job.row.Estimated = job.row.Estimated || call.Estimated
		if call.BillingMode != "" && job.row.BillingMode != call.BillingMode {
			job.row.BillingMode = "mixed"
		}
		if categoryPriority(call.Category) > categoryPriority(job.row.BusinessCategory) {
			job.row.BusinessCategory = call.Category
		}
		if call.CreatedAt.Before(job.row.StartedAt) {
			job.row.StartedAt = call.CreatedAt
		}
		if call.Status == "completed" {
			job.completed++
			if call.CompletedAt != nil && (job.row.CompletedAt == nil || call.CompletedAt.After(*job.row.CompletedAt)) {
				completed := *call.CompletedAt
				job.row.CompletedAt = &completed
			}
		}
		if call.ModelName != "" {
			job.models[call.ModelName] = struct{}{}
		}
		if call.Provider != "" {
			job.providers[call.Provider] = struct{}{}
		}
	}
	result := make([]UsageJobRow, 0, len(ordered))
	for _, id := range ordered {
		job := jobs[id]
		if job.completed == job.row.CallCount {
			job.row.Status = "completed"
		} else if job.completed > 0 {
			job.row.Status = "partial"
		}
		for value := range job.models {
			job.row.Models = append(job.row.Models, value)
		}
		for value := range job.providers {
			job.row.Providers = append(job.row.Providers, value)
		}
		sort.Strings(job.row.Models)
		sort.Strings(job.row.Providers)
		result = append(result, job.row)
	}
	return result, nil
}

func categoryPriority(category string) int {
	switch category {
	case "chat_agent":
		return 4
	case "image_processing":
		return 3
	case "audio_transcription":
		return 2
	case "document_indexing":
		return 1
	default:
		return 0
	}
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
		s.invalidateAccessCache()
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
