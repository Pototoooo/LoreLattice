package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("MeterForge returned HTTP %d: %s", e.Status, e.Body)
}

type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode MeterForge request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create MeterForge request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("call MeterForge: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read MeterForge response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode MeterForge response: %w", err)
		}
	}
	return nil
}

type meterResource struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Aggregation string            `json:"aggregation"`
	EventType   string            `json:"eventType"`
	Value       string            `json:"valueProperty"`
	GroupBy     map[string]string `json:"groupBy"`
}

type featureResource struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type planResource struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Version   int    `json:"version"`
	Status    string `json:"status"`
	PlanID    string `json:"plan_id"`
	CreatedAt string `json:"created_at"`
}

type customerResource struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type subscriptionResource struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	PlanID        string `json:"plan_id"`
	Status        string `json:"status"`
	BillingAnchor string `json:"billing_anchor"`
}

type listResponse[T any] struct {
	Data []T `json:"data"`
}

type entitlementValue struct {
	HasAccess                 bool    `json:"hasAccess"`
	Balance                   float64 `json:"balance"`
	Usage                     float64 `json:"usage"`
	Overage                   float64 `json:"overage"`
	TotalAvailableGrantAmount float64 `json:"totalAvailableGrantAmount"`
}

type creditBalanceResponse struct {
	Balances []struct {
		Currency string      `json:"currency"`
		Live     json.Number `json:"live"`
		Settled  json.Number `json:"settled"`
		Pending  json.Number `json:"pending"`
	} `json:"balances"`
}

func (c *Client) ListMeters(ctx context.Context) ([]meterResource, error) {
	var meters []meterResource
	err := c.do(ctx, http.MethodGet, "/api/v1/meters", nil, &meters)
	return meters, err
}

func (c *Client) CreateMeter(ctx context.Context, def FeatureDefinition) (meterResource, error) {
	var meter meterResource
	err := c.do(ctx, http.MethodPost, "/api/v1/meters", map[string]any{
		"slug": def.MeterSlug, "name": def.Name, "description": "LoreLattice dedicated usage meter",
		"aggregation": "SUM", "eventType": def.EventType, "valueProperty": def.ValueField,
		"groupBy": map[string]string{"model": "$.model", "provider": "$.provider", "operation": "$.operation"},
	}, &meter)
	return meter, err
}

func (c *Client) ListFeatures(ctx context.Context) ([]featureResource, error) {
	var response listResponse[featureResource]
	err := c.do(ctx, http.MethodGet, "/api/v3/meterforge/features?page_size=100", nil, &response)
	return response.Data, err
}

func (c *Client) CreateFeature(ctx context.Context, def FeatureDefinition, meterID string) (featureResource, error) {
	var feature featureResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/features", map[string]any{
		"key": string(def.Key), "name": def.Name,
		"description": "LoreLattice dedicated billing feature",
		"labels":      map[string]string{"application": "lorelattice", "catalog_version": "v1"},
		"meter":       map[string]string{"id": meterID},
		"unit_cost":   map[string]any{"type": "manual", "amount": strconv.FormatFloat(def.UnitPrice, 'f', -1, 64)},
	}, &feature)
	return feature, err
}

func (c *Client) ListPlans(ctx context.Context) ([]planResource, error) {
	var response listResponse[planResource]
	err := c.do(ctx, http.MethodGet, "/api/v3/meterforge/plans?page_size=100", nil, &response)
	return response.Data, err
}

func (c *Client) CreatePlan(ctx context.Context, key, name string, limits map[FeatureKey]float64, features map[FeatureKey]featureResource) (planResource, error) {
	rateCards := make([]map[string]any, 0, len(FeatureDefinitions))
	for _, def := range orderedFeatureDefinitions() {
		rateCards = append(rateCards, map[string]any{
			"key": string(def.Key), "name": def.Name, "billing_cadence": "P1M",
			"feature": map[string]string{"id": features[def.Key].ID},
			"entitlement": map[string]any{
				"type": "metered", "is_soft_limit": false, "limit": limits[def.Key], "usage_period": "P1M",
			},
			"price":        map[string]any{"type": "unit", "amount": strconv.FormatFloat(def.UnitPrice, 'f', -1, 64)},
			"payment_term": "in_arrears",
		})
	}
	var plan planResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/plans", map[string]any{
		"key": key, "name": name, "description": "LoreLattice " + name + " catalog v1",
		"currency": "USD", "billing_cadence": "P1M", "pro_rating_enabled": false,
		"labels": map[string]string{"application": "lorelattice", "catalog_version": "v1"},
		"phases": []map[string]any{{"key": "default", "name": "Default", "rate_cards": rateCards}},
	}, &plan)
	return plan, err
}

func (c *Client) PublishPlan(ctx context.Context, planID string) error {
	return c.do(ctx, http.MethodPost, "/api/v3/meterforge/plans/"+url.PathEscape(planID)+"/publish", nil, nil)
}

func (c *Client) FindCustomerByKey(ctx context.Context, key string) (*customerResource, error) {
	path := "/api/v3/meterforge/customers?filter[key][eq]=" + url.QueryEscape(key)
	var response listResponse[customerResource]
	if err := c.do(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	for _, customer := range response.Data {
		if customer.Key == key {
			found := customer
			return &found, nil
		}
	}
	return nil, nil
}

func (c *Client) CreateCustomer(ctx context.Context, key, name, subject string) (customerResource, error) {
	var customer customerResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/customers", map[string]any{
		"key": key, "name": name, "currency": "USD",
		"description":       "LoreLattice workspace provisioned automatically",
		"labels":            map[string]string{"application": "lorelattice"},
		"usage_attribution": map[string]any{"subject_keys": []string{subject}},
	}, &customer)
	return customer, err
}

