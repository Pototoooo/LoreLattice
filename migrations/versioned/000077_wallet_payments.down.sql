-- Never silently drop financial orders or re-enable legacy charging on rollback.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM billing_payment_orders) THEN
        RAISE EXCEPTION 'Wallet payment orders exist; export/reconcile payments before an explicit rollback';
    END IF;
END $$;
DROP TABLE billing_payment_orders;
DROP TABLE billing_wallet_settings;
DROP INDEX idx_billing_reservations_recovery;
