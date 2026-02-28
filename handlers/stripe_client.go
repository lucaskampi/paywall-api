package handlers

import (
	"fmt"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

type stripeClient struct {
	secretKey string
}

func newStripeClientFromEnv() (*stripeClient, error) {
	secretKey := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if secretKey == "" {
		return nil, fmt.Errorf("missing STRIPE_SECRET_KEY")
	}
	return &stripeClient{secretKey: secretKey}, nil
}

func (client *stripeClient) CreatePaymentIntent(amountCents int64, name string, link string, email string) (*stripe.PaymentIntent, error) {
	stripe.Key = client.secretKey

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		Metadata: map[string]string{
			"name":  firstNonEmpty(name),
			"link":  firstNonEmpty(link),
			"email": firstNonEmpty(email),
		},
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	return paymentintent.New(params)
}

func (client *stripeClient) GetPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	stripe.Key = client.secretKey
	return paymentintent.Get(paymentIntentID, nil)
}
