package billing

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletSettings struct {
	ID        int `gorm:"primaryKey"`
	CNYPerUSD float64
	CreatedAt time.Time
}

func (WalletSettings) TableName() string { return "billing_wallet_settings" }

// Lock the commercial conversion coefficient once for the whole installation.
// This is not a live FX feed: changing an env var must never revalue balances.
func (s *Service) walletSettings(ctx context.Context) (*WalletSettings, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if s.config.CNYPerUSD <= 0 {
		return nil, fmt.Errorf("BILLING_CNY_PER_USD must be positive")
	}
	row := WalletSettings{ID: 1, CNYPerUSD: s.config.CNYPerUSD}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&row, 1).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type PaymentOrder struct {
	ID             string     `gorm:"primaryKey" json:"id"`
	TenantID       uint64     `json:"-"`
	IdempotencyKey string     `json:"-"`
	AmountFen      int64      `json:"amount_fen"`
	Currency       string     `json:"currency"`
	CreditUSD      float64    `json:"-"`
	CNYPerUSD      float64    `json:"-"`
	AppID          string     `json:"-"`
	SellerID       string     `json:"-"`
	Status         string     `json:"status"`
	TradeID        *string    `json:"-"`
	ExpiresAt      time.Time  `json:"expires_at"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	NextCheckAt    time.Time  `json:"-"`
	LastError      string     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"-"`
	CheckoutURL    string     `gorm:"-" json:"checkout_url,omitempty"`
}

func (PaymentOrder) TableName() string { return "billing_payment_orders" }

func (s *Service) CreatePayment(ctx context.Context, tenantID uint64, amountFen int64, key string) (*PaymentOrder, error) {
	if !s.Enabled() || s.payment == nil {
		return nil, billingError("payment_unavailable", "充值暂未开放，请联系管理员", 503, "", nil)
	}
	if amountFen != 1000 && amountFen != 5000 && amountFen != 10000 {
		return nil, billingError("invalid_top_up", "充值金额只能是 ¥10、¥50 或 ¥100", 400, "", nil)
	}
	if _, err := uuid.Parse(key); err != nil {
		return nil, billingError("invalid_top_up", "请求幂等标识无效", 400, "", nil)
	}
	if err := s.ProvisionTenant(ctx, tenantID, ""); err != nil {
		return nil, err
	}
	settings, err := s.walletSettings(ctx)
	if err != nil {
		return nil, err
	}
	var order PaymentOrder
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "tenant_id = ?", tenantID).Error; err != nil {
			return err
		}
		if account.Status == "suspended" {
			return billingError("billing_not_ready", "账户已暂停", 403, "", nil)
		}
		found := tx.Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).First(&order).Error
		if found == nil {
			if order.AmountFen != amountFen {
				return billingError("payment_conflict", "同一请求不能更改充值金额", 409, "", nil)
			}
			return nil
		}
		if found != gorm.ErrRecordNotFound {
			return found
		}
		var count int64
		if err := tx.Model(&PaymentOrder{}).Where("tenant_id = ? AND status = 'pending' AND expires_at > ?", tenantID, time.Now().UTC()).Count(&count).Error; err != nil {
			return err
		}
		if count >= 5 {
			return billingError("payment_conflict", "请先完成已有充值订单", 429, "", nil)
		}
		now := time.Now().UTC()
		order = PaymentOrder{ID: "ll" + uuid.NewString(), TenantID: tenantID, IdempotencyKey: key, AmountFen: amountFen, Currency: "CNY",
			CreditUSD: math.Floor(float64(amountFen)/100/settings.CNYPerUSD*1e8) / 1e8, CNYPerUSD: settings.CNYPerUSD,
			AppID: s.payment.appID, SellerID: s.payment.sellerID, Status: "pending", ExpiresAt: now.Add(30 * time.Minute), NextCheckAt: now.Add(time.Minute)}
		return tx.Create(&order).Error
	})
	if err != nil {
		return nil, err
	}
	if order.Status == "pending" && time.Now().Before(order.ExpiresAt) {
		if order.AppID != s.payment.appID || order.SellerID != s.payment.sellerID {
			return nil, billingError("payment_conflict", "商户配置已变更，请创建新订单", 409, "", nil)
		}
		order.CheckoutURL, err = s.payment.Checkout(order)
	}
	return &order, err
}

