DO $$ BEGIN RAISE NOTICE '[Migration 000076] Adding unified AI credit metadata...'; END $$;

ALTER TABLE billing_reservations
    ADD COLUMN IF NOT EXISTS job_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS business_category VARCHAR(64) NOT NULL DEFAULT 'chat_agent',
    ADD COLUMN IF NOT EXISTS billing_mode VARCHAR(24) NOT NULL DEFAULT 'platform',
    ADD COLUMN IF NOT EXISTS chargeable BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS price_version VARCHAR(64) NOT NULL DEFAULT 'legacy-v1';

UPDATE billing_reservations
SET job_id = COALESCE(NULLIF(request_id, ''), id)
WHERE job_id IS NULL OR job_id = '';

CREATE INDEX IF NOT EXISTS idx_billing_reservations_tenant_job
    ON billing_reservations(tenant_id, job_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_reservations_tenant_chargeable
    ON billing_reservations(tenant_id, chargeable, period_started_at, status);

DO $$ BEGIN RAISE NOTICE '[Migration 000076] Unified AI credit metadata ready'; END $$;
