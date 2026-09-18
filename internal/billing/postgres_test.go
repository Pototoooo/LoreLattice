package billing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Run only against an expendable test database; each run has a private schema.
func TestPostgresWalletConcurrency(t *testing.T) {
	dsn := os.Getenv("BILLING_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set BILLING_TEST_POSTGRES_DSN for real row-lock and migration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := "billing_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
	scoped, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := scoped.DB()
	defer conn.Close()
	conn.SetMaxOpenConns(20)
	if err := scoped.Exec("CREATE TABLE tenants (id BIGINT PRIMARY KEY); INSERT INTO tenants(id) VALUES (42)").Error; err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000075_meterforge_billing.up.sql", "000076_unified_ai_credits.up.sql", "000077_wallet_payments.up.sql"} {
		sql, err := os.ReadFile("../../migrations/versioned/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := scoped.Exec(string(sql)).Error; err != nil {
			t.Fatalf("migration %s: %v", name, err)
		}
	}
	s := NewService(scoped)
	s.config.Enabled = true
	s.config.MeterForgeEnabled = false
	s.config.WelcomeCreditUSD = 0
	s.config.CNYPerUSD = 7
	if err := s.ProvisionTenant(context.Background(), 42, ""); err != nil {
		t.Fatal(err)
	}
	// Exactly ten $0.10 reservations fit a $1 wallet even with 30 simultaneous calls.
	scoped.Model(&Account{}).Where("tenant_id = ?", 42).Update("local_credit_granted", 1)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Reserve(context.Background(), 42, FeatureLLMTokens, 1000, UsageMetadata{Mode: BillingModePlatform})
			if err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 10 {
		t.Fatalf("concurrent reservations = %d, want 10", successes.Load())
	}
	// Duplicate callbacks racing across replicas must credit precisely once.
	order := PaymentOrder{ID: "test-order", TenantID: 42, IdempotencyKey: uuid.NewString(), AmountFen: 1000, Currency: "CNY", CreditUSD: 1.42857142, CNYPerUSD: 7, AppID: "app", SellerID: "seller", Status: "pending"}
	if err := scoped.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	errors := make(chan error, 30)
	for range 30 {
		wg.Add(1)
		go func() { defer wg.Done(); errors <- s.ApplyPayment(context.Background(), confirm(&order)) }()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var account Account
	scoped.First(&account, "tenant_id = ?", 42)
	if fmt.Sprintf("%.8f", account.LocalCreditGranted) != "2.42857142" {
		t.Fatalf("duplicate payment balance %v", account.LocalCreditGranted)
	}
	var n int64
	scoped.Model(&CreditOperation{}).Count(&n)
	if n != 1 {
		t.Fatalf("credit entries %d", n)
	}
	// Down migration refuses to erase real financial history.
	down, _ := os.ReadFile("../../migrations/versioned/000077_wallet_payments.down.sql")
	if err := scoped.Exec(string(down)).Error; err == nil {
		t.Fatal("rollback erased payment orders")
	}
}
