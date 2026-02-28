package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type abacatePayClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type abacateBilling struct {
	ID     string
	URL    string
	Status string
}

func newAbacatePayClientFromEnv() (*abacatePayClient, error) {
	apiKey := strings.TrimSpace(os.Getenv("ABACATEPAY_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("missing ABACATEPAY_API_KEY")
	}

	baseURL := strings.TrimSpace(os.Getenv("ABACATEPAY_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.abacatepay.com"
	}

	timeoutSeconds := getenvIntWithDefault("ABACATEPAY_TIMEOUT_SECONDS", 15)
	return &abacatePayClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}, nil
}

func (client *abacatePayClient) CreateBilling(amountCents int64, description string, customerName string, customerEmail string) (abacateBilling, error) {
	methods := getAbacatePayMethods()
	frequency := getAbacatePayFrequency()
	returnURL := getAbacatePayReturnURL()
	completionURL := getAbacatePayCompletionURL()
	customer := map[string]interface{}{
		"name":      firstNonEmpty(customerName, "Paywall Customer"),
		"email":     firstNonEmpty(customerEmail, os.Getenv("ABACATEPAY_DEFAULT_CUSTOMER_EMAIL"), "customer@example.com"),
		"cellphone": firstNonEmpty(strings.TrimSpace(os.Getenv("ABACATEPAY_CUSTOMER_CELLPHONE")), "+5511999999999"),
		"taxId":     firstNonEmpty(strings.TrimSpace(os.Getenv("ABACATEPAY_CUSTOMER_TAX_ID")), "11144477735"),
	}
	products := []map[string]interface{}{
		{
			"externalId": "paywall-payment",
			"name":       "Paywall Payment",
			"quantity":   1,
			"price":      amountCents,
		},
	}
	payload := map[string]interface{}{
		"amount":        amountCents,
		"description":   description,
		"methods":       methods,
		"frequency":     frequency,
		"products":      products,
		"returnUrl":     returnURL,
		"completionUrl": completionURL,
		"customer":      customer,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return abacateBilling{}, fmt.Errorf("marshal billing payload: %w", err)
	}

	paths := []string{"/v1/billing/create", "/billing/create", "/v1/payments"}
	if customPath := strings.TrimSpace(os.Getenv("ABACATEPAY_CREATE_PATH")); customPath != "" {
		paths = append([]string{customPath}, paths...)
	}

	var lastErr error
	for _, path := range uniquePaths(paths) {
		billing, err := client.createBillingWithPath(path, body)
		if err == nil {
			return billing, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no create paths tried")
	}
	return abacateBilling{}, lastErr
}

func (client *abacatePayClient) createBillingWithPath(path string, body []byte) (abacateBilling, error) {
	endpoint := client.baseURL + normalizePath(path)
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return abacateBilling{}, fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return abacateBilling{}, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	responseBody, _ := io.ReadAll(response.Body)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return abacateBilling{}, fmt.Errorf("abacatepay create failed (%d): %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return abacateBilling{}, fmt.Errorf("parse response: %w", err)
	}

	if apiErr := nestedString(parsed, []string{"error", "message"}); apiErr != "" {
		return abacateBilling{}, fmt.Errorf("abacatepay error: %s", apiErr)
	}
	if apiErr := nestedString(parsed, []string{"error"}); apiErr != "" {
		return abacateBilling{}, fmt.Errorf("abacatepay error: %s", apiErr)
	}

	billingID := firstNonEmpty(
		nestedString(parsed, []string{"data", "id"}),
		nestedString(parsed, []string{"id"}),
	)
	billingURL := firstNonEmpty(
		nestedString(parsed, []string{"data", "url"}),
		nestedString(parsed, []string{"data", "checkoutUrl"}),
		nestedString(parsed, []string{"url"}),
		nestedString(parsed, []string{"checkoutUrl"}),
	)
	billingStatus := normalizeProviderStatus(firstNonEmpty(
		nestedString(parsed, []string{"data", "status"}),
		nestedString(parsed, []string{"status"}),
		"pending",
	))

	if billingID == "" {
		return abacateBilling{}, fmt.Errorf("abacatepay response missing billing id")
	}

	return abacateBilling{ID: billingID, URL: billingURL, Status: billingStatus}, nil
}

func verifyAbacatePaySignature(body []byte, headerSignature string, secret string) bool {
	provided := strings.TrimSpace(headerSignature)
	if provided == "" || secret == "" {
		return false
	}

	if strings.Contains(provided, "=") {
		parts := strings.SplitN(provided, "=", 2)
		provided = strings.TrimSpace(parts[1])
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(strings.ToLower(provided)), []byte(strings.ToLower(expected)))
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	var out []string
	for _, path := range paths {
		normalized := normalizePath(strings.TrimSpace(path))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func nestedString(data map[string]interface{}, path []string) string {
	if len(path) == 0 {
		return ""
	}

	current := interface{}(data)
	for _, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		value, ok := obj[key]
		if !ok {
			return ""
		}
		current = value
	}

	switch value := current.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

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
	case "expired", "canceled", "cancelled", "failed", "refunded", "chargeback":
		return status
	default:
		return status
	}
}

func getenvIntWithDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getAbacatePayMethods() []string {
	raw := strings.TrimSpace(os.Getenv("ABACATEPAY_METHODS"))
	if raw == "" {
		return []string{"PIX"}
	}

	parts := strings.Split(raw, ",")
	methods := make([]string, 0, len(parts))
	for _, part := range parts {
		method := strings.ToUpper(strings.TrimSpace(part))
		if method == "" {
			continue
		}
		methods = append(methods, method)
	}
	if len(methods) == 0 {
		return []string{"PIX"}
	}
	return methods
}

func getAbacatePayFrequency() string {
	value := strings.ToUpper(strings.TrimSpace(os.Getenv("ABACATEPAY_FREQUENCY")))
	if value == "" {
		return "ONE_TIME"
	}
	return value
}

func getAbacatePayReturnURL() string {
	returnURL := strings.TrimSpace(os.Getenv("ABACATEPAY_RETURN_URL"))
	if returnURL != "" {
		return returnURL
	}

	frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	if frontendOrigin != "" {
		return strings.TrimRight(frontendOrigin, "/") + "/payment-return"
	}

	return "http://localhost:3000/payment-return"
}

func getAbacatePayCompletionURL() string {
	completionURL := strings.TrimSpace(os.Getenv("ABACATEPAY_COMPLETION_URL"))
	if completionURL != "" {
		return completionURL
	}

	frontendOrigin := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	if frontendOrigin != "" {
		return strings.TrimRight(frontendOrigin, "/") + "/payment-complete"
	}

	return "http://localhost:3000/payment-complete"
}
