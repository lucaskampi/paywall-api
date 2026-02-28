-- Rolls back provider status/tracking fields.

DROP TABLE IF EXISTS webhook_events;
DROP INDEX IF EXISTS idx_payments_stripe_checkout_session_id;
DROP INDEX IF EXISTS idx_payments_stripe_payment_intent_id;

ALTER TABLE payments DROP COLUMN IF EXISTS status;
ALTER TABLE payments DROP COLUMN IF EXISTS currency;
ALTER TABLE payments DROP COLUMN IF EXISTS provider;
ALTER TABLE payments DROP COLUMN IF EXISTS stripe_checkout_session_id;
ALTER TABLE payments DROP COLUMN IF EXISTS stripe_payment_intent_id;
ALTER TABLE payments DROP COLUMN IF EXISTS paid_at;
ALTER TABLE payments DROP COLUMN IF EXISTS updated_at;
