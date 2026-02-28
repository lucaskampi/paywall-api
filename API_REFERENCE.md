# Backend API Endpoints

## Payment Flow

### Create Payment Intent

**Endpoint:** `POST /create-payment-intent`

**Request:**

```json
{
  "name": "Username",
  "link": "https://twitter.com/user",
  "email": "user@example.com",
  "amount_cents": 5000
}
```

**Response (200):**

```json
{
  "clientSecret": "pi_xxx_secret_xxx",
  "paymentIntentId": "pi_xxx"
}
```

Use `clientSecret` in Stripe Elements on the frontend.

---

### Confirm Payment

**Endpoint:** `POST /pay/confirm`

**Request:**

```json
{
  "payment_intent_id": "pi_xxx",
  "name": "Username",
  "link": "https://twitter.com/user",
  "email": "user@example.com",
  "amount_cents": 5000
}
```

This confirms the payment intent with Stripe and persists/updates the payment row.

---

### Compatibility Route

**Endpoint:** `POST /pay`

For backward compatibility this route currently delegates to `POST /create-payment-intent`.

---

## Other Endpoints

- **GET /health** — Health check
- **GET /leaderboard** — Paid payments ordered by amount
- **GET /total** — Total amount invested
- **GET /ws** — WebSocket endpoint
- **POST /webhook** — Stripe webhook receiver

---

## WebSocket Events

Connect to `ws://localhost:8080/ws`

**Events received:**
- `payment.created` - new payment row created
- `stripe.*` - provider webhook updates for subscribed payment intent

**Subscribe commands:**

```json
{"type": "subscribe", "session_id": "pi_xxx"}
```

```json
{"type": "unsubscribe", "session_id": "pi_xxx"}
```

---

## Environment Variables

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/paywall?sslmode=disable
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
```
