# Backend API Endpoints

## Payment Flow

### Create Billing

**Endpoint:** `POST /pay`

**Request:**

```json
{
  "name": "Username",
  "link": "https://twitter.com/user",
  "email": "user@example.com",
  "amount_cents": 5000
}
```

**Response (201):**

```json
{
  "status": "created",
  "checkout_url": "https://abacatepay.com/pay/...",
  "session_id": "bill_123",
  "billing_id": "bill_123"
}
```

Use `checkout_url` to redirect the user to payment.

---

### Backward-Compatible Route

**Endpoint:** `POST /create-payment-intent`

This route now uses the same AbacatePay billing flow as `/pay` and returns the same response shape.

---

## Other Endpoints

- **GET /health** — Health check
- **GET /leaderboard** — Paid payments ordered by amount
- **GET /total** — Total amount invested
- **GET /ws** — WebSocket endpoint
- **POST /webhook** — AbacatePay webhook receiver

---

## WebSocket Events

Connect to `ws://localhost:8080/ws`

**Events received:**
- `payment.created` - new payment row created
- `abacatepay.*` - provider webhook updates for subscribed billing id

**Subscribe commands:**

```json
{"type": "subscribe", "session_id": "bill_123"}
```

```json
{"type": "unsubscribe", "session_id": "bill_123"}
```

---

## Environment Variables

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/paywall?sslmode=disable
ABACATEPAY_API_KEY=your_api_key_here
ABACATEPAY_BASE_URL=https://api.abacatepay.com
ABACATEPAY_CREATE_PATH=/v1/billing/create
ABACATEPAY_WEBHOOK_SECRET=
```
