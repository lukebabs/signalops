package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lukebabs/signalops/internal/storage"
)

const stripeCheckoutEndpoint = "https://api.stripe.com/v1/checkout/sessions"

type stripeCheckoutClient interface {
	CreateCheckoutSession(context.Context, stripeCheckoutSessionRequest) (stripeCheckoutSession, error)
}

type stripeCheckoutSessionRequest struct {
	PriceID       string
	SuccessURL    string
	CancelURL     string
	CheckoutRef   string
	ProductKey    string
	BillingPeriod string
}

type stripeCheckoutSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type stripeHTTPCheckoutClient struct {
	apiKey     string
	httpClient *http.Client
}

func (c stripeHTTPCheckoutClient) CreateCheckoutSession(ctx context.Context, input stripeCheckoutSessionRequest) (stripeCheckoutSession, error) {
	apiKey := strings.TrimSpace(c.apiKey)
	if apiKey == "" {
		return stripeCheckoutSession{}, errors.New("stripe API key is not configured")
	}
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("line_items[0][price]", strings.TrimSpace(input.PriceID))
	form.Set("line_items[0][quantity]", "1")
	form.Set("success_url", strings.TrimSpace(input.SuccessURL))
	form.Set("cancel_url", strings.TrimSpace(input.CancelURL))
	form.Set("automatic_tax[enabled]", "true")
	form.Set("metadata[checkout_ref]", strings.TrimSpace(input.CheckoutRef))
	form.Set("subscription_data[metadata][checkout_ref]", strings.TrimSpace(input.CheckoutRef))
	form.Set("metadata[source]", "marketops_public_checkout")
	form.Set("subscription_data[metadata][source]", "marketops_public_checkout")
	form.Set("metadata[product_key]", strings.TrimSpace(input.ProductKey))
	form.Set("subscription_data[metadata][product_key]", strings.TrimSpace(input.ProductKey))
	form.Set("metadata[billing_period]", strings.TrimSpace(input.BillingPeriod))
	form.Set("subscription_data[metadata][billing_period]", strings.TrimSpace(input.BillingPeriod))
	form.Set("integration_identifier", "signalops_marketops_checkout_v1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stripeCheckoutEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "SignalOps/marketops-subscription-checkout")
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var stripeErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &stripeErr)
		message := firstNonEmptyStripeValue(stripeErr.Error.Message, resp.Status)
		return stripeCheckoutSession{}, fmt.Errorf("stripe checkout session rejected: %s", message)
	}
	var session stripeCheckoutSession
	if err := json.Unmarshal(body, &session); err != nil {
		return stripeCheckoutSession{}, err
	}
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.URL) == "" {
		return stripeCheckoutSession{}, errors.New("stripe checkout response missing session id or URL")
	}
	return session, nil
}

func resolveStripeCheckoutClient(cfg RouterConfig) stripeCheckoutClient {
	if cfg.StripeCheckoutClient != nil {
		return cfg.StripeCheckoutClient
	}
	if strings.TrimSpace(cfg.StripeAPIKey) == "" {
		return nil
	}
	return stripeHTTPCheckoutClient{apiKey: cfg.StripeAPIKey, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func stripeCheckoutPrice(product storage.SubscriberSubscriptionProductRecord, billingPeriod string) string {
	switch strings.TrimSpace(billingPeriod) {
	case "monthly":
		return strings.TrimSpace(product.StripeMonthlyPriceID)
	case "annual", "yearly":
		return strings.TrimSpace(product.StripeAnnualPriceID)
	default:
		return ""
	}
}

func normalizeStripeBillingPeriod(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "monthly", "month":
		return "monthly"
	case "annual", "yearly", "year":
		return "annual"
	default:
		return ""
	}
}

func firstNonEmptyStripeValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
