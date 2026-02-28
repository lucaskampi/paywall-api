DROP INDEX IF EXISTS idx_payments_provider_charge_id;

ALTER TABLE payments DROP COLUMN IF EXISTS provider_charge_id;
ALTER TABLE payments DROP COLUMN IF EXISTS provider_checkout_url;