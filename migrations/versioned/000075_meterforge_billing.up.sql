DO $$ BEGIN RAISE NOTICE '[Migration 000075] Creating LoreLattice billing ledger...'; END $$;

CREATE TABLE IF NOT EXISTS billing_accounts (
    tenant_id BIGINT PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    meterforge_customer_id VARCHAR(26),
    meterforge_customer_key VARCHAR(256) NOT NULL,
    meterforge_subscription_id VARCHAR(26),
    plan_key VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    trial_started_at TIMESTAMP WITH TIME ZONE,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    period_started_at TIMESTAMP WITH TIME ZONE,
    period_ends_at TIMESTAMP WITH TIME ZONE,
    local_credit_granted NUMERIC(20, 8) NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_accounts_status
    ON billing_accounts(status);

CREATE TABLE IF NOT EXISTS billing_reservations (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    feature_key VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    meter_unit VARCHAR(16) NOT NULL,
    model_id VARCHAR(128),
    model_name VARCHAR(256),
    provider VARCHAR(128),
    operation VARCHAR(128),
    request_id VARCHAR(128),
    reserved_quantity NUMERIC(20, 6) NOT NULL,
    actual_quantity NUMERIC(20, 6),
    unit_price NUMERIC(20, 10) NOT NULL,
    reserved_cost NUMERIC(20, 8) NOT NULL,
    actual_cost NUMERIC(20, 8),
    estimated BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(24) NOT NULL,
    period_started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_reservations_tenant_period
    ON billing_reservations(tenant_id, feature_key, period_started_at, status);

CREATE TABLE IF NOT EXISTS billing_outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL UNIQUE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reservation_id VARCHAR(36) NOT NULL REFERENCES billing_reservations(id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_error TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_outbox_dispatch
    ON billing_outbox(status, next_attempt_at);

CREATE TABLE IF NOT EXISTS billing_credit_operations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(256) NOT NULL,
    amount NUMERIC(20, 8) NOT NULL,
    kind VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    meterforge_grant_id VARCHAR(26),
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, idempotency_key)
);

DO $$ BEGIN RAISE NOTICE '[Migration 000075] LoreLattice billing ledger ready'; END $$;