// The only credit-entry path. Called solely with a cryptographically verified
// notification or signed query result, never with a browser return URL.
func (s *Service) ApplyPayment(ctx context.Context, confirmation *PaymentConfirmation) error {
	if confirmation == nil || confirmation.TradeID == "" || len(confirmation.TradeID) > 128 {
		return fmt.Errorf("invalid payment confirmation")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order PaymentOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, "id = ?", confirmation.OrderID).Error; err != nil {
			return err
		}
		if confirmation.AmountFen != order.AmountFen || confirmation.AppID != order.AppID || confirmation.SellerID != order.SellerID || order.Currency != "CNY" {
			return fmt.Errorf("payment order mismatch")
		}
		if order.TradeID != nil && *order.TradeID != confirmation.TradeID {
			return fmt.Errorf("payment transaction mismatch")
		}
		if order.Status == "paid" {
			return nil
		}
		if confirmation.Status != "TRADE_SUCCESS" && confirmation.Status != "TRADE_FINISHED" {
			if confirmation.Status == "TRADE_CLOSED" {
				return tx.Model(&order).Updates(map[string]any{"status": "closed", "updated_at": time.Now().UTC()}).Error
			}
			return nil
		}
		// An expired local checkout may have been paid just before expiry. Provider
		// payment confirmation is authoritative and must still credit the customer.
		now := time.Now().UTC()
		result := tx.Model(&PaymentOrder{}).Where("id = ? AND status <> 'paid'", order.ID).Updates(map[string]any{"status": "paid", "trade_id": confirmation.TradeID, "paid_at": now, "last_error": "", "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		op := CreditOperation{TenantID: order.TenantID, IdempotencyKey: "alipay:" + order.ID, Amount: order.CreditUSD, Kind: "alipay_topup", Status: "succeeded"}
		if err := tx.Create(&op).Error; err != nil {
			return err
		}
		result = tx.Model(&Account{}).Where("tenant_id = ?", order.TenantID).UpdateColumn("local_credit_granted", gorm.Expr("local_credit_granted + ?", order.CreditUSD))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("payment wallet missing")
		}
		return nil
	})
}
func (s *Service) PaymentOrders(ctx context.Context, tenantID uint64) ([]PaymentOrder, error) {
	orders := []PaymentOrder{}
	err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Limit(50).Find(&orders).Error
	return orders, err
}
func (s *Service) PaymentOrder(ctx context.Context, tenantID uint64, id string) (*PaymentOrder, error) {
	var order PaymentOrder
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&order).Error; err != nil {
		return nil, billingError("payment_not_found", "充值订单不存在", 404, "", nil)
	}
	if order.Status == "pending" {
		_ = s.checkPayment(ctx, order)
	}
	if err := s.db.WithContext(ctx).First(&order, "id = ?", order.ID).Error; err != nil {
		return nil, err
	}
	if s.payment != nil && order.Status == "pending" && time.Now().Before(order.ExpiresAt) && order.AppID == s.payment.appID && order.SellerID == s.payment.sellerID {
		order.CheckoutURL, _ = s.payment.Checkout(order)
	}
	return &order, nil
}
func (s *Service) checkPayment(ctx context.Context, order PaymentOrder) error {
	if s.payment == nil {
		return nil
	}
	if order.AppID != s.payment.appID || order.SellerID != s.payment.sellerID {
		return fmt.Errorf("payment merchant changed; old order requires original merchant configuration")
	}
	now := time.Now().UTC()
	delay := time.Minute
	if now.After(order.ExpiresAt) {
		delay = time.Hour
	}
	// Atomic claim prevents browser polling / multiple replicas from flooding the gateway.
	claimed := s.db.WithContext(ctx).Model(&PaymentOrder{}).Where("id = ? AND status = 'pending' AND next_check_at <= ?", order.ID, now).Update("next_check_at", now.Add(delay))
	if claimed.Error != nil {
		return claimed.Error
	}
	if claimed.RowsAffected == 0 {
		return nil
	}
	confirmation, err := s.payment.Query(ctx, order)
	if err == nil {
		err = s.ApplyPayment(ctx, confirmation)
	}
	if err != nil {
		_ = s.db.WithContext(ctx).Model(&PaymentOrder{}).Where("id = ?", order.ID).Update("last_error", err.Error()).Error
	}
	return err
}
func (s *Service) ReconcilePayments(ctx context.Context) error {
	if s.payment == nil {
		return nil
	}
	var orders []PaymentOrder
	if err := s.db.WithContext(ctx).Where("status = 'pending' AND next_check_at <= ?", time.Now().UTC()).Order("next_check_at ASC").Limit(20).Find(&orders).Error; err != nil {
		return err
	}
	for _, order := range orders {
		_ = s.checkPayment(ctx, order)
	}
	return nil
}
