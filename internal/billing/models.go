package billing

import (
	"encoding/json"
	"time"
)

type Account struct {
	TenantID                 uint64     `gorm:"primaryKey;column:tenant_id"`
	MeterForgeCustomerID     string     `gorm:"column:meterforge_customer_id"`
	MeterForgeCustomerKey    string     `gorm:"column:meterforge_customer_key"`
	MeterForgeSubscriptionID string     `gorm:"column:meterforge_subscription_id"`
	PlanKey                  string     `gorm:"column:plan_key"`
	Status                   string     `gorm:"column:status"`
	TrialStartedAt           *time.Time `gorm:"column:trial_started_at"`
	TrialEndsAt              *time.Time `gorm:"column:trial_ends_at"`
	PeriodStartedAt          *time.Time `gorm:"column:period_started_at"`
	PeriodEndsAt             *time.Time `gorm:"column:period_ends_at"`
	LocalCreditGranted       float64    `gorm:"column:local_credit_granted"`
	LastSyncedAt             *time.Time `gorm:"column:last_synced_at"`
	LastError                string     `gorm:"column:last_error"`
	CreatedAt                time.Time  `gorm:"column:created_at"`
	UpdatedAt                time.Time  `gorm:"column:updated_at"`
}

func (Account) TableName() string { return "billing_accounts" }

type Reservation struct {
	ID               string     `gorm:"primaryKey;column:id"`
	TenantID         uint64     `gorm:"column:tenant_id"`
	FeatureKey       string     `gorm:"column:feature_key"`
	EventType        string     `gorm:"column:event_type"`
	MeterUnit        string     `gorm:"column:meter_unit"`
	ModelID          string     `gorm:"column:model_id"`
	ModelName        string     `gorm:"column:model_name"`
	Provider         string     `gorm:"column:provider"`
	Operation        string     `gorm:"column:operation"`
	RequestID        string     `gorm:"column:request_id"`
	JobID            string     `gorm:"column:job_id"`
	BusinessCategory string     `gorm:"column:business_category"`
	BillingMode      string     `gorm:"column:billing_mode"`
	Chargeable       bool       `gorm:"column:chargeable"`
	PriceVersion     string     `gorm:"column:price_version"`
	ReservedQuantity float64    `gorm:"column:reserved_quantity"`
	ActualQuantity   *float64   `gorm:"column:actual_quantity"`
	UnitPrice        float64    `gorm:"column:unit_price"`
	ReservedCost     float64    `gorm:"column:reserved_cost"`
	ActualCost       *float64   `gorm:"column:actual_cost"`
	Estimated        bool       `gorm:"column:estimated"`
	Status           string     `gorm:"column:status"`
	PeriodStartedAt  time.Time  `gorm:"column:period_started_at"`
	ErrorMessage     string     `gorm:"column:error_message"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (Reservation) TableName() string { return "billing_reservations" }

type OutboxRow struct {
	ID            uint64          `gorm:"primaryKey;column:id"`
	EventID       string          `gorm:"column:event_id"`
	TenantID      uint64          `gorm:"column:tenant_id"`
	ReservationID string          `gorm:"column:reservation_id"`
	Payload       json.RawMessage `gorm:"column:payload;type:jsonb"`
	Status        string          `gorm:"column:status"`
	Attempts      int             `gorm:"column:attempts"`
	NextAttemptAt time.Time       `gorm:"column:next_attempt_at"`
	LastError     string          `gorm:"column:last_error"`
	SentAt        *time.Time      `gorm:"column:sent_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
	UpdatedAt     time.Time       `gorm:"column:updated_at"`
}

func (OutboxRow) TableName() string { return "billing_outbox" }

type CreditOperation struct {
	ID                uint64    `gorm:"primaryKey;column:id"`
	TenantID          uint64    `gorm:"column:tenant_id"`
	IdempotencyKey    string    `gorm:"column:idempotency_key"`
	Amount            float64   `gorm:"column:amount"`
	Kind              string    `gorm:"column:kind"`
	Status            string    `gorm:"column:status"`
	MeterForgeGrantID string    `gorm:"column:meterforge_grant_id"`
	LastError         string    `gorm:"column:last_error"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (CreditOperation) TableName() string { return "billing_credit_operations" }

type UsageMetadata struct {
	ModelID   string
	ModelName string
	Provider  string
	Operation string
	RequestID string
	Estimated bool
	JobID     string
	Category  string
	Mode      BillingMode
}

type BillingError struct {
	Code       string
	Message    string
	HTTPStatus int
	Feature    FeatureKey
	ResetAt    *time.Time
}

func (e *BillingError) Error() string { return e.Message }
