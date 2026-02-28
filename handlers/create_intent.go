package handlers

import "net/http"

// CreatePaymentIntent keeps backward compatibility for clients still calling
// this route; AbacatePay uses the same billing flow as /pay.
func CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	Pay(w, r)
}