func (c *Client) ListSubscriptions(ctx context.Context, customerID string) ([]subscriptionResource, error) {
	path := "/api/v3/meterforge/subscriptions?filter[customer_id][eq]=" + url.QueryEscape(customerID) + "&page_size=100"
	var response listResponse[subscriptionResource]
	err := c.do(ctx, http.MethodGet, path, nil, &response)
	return response.Data, err
}

func (c *Client) CreateSubscription(ctx context.Context, customerKey, planKey string) (subscriptionResource, error) {
	var subscription subscriptionResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/subscriptions", map[string]any{
		"customer":        map[string]string{"key": customerKey},
		"plan":            map[string]any{"key": planKey, "version": 1},
		"settlement_mode": "credit_only",
		"labels":          map[string]string{"application": "lorelattice"},
	}, &subscription)
	return subscription, err
}

func (c *Client) ChangeSubscription(ctx context.Context, subscriptionID, customerKey, planKey string) (subscriptionResource, error) {
	var response struct {
		Current subscriptionResource `json:"current"`
		Next    subscriptionResource `json:"next"`
	}
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/subscriptions/"+url.PathEscape(subscriptionID)+"/change", map[string]any{
		"customer":        map[string]string{"key": customerKey},
		"plan":            map[string]any{"key": planKey, "version": 1},
		"settlement_mode": "credit_only", "timing": "immediate",
		"labels": map[string]string{"application": "lorelattice"},
	}, &response)
	return response.Next, err
}

func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string, timing any) (subscriptionResource, error) {
	var subscription subscriptionResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/subscriptions/"+url.PathEscape(subscriptionID)+"/cancel",
		map[string]any{"timing": timing}, &subscription)
	return subscription, err
}

func (c *Client) UnscheduleCancel(ctx context.Context, subscriptionID string) (subscriptionResource, error) {
	var subscription subscriptionResource
	err := c.do(ctx, http.MethodPost, "/api/v3/meterforge/subscriptions/"+url.PathEscape(subscriptionID)+"/unschedule-cancelation",
		nil, &subscription)
	return subscription, err
}

type creditGrantResource struct {
	ID string `json:"id"`
}

func (c *Client) CreateCreditGrant(ctx context.Context, customerID, key, name string, amount float64, expiresAfter string, features []string) (creditGrantResource, error) {
	body := map[string]any{
		"key": key, "name": name, "funding_method": "none", "currency": "USD",
		"amount": strconv.FormatFloat(amount, 'f', 8, 64), "priority": 10,
		"labels": map[string]string{"application": "lorelattice"},
	}
	if expiresAfter != "" {
		body["expires_after"] = expiresAfter
	}
	if len(features) > 0 {
		body["filters"] = map[string]any{"features": features}
	}
	var grant creditGrantResource
	err := c.do(ctx, http.MethodPost,
		"/api/v3/meterforge/customers/"+url.PathEscape(customerID)+"/credits/grants", body, &grant)
	return grant, err
}

func (c *Client) EntitlementValue(ctx context.Context, customerKey string, feature FeatureKey) (entitlementValue, error) {
	var value entitlementValue
	err := c.do(ctx, http.MethodGet,
		"/api/v1/customers/"+url.PathEscape(customerKey)+"/entitlements/"+url.PathEscape(string(feature))+"/value",
		nil, &value)
	return value, err
}

func (c *Client) CreditBalance(ctx context.Context, customerID string) (float64, error) {
	var response creditBalanceResponse
	if err := c.do(ctx, http.MethodGet,
		"/api/v3/meterforge/customers/"+url.PathEscape(customerID)+"/credits/balance", nil, &response); err != nil {
		return 0, err
	}
	for _, balance := range response.Balances {
		if balance.Currency == "USD" {
			live, err := strconv.ParseFloat(balance.Live.String(), 64)
			if err != nil {
				return 0, fmt.Errorf("parse MeterForge USD balance: %w", err)
			}
			pending, err := strconv.ParseFloat(balance.Pending.String(), 64)
			if err != nil {
				return 0, fmt.Errorf("parse MeterForge pending USD balance: %w", err)
			}
			// A freshly-created Sandbox grant can spend a short time in the
			// pending ledger state. It is already committed and idempotent, so
			// treat it as available for LoreLattice's pre-call gate.
			return live + pending, nil
		}
	}
	return 0, nil
}

func (c *Client) SendEvent(ctx context.Context, eventID string, payload json.RawMessage) error {
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}
	var reader io.Reader
	reader = bytes.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/events", reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/cloudevents+json")
	req.Header.Set("Idempotency-Key", eventID)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	return nil
}

func (c *Client) ListInvoices(ctx context.Context, customerID string) (json.RawMessage, error) {
	path := "/api/v3/meterforge/billing/invoices?filter[customer_id][eq]=" + url.QueryEscape(customerID) + "&page_size=100"
	var raw json.RawMessage
	err := c.do(ctx, http.MethodGet, path, nil, &raw)
	return raw, err
}

func isHTTPStatus(err error, status int) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.Status == status
}

func orderedFeatureDefinitions() []FeatureDefinition {
	return []FeatureDefinition{
		FeatureDefinitions[FeatureLLMTokens],
		FeatureDefinitions[FeatureEmbeddingTokens],
		FeatureDefinitions[FeatureRerankTokens],
		FeatureDefinitions[FeatureASRSeconds],
	}
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}
