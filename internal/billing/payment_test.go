package billing

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Pototoooo/lorelattice/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testWallet(t *testing.T) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&Account{}, &Reservation{}, &WalletSettings{}, &PaymentOrder{}, &CreditOperation{}, &OutboxRow{}); err != nil {
		t.Fatal(err)
	}
	// Match PostgreSQL's financial uniqueness constraints in fast unit tests.
	for _, stmt := range []string{
		"CREATE UNIQUE INDEX payment_idempotency ON billing_payment_orders(tenant_id,idempotency_key)",
		"CREATE UNIQUE INDEX payment_transaction ON billing_payment_orders(trade_id)",
		"CREATE UNIQUE INDEX credit_idempotency ON billing_credit_operations(tenant_id,idempotency_key)",
		"CREATE UNIQUE INDEX event_identity ON billing_outbox(event_id)",
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	s := NewService(db)
	s.config.Enabled = true
	s.config.MeterForgeEnabled = false
	s.config.WelcomeCreditUSD = 0
	s.config.CNYPerUSD = 7
	s.config.ReservationTTL = 24 * time.Hour
	merchant, _ := rsa.GenerateKey(rand.Reader, 2048)
	platform, _ := rsa.GenerateKey(rand.Reader, 2048)
	s.payment = &AlipayClient{appID: "app-test", sellerID: "seller-test", privateKey: merchant, publicKey: &platform.PublicKey, gateway: "https://openapi.alipay.com/gateway.do", notifyURL: "https://example.com/notify", returnURL: "https://example.com/return", httpClient: &http.Client{Timeout: time.Second}}
	return s
}
func createTestOrder(t *testing.T, s *Service, tenant uint64) *PaymentOrder {
	t.Helper()
	order, err := s.CreatePayment(context.Background(), tenant, 1000, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	return order
}
func confirm(order *PaymentOrder) *PaymentConfirmation {
	return &PaymentConfirmation{OrderID: order.ID, TradeID: "trade-" + order.ID, AppID: order.AppID, SellerID: order.SellerID, AmountFen: order.AmountFen, Status: "TRADE_SUCCESS"}
}
func TestPaymentIdempotencyAndAtomicCredit(t *testing.T) {
	s := testWallet(t)
	ctx := context.Background()
	key := uuid.NewString()
	first, err := s.CreatePayment(ctx, 42, 1000, key)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreatePayment(ctx, 42, 1000, key)
	if err != nil || again.ID != first.ID {
		t.Fatalf("duplicate create: %v", err)
	}
	if _, err := s.CreatePayment(ctx, 42, 5000, key); err == nil {
		t.Fatal("reused key with new amount accepted")
	}
	var before Account
	s.db.First(&before, "tenant_id = ?", 42)
	if before.LocalCreditGranted != 0 {
		t.Fatal("checkout credited money")
	}
	for range 3 {
		if err := s.ApplyPayment(ctx, confirm(first)); err != nil {
			t.Fatal(err)
		}
	}
	var after Account
	s.db.First(&after, "tenant_id = ?", 42)
	if after.LocalCreditGranted != first.CreditUSD {
		t.Fatalf("credited %v, want once %v", after.LocalCreditGranted, first.CreditUSD)
	}
	var n int64
	s.db.Model(&CreditOperation{}).Count(&n)
	if n != 1 {
		t.Fatalf("credit ledger count %d", n)
	}
	if _, err := s.PaymentOrder(ctx, 99, first.ID); err == nil {
		t.Fatal("cross-tenant order exposed")
	}
}
func TestPaymentRejectsMismatchesAndUnpaid(t *testing.T) {
	s := testWallet(t)
	order := createTestOrder(t, s, 42)
	ctx := context.Background()
	for _, mutation := range []func(*PaymentConfirmation){func(c *PaymentConfirmation) { c.AmountFen++ }, func(c *PaymentConfirmation) { c.AppID = "other" }, func(c *PaymentConfirmation) { c.SellerID = "other" }, func(c *PaymentConfirmation) { c.OrderID = "missing" }} {
		c := confirm(order)
		mutation(c)
		if err := s.ApplyPayment(ctx, c); err == nil {
			t.Fatal("mismatch accepted")
		}
	}
	c := confirm(order)
	c.Status = "WAIT_BUYER_PAY"
	if err := s.ApplyPayment(ctx, c); err != nil {
		t.Fatal(err)
	}
	var account Account
	s.db.First(&account, "tenant_id = ?", 42)
	if account.LocalCreditGranted != 0 {
		t.Fatal("unpaid order credited")
	}
	// Late signed success still credits even after the checkout has expired.
	s.db.Model(order).Update("expires_at", time.Now().Add(-time.Hour))
	if err := s.ApplyPayment(ctx, confirm(order)); err != nil {
		t.Fatal(err)
	}
}
func TestPaymentCreditRollsBackOnLedgerConflict(t *testing.T) {
	s := testWallet(t)
	order := createTestOrder(t, s, 42)
	s.db.Create(&CreditOperation{TenantID: 42, IdempotencyKey: "alipay:" + order.ID, Status: "succeeded"})
	if err := s.ApplyPayment(context.Background(), confirm(order)); err == nil {
		t.Fatal("expected credit ledger conflict")
	}
	var current PaymentOrder
	s.db.First(&current, "id = ?", order.ID)
	if current.Status != "pending" {
		t.Fatal("payment status committed without credit")
	}
	var account Account
	s.db.First(&account, "tenant_id = ?", 42)
	if account.LocalCreditGranted != 0 {
		t.Fatal("wallet mutated despite rollback")
	}
}
func signedNotify(t *testing.T, key *rsa.PrivateKey, order *PaymentOrder) url.Values {
	t.Helper()
	v := url.Values{"app_id": {order.AppID}, "seller_id": {order.SellerID}, "out_trade_no": {order.ID}, "trade_no": {"ali-" + order.ID}, "total_amount": {formatFen(order.AmountFen)}, "trade_status": {"TRADE_SUCCESS"}, "sign_type": {"RSA2"}}
	canonical, err := alipayCanonical(v, true)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := rsaSign(key, []byte(canonical))
	if err != nil {
		t.Fatal(err)
	}
	v.Set("sign", sig)
	return v
}
func TestAlipayWebhookVerifiesBeforeCrediting(t *testing.T) {
	s := testWallet(t)
	order := createTestOrder(t, s, 42)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s.payment.publicKey = &key.PublicKey
	h := NewHandler(s)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/notify", h.AlipayNotify)
	post := func(v url.Values) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/notify", strings.NewReader(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.ServeHTTP(rec, req)
		return rec.Code
	}
	bad := signedNotify(t, key, order)
	bad.Set("total_amount", "100.00")
	if post(bad) != 400 {
		t.Fatal("tampered notification accepted")
	}
	duplicate := signedNotify(t, key, order)
	duplicate.Add("app_id", "other")
	if post(duplicate) != 400 {
		t.Fatal("duplicate parameters accepted")
	}
	for range 2 {
		if post(signedNotify(t, key, order)) != 200 {
			t.Fatal("valid notification failed")
		}
	}
	var account Account
	s.db.First(&account, "tenant_id = ?", 42)
	if account.LocalCreditGranted != order.CreditUSD {
		t.Fatal("webhook did not credit exactly once")
	}
}
func TestAlipayCheckoutAndSignedQuery(t *testing.T) {
	s := testWallet(t)
	order := createTestOrder(t, s, 42)
	checkout, _ := url.Parse(order.CheckoutURL)
	v := checkout.Query()
	canonical, _ := alipayCanonical(v, false)
	if err := rsaVerify(&s.payment.privateKey.PublicKey, []byte(canonical), v.Get("sign")); err != nil {
		t.Fatal(err)
	}
	var biz map[string]any
	json.Unmarshal([]byte(v.Get("biz_content")), &biz)
	if biz["total_amount"] != "10.00" || biz["out_trade_no"] != order.ID {
		t.Fatal("wrong signed checkout")
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s.payment.publicKey = &key.PublicKey
	tamper := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		canonical, _ := alipayCanonical(r.PostForm, false)
		if err := rsaVerify(&s.payment.privateKey.PublicKey, []byte(canonical), r.PostForm.Get("sign")); err != nil {
			t.Error(err)
		}
		raw := fmt.Sprintf(`{"code":"10000","out_trade_no":%q,"trade_no":"query-trade","trade_status":"TRADE_SUCCESS","total_amount":"10.00"}`, order.ID)
		sig, _ := rsaSign(key, []byte(raw))
		if tamper {
			raw = strings.Replace(raw, "10.00", "99.00", 1)
		}
		fmt.Fprintf(w, `{"alipay_trade_query_response":%s,"sign":%q}`, raw, sig)
	}))
	defer server.Close()
	s.payment.gateway = server.URL
	result, err := s.payment.Query(context.Background(), *order)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyPayment(context.Background(), result); err != nil {
		t.Fatal(err)
	}
	tamper = true
	if _, err := s.payment.Query(context.Background(), *order); err == nil {
		t.Fatal("unsigned query accepted")
	}
}
func TestWalletRateCannotBeChangedByConfig(t *testing.T) {
	s := testWallet(t)
	first, err := s.walletSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.config.CNYPerUSD = 99
	second, err := s.walletSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.CNYPerUSD != second.CNYPerUSD {
		t.Fatal("existing wallet revalued")
	}
}
func TestParseFen(t *testing.T) {
	for _, raw := range []string{"-1", "NaN", "1.001", "1e2", "+1", "1.", " 1", "0", "1.2.3"} {
		if _, err := parseFen(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	for raw, want := range map[string]int64{"10.00": 1000, "0.01": 1, "1.2": 120, "10": 1000} {
		got, err := parseFen(raw)
		if err != nil || got != want {
			t.Fatalf("parse %s: %v %v", raw, got, err)
		}
	}
}

func TestFinancialHandlersAlwaysRequireOwner(t *testing.T) {
	s := testWallet(t)
	h := NewHandler(s)
	gin.SetMode(gin.TestMode)
	for _, role := range []types.TenantRole{types.TenantRoleViewer, types.TenantRoleContributor, types.TenantRoleAdmin} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			ctx := context.WithValue(c.Request.Context(), types.TenantRoleContextKey, role)
			ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(42))
			c.Request = c.Request.WithContext(ctx)
		})
		r.GET("/payments", h.PaymentOrders)
		r.POST("/payments", h.CreatePayment)
		for _, method := range []string{"GET", "POST"} {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(method, "/payments", nil))
			if rec.Code != 403 {
				t.Fatalf("%s %s allowed: %d", role, method, rec.Code)
			}
		}
	}
}
