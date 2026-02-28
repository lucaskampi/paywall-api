# paywall-api

Backend API for a pay-to-rank leaderboard in Go, using PostgreSQL and AbacatePay.

## Quick Start

1. Set PostgreSQL DSN in `DATABASE_URL`.
2. Set `ABACATEPAY_API_KEY`.
3. Run:

```bash
./start.sh
```

Server starts on `http://localhost:8080`.

## Required Environment Variables

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/paywall?sslmode=disable
ABACATEPAY_API_KEY=your_api_key_here
```

## Optional Environment Variables

```env
PORT=8080
FRONTEND_ORIGIN=http://localhost:3000
ABACATEPAY_BASE_URL=https://api.abacatepay.com
ABACATEPAY_CREATE_PATH=/v1/billing/create
ABACATEPAY_TIMEOUT_SECONDS=15
ABACATEPAY_WEBHOOK_SECRET=
DB_MAX_OPEN_CONNS=20
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MIN=30
```

## Endpoints

- `GET /health`
- `GET /leaderboard`
- `POST /pay`
- `POST /create-payment-intent` (compat route; same behavior as `/pay`)
- `POST /webhook`
- `GET /total`
- `GET /ws`

## Payment Request (`POST /pay`)

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
  "status": "created",
  "checkout_url": "https://...",
  "session_id": "bill_xxx",
  "billing_id": "bill_xxx"
}
```

## Webhooks

- Configure AbacatePay webhook URL to `POST /webhook`.
- If `ABACATEPAY_WEBHOOK_SECRET` is set, signature validation is enforced.
- Webhook events are idempotent via the `webhook_events` table.

## Local Helpers

- `go run tools/run_migrations.go`
- `go run tools/inspect_db.go`
- `go run tools/insert_payment.go`
- `go run tools/clear_payments.go`
