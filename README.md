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
