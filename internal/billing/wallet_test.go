package billing

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"
)

func TestWalletIgnoresLegacyExpiryQuotaAndRemoteState(t *testing.T) {
	s := testWallet(t)
	ctx := context.Background()
	past := time.Now().Add(-90 * 24 * time.Hour)
	s.db.Create(&Account{TenantID: 42, Status: "reconciling_active", PlanKey: PlanTrial, TrialEndsAt: &past, LocalCreditGranted: 100})
	s.client.baseURL = "http://127.0.0.1:1"
	r, err := s.Reserve(ctx, 42, FeatureEmbeddingTokens, 2000000, UsageMetadata{Mode: BillingModePlatform})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(ctx, r, 1500000, false); err != nil {
		t.Fatal(err)
	}
	balance, err := s.localAvailableCredit(ctx, 42, 100)
	if err != nil || math.Abs(balance-85) > 1e-8 {
		t.Fatalf("balance %v %v", balance, err)
	}
}
func TestWalletBYOKDoesNotRequireCredit(t *testing.T) {
	s := testWallet(t)
	ctx := context.Background()
	r, err := s.Reserve(ctx, 42, FeatureLLMTokens, 10000000, UsageMetadata{Mode: BillingModeBYOK})
	if err != nil {
		t.Fatal(err)
	}
	if r.ReservedCost != 0 || r.Chargeable {
		t.Fatal("BYOK charged")
	}
	if err := s.Complete(ctx, r, 10000000, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reserve(ctx, 42, FeatureLLMTokens, 1, UsageMetadata{Mode: BillingModePlatform}); err == nil {
		t.Fatal("unfunded platform call accepted")
	}
}
func TestWalletSettlementCapAndCrashRecovery(t *testing.T) {
	s := testWallet(t)
	ctx := context.Background()
	s.db.Create(&Account{TenantID: 42, Status: "active", LocalCreditGranted: 1})
	r, err := s.Reserve(ctx, 42, FeatureLLMTokens, 1000, UsageMetadata{Mode: BillingModePlatform})
	if err != nil {
		t.Fatal(err)
	}
	s.db.Model(r).Update("created_at", time.Now().Add(-48*time.Hour))
	if err := s.RecoverReservations(ctx); err != nil {
		t.Fatal(err)
	}
	var recovered Reservation
	s.db.First(&recovered, "id = ?", r.ID)
	if recovered.Status != "recovered" || recovered.ActualCost == nil {
		t.Fatal("stale reservation not recovered")
	}
	// Late actual usage replaces the estimate; duplicate completion is idempotent.
	for range 2 {
		if err := s.Complete(ctx, r, 500, false); err != nil {
			t.Fatal(err)
		}
	}
	balance, _ := s.localAvailableCredit(ctx, 42, 1)
	if math.Abs(balance-.95) > 1e-8 {
		t.Fatalf("late settlement %v", balance)
	}
	next, err := s.Reserve(ctx, 42, FeatureLLMTokens, 100, UsageMetadata{Mode: BillingModePlatform})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(ctx, next, 10000, false); err != nil {
		t.Fatal(err)
	}
	balance, _ = s.localAvailableCredit(ctx, 42, 1)
	if math.Abs(balance-.94) > 1e-8 {
		t.Fatalf("charge exceeded authorized reserve: %v", balance)
	}
}
func TestTelemetryUsesUnpricedNamespace(t *testing.T) {
	s := testWallet(t)
	s.config.MeterForgeEnabled = true
	ctx := context.Background()
	s.db.Create(&Account{TenantID: 42, Status: "active", LocalCreditGranted: 1})
	r, err := s.Reserve(ctx, 42, FeatureLLMTokens, 100, UsageMetadata{Mode: BillingModePlatform})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Complete(ctx, r, 50, false); err != nil {
			t.Fatal(err)
		}
	}
	var n int64
	s.db.Model(&OutboxRow{}).Count(&n)
	if n != 1 {
		t.Fatalf("outbox count %d", n)
	}
	var row OutboxRow
	s.db.First(&row)
	if !strings.Contains(string(row.Payload), `"type":"lorelattice.wallet.usage.v1"`) || !strings.Contains(string(row.Payload), `wallet:42`) {
		t.Fatalf("old priced namespace used: %s", row.Payload)
	}
}
