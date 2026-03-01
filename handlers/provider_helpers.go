package handlers

import "strings"

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeProviderStatus(value string) string {
	status := strings.ToLower(strings.TrimSpace(value))
	if status == "" {
		return "pending"
	}

	switch status {
	case "paid", "approved", "confirmed", "succeeded", "success":
		return "paid"
	case "expired", "canceled", "cancelled", "failed", "refunded", "chargeback", "requires_payment_method", "requires_action":
		return status
	default:
		return status
	}
}
