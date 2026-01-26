package models

// Payment represents a payment record.
type Payment struct {
	ID          uint  `json:"id"`
	UserID      uint  `json:"user_id"`
	AmountCents int64 `json:"amount_cents"`
}
