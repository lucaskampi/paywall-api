ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider_charge_id TEXT;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider_checkout_url TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_charge_id
  ON payments(provider_charge_id);