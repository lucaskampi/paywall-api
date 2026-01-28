-- Adds Stripe-related fields for payment tracking and a webhook event dedupe table.

ALTER TABLE payments ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE payments ADD COLUMN currency TEXT NOT NULL DEFAULT 'usd';
ALTER TABLE payments ADD COLUMN provider TEXT NOT NULL DEFAULT 'stripe';
ALTER TABLE payments ADD COLUMN stripe_checkout_session_id TEXT;
ALTER TABLE payments ADD COLUMN stripe_payment_intent_id TEXT;
ALTER TABLE payments ADD COLUMN paid_at DATETIME;
ALTER TABLE payments ADD COLUMN updated_at DATETIME;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_stripe_checkout_session_id
  ON payments(stripe_checkout_session_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_stripe_payment_intent_id
  ON payments(stripe_payment_intent_id);

CREATE TABLE IF NOT EXISTS webhook_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider TEXT NOT NULL,
  event_id TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(provider, event_id)
);
