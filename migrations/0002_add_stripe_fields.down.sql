-- Rolls back Stripe-related fields by recreating the payments table.

DROP TABLE IF EXISTS webhook_events;
DROP INDEX IF EXISTS idx_payments_stripe_checkout_session_id;
DROP INDEX IF EXISTS idx_payments_stripe_payment_intent_id;

CREATE TABLE payments_new (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  link TEXT,
  email TEXT,
  amount_cents INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO payments_new (id, name, link, email, amount_cents, created_at)
SELECT id, name, link, email, amount_cents, created_at
FROM payments;

DROP TABLE payments;
ALTER TABLE payments_new RENAME TO payments;
