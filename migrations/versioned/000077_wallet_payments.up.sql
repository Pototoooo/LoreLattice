-- Keep historical USD ledger amounts intact. CNY conversion is locked once by
-- billing_wallet_settings at first use; existing accounts receive no new grant.
CREATE TABLE billing_wallet_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    cny_per_usd NUMERIC(20,8) NOT NULL CHECK (cny_per_usd > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE billing_payment_orders (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id),
    idempotency_key VARCHAR(64) NOT NULL,
    amount_fen BIGINT NOT NULL CHECK (amount_fen > 0),
    currency VARCHAR(3) NOT NULL CHECK (currency = 'CNY'),
    credit_usd NUMERIC(20,8) NOT NULL CHECK (credit_usd > 0),
    cny_per_usd NUMERIC(20,8) NOT NULL CHECK (cny_per_usd > 0),
    app_id VARCHAR(64) NOT NULL,
    seller_id VARCHAR(64) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    trade_id VARCHAR(128) UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    next_check_at TIMESTAMPTZ NOT NULL,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, idempotency_key)
);
CREATE INDEX idx_billing_payments_pending ON billing_payment_orders(status, next_check_at);
CREATE INDEX idx_billing_payments_tenant ON billing_payment_orders(tenant_id, created_at DESC);
CREATE INDEX idx_billing_reservations_recovery ON billing_reservations(status, created_at);
-- Retain old statuses/subscription IDs for audit. Runtime ignores old quota and
-- trial expiry; only an explicit suspended status blocks platform usage.
