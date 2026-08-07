package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOverviewFallsBackToLocalLedgerWhenMeterForgeIsUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:billing-overview-fallback?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Account{}, &Reservation{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)
	account := Account{
		TenantID: 42, MeterForgeCustomerID: "customer-42", MeterForgeCustomerKey: "tenant-42",
		PlanKey: PlanPro, Status: "active", PeriodStartedAt: &now, PeriodEndsAt: &periodEnd,
		LocalCreditGranted: 10,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatal(err)
	}
	actualQuantity, actualCost := 100.0, 2.0
	if err := db.Create(&Reservation{
		ID: "charged-call", TenantID: 42, FeatureKey: string(FeatureLLMTokens),
		BillingMode: string(BillingModePlatform), Chargeable: true,
		ActualQuantity: &actualQuantity, ActualCost: &actualCost, Status: "completed",
		CreatedAt: now, PeriodStartedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporary outage", http.StatusServiceUnavailable)
	}))
	defer remote.Close()
	service := NewService(db)
	service.config.Enabled = true
	service.client = &Client{baseURL: remote.URL, client: remote.Client()}

	overview, err := service.Overview(context.Background(), 42, true)
	if err != nil {
		t.Fatalf("Overview() returned an error during remote outage: %v", err)
	}
	if overview.FinancialStatus != "local_fallback" {
		t.Fatalf("FinancialStatus = %q, want local_fallback", overview.FinancialStatus)
	}
	if overview.AICredits.RemainingUSD == nil || *overview.AICredits.RemainingUSD != 8 {
		t.Fatalf("remaining credits = %v, want 8", overview.AICredits.RemainingUSD)
	}
}
