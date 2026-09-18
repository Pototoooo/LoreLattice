package billing

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

type AICreditOverview struct {
	GrantedUSD   float64  `json:"granted_usd"`
	UsedUSD      float64  `json:"used_usd"`
	RemainingUSD *float64 `json:"remaining_usd,omitempty"`
	Currency     string   `json:"currency"`
}

type Overview struct {
	Enabled         bool             `json:"enabled"`
	Status          string           `json:"status"`
	FinancialStatus string           `json:"financial_status"`
	CNYPerUSD       float64          `json:"cny_per_usd"`
	PaymentEnabled  bool             `json:"payment_enabled"`
	TopUpAmountsFen []int64          `json:"top_up_amounts_fen"`
	AICredits       AICreditOverview `json:"ai_credits"`
}

func (s *Service) Overview(ctx context.Context, tenantID uint64, includeFinancial bool) (*Overview, error) {
	if !s.Enabled() {
		return &Overview{Enabled: false, Status: "disabled"}, nil
	}
	if err := s.ProvisionTenant(ctx, tenantID, ""); err != nil {
		return nil, err
	}
	var account Account
	if err := s.db.WithContext(ctx).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
		return nil, err
	}
	settings, err := s.walletSettings(ctx)
	if err != nil {
		return nil, err
	}
	result := &Overview{Enabled: true, Status: "active", FinancialStatus: "local",
		CNYPerUSD:      settings.CNYPerUSD,
		PaymentEnabled: s.payment != nil, TopUpAmountsFen: []int64{1000, 5000, 10000},
		AICredits: AICreditOverview{Currency: "USD"},
	}
	if account.Status == "suspended" {
		result.Status = "suspended"
	}
	if includeFinancial {
		balance, err := s.localAvailableCredit(ctx, tenantID, account.LocalCreditGranted)
		if err != nil {
			return nil, err
		}
		result.AICredits.GrantedUSD = account.LocalCreditGranted
		result.AICredits.RemainingUSD = &balance
		result.AICredits.UsedUSD = account.LocalCreditGranted - balance
	}
	return result, nil
}

func (s *Service) localAvailableCredit(ctx context.Context, tenantID uint64, granted float64) (float64, error) {
	var localCost float64
	if err := s.db.WithContext(ctx).Model(&Reservation{}).
		Select(`COALESCE(SUM(CASE WHEN status = 'reserved' THEN reserved_cost WHEN status IN ('completed', 'recovered') THEN actual_cost ELSE 0 END), 0)`).
		Where("tenant_id = ? AND chargeable = ? AND status IN ?", tenantID, true, []string{"reserved", "completed", "recovered"}).
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
