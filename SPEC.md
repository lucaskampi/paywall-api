# Stripe Reintroduction Spec

Date: 2026-02-28
Project: paywall-api + paywall-frontend

## Objective
Move the payment flow back from AbacatePay to Stripe with embedded checkout UX (Stripe Elements modal), while keeping backend route compatibility during rollout.

## Decisions Locked
- Checkout UX: Stripe Elements modal embedded in frontend.
- Rollout strategy: keep `/pay` compatibility during migration to reduce downtime risk.
- Webhook model: Stripe webhooks with idempotency via `webhook_events` table.
- Data model: keep existing provider-neutral + legacy Stripe columns; write new rows with `provider='stripe'`.

## Scope
Reintroduce Stripe across backend and frontend runtime code, env/config, and docs. Preserve existing non-payment endpoints and leaderboard behavior.

## Implementation Requirements

### 1) Backend API contract and flow
- Restore Stripe-first flow in backend handlers:
  - `POST /create-payment-intent` returns `clientSecret` and `paymentIntentId`.
  - `POST /pay/confirm` persists/updates payment row after successful intent confirmation.
  - `POST /pay` remains available as compatibility route during transition.
- Ensure `handlers/pay.go` is no longer AbacatePay-dependent at runtime.

### 2) Stripe integration backend
- Add Stripe Go SDK dependency.
- Introduce Stripe client/service helper that:
  - Creates PaymentIntents in cents with USD currency.
  - Reads `STRIPE_SECRET_KEY` from env.
- Replace AbacatePay webhook verification/parse logic with Stripe signature verification (`STRIPE_WEBHOOK_SECRET`) and Stripe event handling.

### 3) Database writes and status transitions
- On creation/confirmation, write provider fields consistently:
  - `provider='stripe'`
  - `stripe_payment_intent_id` and optional `stripe_checkout_session_id` when applicable
  - `status` transitions (`pending` -> `paid`)
- Keep idempotency using `webhook_events(provider, event_id)` with `provider='stripe'`.
- Preserve existing rows from prior providers (no destructive backfill required).

### 4) Frontend checkout flow
- Restore Stripe Elements modal UI in `paywall-frontend`:
  - Form submit -> call `/create-payment-intent`
  - Render Stripe Payment Element in modal
  - Confirm payment client-side with publishable key
  - Call backend confirm route to persist row and broadcast update
- Remove AbacatePay checkout URL popup UX from active flow.

### 5) Environment variables and deployment
- Backend required env:
  - `STRIPE_SECRET_KEY`
  - `STRIPE_WEBHOOK_SECRET`
  - `FRONTEND_ORIGIN`
  - `DATABASE_URL`
- Frontend required env:
  - `NEXT_PUBLIC_API_URL`
  - `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`
- Remove AbacatePay vars from active deployment after cutover validation.

### 6) Documentation updates
- Update backend docs (`README.md`, `API_REFERENCE.md`) to Stripe-first endpoints and webhook expectations.
- Update frontend docs to Stripe Elements setup and test card flow.
- Keep migration notes concise and operationally focused.

## Out of Scope
- Multi-provider runtime switching between Stripe and AbacatePay.
- Historical data migration between providers.
- UI redesign unrelated to checkout behavior.

## Acceptance Criteria
- Backend builds: `go build ./...` succeeds.
- Frontend builds: `npm run build` succeeds.
- End-to-end Stripe test mode flow works:
  1) create intent
  2) confirm payment in modal
  3) payment persists in DB with `provider='stripe'`
  4) leaderboard updates
- Webhook endpoint verifies Stripe signatures and handles retries idempotently.

## Verification Checklist
1. Backend: `go build ./...`
2. Frontend: `npm install && npm run build`
3. API smoke tests: `/health`, `/create-payment-intent`, `/pay/confirm`, `/leaderboard`, `/total`
4. Stripe test payment with card `4242 4242 4242 4242`
5. Webhook replay in Stripe dashboard confirms idempotent processing
6. Production env vars validated on Railway + Vercel

## Risks
- Frontend/backend contract drift if route payloads mismatch.
- Missing Stripe env vars causing runtime 500s.
- Webhook signature failures if endpoint secret mismatched across environments.

## Rollback Strategy
- Keep compatibility route (`/pay`) live during transition.
- If Stripe rollout fails, redeploy previous revision and restore previous env set.
- Avoid destructive migrations so previous data remains queryable.
