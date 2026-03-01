# paywall-api

Backend API for a pay-to-rank leaderboard in Go, using PostgreSQL and Stripe.

## Quick Start

1. Set PostgreSQL DSN in `DATABASE_URL`.
2. Set `STRIPE_SECRET_KEY`.
3. Run:

```bash
./start.sh
```

Server starts on `http://localhost:8080`.

## Required Environment Variables

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/paywall?sslmode=disable
STRIPE_SECRET_KEY=sk_test_xxx
```

## Optional Environment Variables

```env
PORT=8080
FRONTEND_ORIGIN=http://localhost:3000
STRIPE_WEBHOOK_SECRET=whsec_xxx
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MIN=30
```

## Endpoints

- `GET /health`
- `GET /leaderboard`
- `POST /pay` (compat route; delegates to create payment intent)
- `POST /create-payment-intent`
- `POST /pay/confirm`
- `POST /webhook`
- `GET /total`
- `GET /ws`

## Payment Intent Request (`POST /create-payment-intent`)

Request body:

```json
{
  "name": "Username",
  "link": "https://twitter.com/user",
  "email": "user@example.com",
  "amount_cents": 5000
}
```

Response:

```json
{
  "clientSecret": "pi_xxx_secret_xxx",
  "paymentIntentId": "pi_xxx"
}
```

## Webhooks

- Configure Stripe webhook URL to `POST /webhook`.
- If `STRIPE_WEBHOOK_SECRET` is set, signature validation is enforced.
- Webhook events are idempotent via the `webhook_events` table.

## Local Helpers

- `go run tools/run_migrations.go`
- `go run tools/inspect_db.go`
- `go run tools/insert_payment.go`
- `go run tools/clear_payments.go`
