**Overview**
- **Repo**: Minimal scaffold for a pay-to-rank leaderboard backend in Go.

**Quick Start (local)**
- **Create data dir**: `mkdir -p data`
- **Set DB URL**: `export DATABASE_URL="file:$(pwd)/data/paywall.db?_busy_timeout=5000&_foreign_keys=1"`
- **Run migrations** (preferred):
  - Install CLI once: `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
  - Run: `migrate -path ./migrations -database sqlite3://$(pwd)/data/paywall.db up`
- Or run programmatic runner: `go run tools/run_migrations.go`
- **Start server**: `go run main.go`

**Endpoints**
- **GET /health**: health check
- **GET /leaderboard**: leaderboard data
- **POST /pay**: create a payment (expects JSON with `name`, `link`, `amount_cents`)
- **POST /webhook**: payment provider webhooks
- **GET /total**: total invested

**Database & Migrations**
- **Driver**: uses modernc.org/sqlite (pure-Go SQLite driver — no CGO required).
- **Default DB path**: `./data/paywall.db` (controlled via `DATABASE_URL`).
- **PRAGMAs applied on connect**: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`.
- **Migrations**: SQL files live in `migrations/` and are applied with `golang-migrate` or programmatically via `db.RunMigrations`.
- **Dev note**: keep `migrations/` under version control so switching to Postgres later is straightforward.

**Writing to the DB**
- Writes are serialized by a single-writer goroutine (`db.StartWriter`) to avoid SQLITE_BUSY contention in-process. The handler `POST /pay` uses this writer.

**Local helpers**
- Inspect DB (pure-Go): `go run tools/inspect_db.go`
- Insert a test payment via helper: `go run tools/insert_payment.go`

**Files of interest**
- `main.go` — app bootstrap (connects DB, runs migrations, starts writer, serves HTTP)
- `db/` — DB helpers (`connect.go`, `migrate.go`, `writer.go`)
- `migrations/` — SQL migration files
- `handlers/` — HTTP endpoints

**Git / env**
- Commit `go.mod` and `go.sum` for reproducible builds.
- `data/`, `*.db`, and `.env` are ignored by `.gitignore`.

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
