**Overview**
- **Repo**: Minimal scaffold for a pay-to-rank leaderboard backend in Go.

**Quick Start (local)**
1. **Create data directory**: `mkdir -p data`
2. **Copy environment template**: `cp .env.example .env`
3. **Configure `.env`**:
   - Add your Stripe keys (`STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`)
   - Set success/cancel URLs for your frontend
   - Optional: set `STRIPE_PRICE_ID` for fixed-price payments
4. **Start server**: `./start.sh` (loads `.env` automatically)
   - Or manually: `export $(grep -v '^#' .env | xargs) && go run .`

**Stripe Webhook Setup (local testing)**
- Install Stripe CLI: [stripe.com/docs/stripe-cli](https://stripe.com/docs/stripe-cli)
- Run listener: `/path/to/stripe listen --forward-to http://localhost:8080/webhook`
- Copy the webhook signing secret (`whsec_...`) to `.env` as `STRIPE_WEBHOOK_SECRET`
- Trigger test events: `/path/to/stripe trigger checkout.session.completed`

**Endpoints**
- **GET /health**: health check
- **GET /leaderboard**: returns paid payments ordered by amount (highest first)
- **POST /pay**: create a Stripe checkout session (expects JSON: `name`, `link`, `email`, `amount_cents`)
- **POST /webhook**: Stripe webhook handler (verifies signature, updates payment status, broadcasts WS events)
- **GET /total**: total amount invested
- **GET /ws**: WebSocket endpoint for real-time leaderboard updates

**Database & Migrations**
- **Driver**: uses modernc.org/sqlite (pure-Go SQLite driver — no CGO required).
- **Default DB path**: `./data/paywall.db` (controlled via `DATABASE_URL`).
- **PRAGMAs applied on connect**: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`.
- **Migrations**: SQL files live in `migrations/` and are applied with `golang-migrate` or programmatically via `db.RunMigrations`.
- **Dev note**: keep `migrations/` under version control so switching to Postgres later is straightforward.

**Writing to the DB**
- Writes are serialized by a single-writer goroutine (`db.StartWriter`) to avoid SQLITE_BUSY contention in-process. The handler `POST /pay` uses this writer.

**Local helpers**
- **Start server with env vars**: `./start.sh`
- **Inspect DB** (pure-Go): `go run tools/inspect_db.go`
- **Insert test payment**: `go run tools/insert_payment.go`
- **Clear all payments**: `go run tools/clear_payments.go`

**Files of interest**
- `main.go` — app bootstrap (connects DB, runs migrations, starts writer, serves HTTP)
- `db/` — DB helpers (`connect.go`, `migrate.go`, `writer.go`)
- `migrations/` — SQL migration files
- `handlers/` — HTTP endpoints

**Git / env**
- Commit `go.mod` and `go.sum` for reproducible builds.
- `data/`, `*.db`, and `.env` are ignored by `.gitignore`.

**Deployment / env notes**
- `FRONTEND_ORIGIN`: restrict CORS in production (defaults to `*` in dev if unset).
- `DATABASE_URL`: path/DSN for SQLite (default `file:./data/paywall.db?...`).
- `MIGRATIONS_DIR`: optional — consider using an absolute path or embedding migrations for production. Avoid relying on `./migrations` when running a built binary from other CWDs.
- `NEXT_PUBLIC_WS_URL` / `WS_URL`: frontend may set this to the websocket endpoint (default `ws://localhost:8080/ws`).

**Server timeouts**
- The server uses conservative timeouts: `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` to improve robustness in production. Adjust values in `main.go` as needed.


If you want, I can add a short integration test or update `POST /pay` to return the created row id — tell me which.
# paywall-api (scaffold)

Minimal scaffold for a pay-to-rank leaderboard backend in Go.

Run locally:

```powershell
go run main.go
# then open http://localhost:8080/health
```

Endpoints (placeholders):
- `GET /health` — health check
- `GET /leaderboard` — leaderboard data (placeholder)
- `POST /pay` — create a payment (placeholder)
- `POST /webhook` — payment provider webhooks (placeholder)
- `GET /total` — total invested (placeholder)

Next steps:
- Add dependencies (Gin, GORM), wire DB, implement payment flow (Stripe), and add tests.
# paywall-api
