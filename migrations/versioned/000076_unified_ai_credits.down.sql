DROP INDEX IF EXISTS idx_billing_reservations_tenant_chargeable;
DROP INDEX IF EXISTS idx_billing_reservations_tenant_job;

ALTER TABLE billing_reservations
    DROP COLUMN IF EXISTS price_version,
    DROP COLUMN IF EXISTS chargeable,
    DROP COLUMN IF EXISTS billing_mode,
    DROP COLUMN IF EXISTS business_category,
    DROP COLUMN IF EXISTS job_id;
