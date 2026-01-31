# Backend API Endpoints

## Payment Flow Options

### Option 1: Checkout Session (Redirect Flow) - Recommended
Use this for simpler integration - redirects user to Stripe-hosted checkout page.

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
  "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_...",
  "session_id": "cs_test_..."
}
```

**Frontend code:**
```javascript
const response = await fetch('http://localhost:8080/pay', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name, link, email, amount_cents })
});
const data = await response.json();
// Redirect to Stripe Checkout
window.location.href = data.checkout_url;
// or: stripe.redirectToCheckout({ sessionId: data.session_id });
```

---

### Option 2: PaymentIntent (In-Modal Flow) - For Stripe Elements
Use this for in-app card collection using Stripe Elements.

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

**Response (201):**
```json
{
  "clientSecret": "pi_xxx_secret_xxx",
  "paymentId": 14
}
```

**Frontend code:**
```javascript
const response = await fetch('http://localhost:8080/create-payment-intent', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name, link, email, amount_cents })
});
const data = await response.json();
// Use clientSecret with Stripe Elements
const { error } = await stripe.confirmCardPayment(data.clientSecret, {
  payment_method: {
    card: cardElement,
    billing_details: { name, email }
  }
});
```

---

## Other Endpoints

**GET /health**  
Health check - returns "OK"

**GET /leaderboard**  
Returns paid payments ordered by amount (highest first)

**GET /total**  
Returns total amount invested

**GET /ws**  
WebSocket endpoint for real-time updates

**POST /webhook**  
Stripe webhook handler (for Stripe events)

---

## WebSocket Events

Connect to `ws://localhost:8080/ws`

**Events received:**
- `payment.created` - New payment created
- `stripe.checkout.session.completed` - Payment succeeded
- `stripe.checkout.session.expired` - Checkout expired

**Events you can send:**
```json
{"type": "subscribe", "session_id": "cs_test_..."}
{"type": "unsubscribe", "session_id": "cs_test_..."}
{"type": "ping"}
```

---

## Environment Variables Required

```env
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_SUCCESS_URL=http://localhost:3000/success
STRIPE_CANCEL_URL=http://localhost:3000/cancel
STRIPE_CURRENCY=usd
STRIPE_PRODUCT_NAME=Paywall payment
```
