package forgebit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type WebhookEndpoint struct {
	ID        string   `json:"id"`
	VendorID  string   `json:"vendor_id"`
	Name      *string  `json:"name"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	IsEnabled bool     `json:"is_enabled"`
	HasSecret bool     `json:"has_secret"`
	Secret    *string  `json:"secret,omitempty"`
	CreatedAt string   `json:"created_at"`
}

type WebhookLog struct {
	ID                string         `json:"id"`
	WebhookEndpointID *string        `json:"webhook_endpoint_id"`
	Event             string         `json:"event"`
	Status            string         `json:"status"`
	HTTPCode          *int           `json:"http_code"`
	RetryCount        int            `json:"retry_count"`
	URL               string         `json:"url"`
	DeliveredAt       *string        `json:"delivered_at"`
	Payload           map[string]any `json:"payload"`
	Response          *string        `json:"response"`
	CreatedAt         string         `json:"created_at"`
}

type CreateWebhookParams struct {
	Name      string   `json:"name,omitempty"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	IsEnabled bool     `json:"is_enabled,omitempty"`
}

type CreateWebhookResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    WebhookEndpoint `json:"data"`
}

func (c *APIClient) CreateWebhook(ctx context.Context, params CreateWebhookParams) (CreateWebhookResponse, error) {
	var out CreateWebhookResponse
	err := c.doJSONIdempotent(ctx, http.MethodPost, "/v1/webhooks", params, &out)
	return out, err
}

type ListWebhooksParams struct {
	PerPage int
}

type ListWebhooksResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    []WebhookEndpoint `json:"data"`
	Meta    struct {
		Pagination struct {
			PerPage    int     `json:"per_page"`
			NextCursor *string `json:"next_cursor"`
			PrevCursor *string `json:"prev_cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

func (c *APIClient) ListWebhooks(ctx context.Context, params ListWebhooksParams) (ListWebhooksResponse, error) {
	query := url.Values{}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}

	path := "/v1/webhooks"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out ListWebhooksResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

type ShowWebhookResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    WebhookEndpoint `json:"data"`
}

func (c *APIClient) ShowWebhook(ctx context.Context, id string) (ShowWebhookResponse, error) {
	var out ShowWebhookResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/webhooks/"+id, nil, &out)
	return out, err
}

type UpdateWebhookParams struct {
	Name      *string  `json:"name,omitempty"`
	URL       string   `json:"url,omitempty"`
	Events    []string `json:"events,omitempty"`
	IsEnabled *bool    `json:"is_enabled,omitempty"`
}

type UpdateWebhookResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    WebhookEndpoint `json:"data"`
}

func (c *APIClient) UpdateWebhook(ctx context.Context, id string, params UpdateWebhookParams) (UpdateWebhookResponse, error) {
	var out UpdateWebhookResponse
	err := c.doJSON(ctx, http.MethodPatch, "/v1/webhooks/"+id, params, &out)
	return out, err
}

type DeleteWebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (c *APIClient) DeleteWebhook(ctx context.Context, id string) (DeleteWebhookResponse, error) {
	var out DeleteWebhookResponse
	err := c.doJSON(ctx, http.MethodDelete, "/v1/webhooks/"+id, nil, &out)
	return out, err
}

type RotateWebhookSecretResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    WebhookEndpoint `json:"data"`
}

func (c *APIClient) RotateWebhookSecret(ctx context.Context, id string) (RotateWebhookSecretResponse, error) {
	var out RotateWebhookSecretResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/webhooks/"+id+"/rotate-secret", nil, &out)
	return out, err
}

type TestWebhookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Indicator string         `json:"indicator"`
		HTTPCode  *int           `json:"http_code"`
		Message   string         `json:"message"`
		Response  string         `json:"response"`
		Payload   map[string]any `json:"payload"`
	} `json:"data"`
}

func (c *APIClient) TestWebhook(ctx context.Context, id, event string) (TestWebhookResponse, error) {
	var out TestWebhookResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/webhooks/"+id+"/test", map[string]string{"event": event}, &out)
	return out, err
}

type ListWebhookLogsParams struct {
	EndpointID string
	Status     string
	PerPage    int
}

type ListWebhookLogsResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    []WebhookLog `json:"data"`
	Meta    struct {
		Pagination struct {
			PerPage    int     `json:"per_page"`
			NextCursor *string `json:"next_cursor"`
			PrevCursor *string `json:"prev_cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

func (c *APIClient) ListWebhookLogs(ctx context.Context, params ListWebhookLogsParams) (ListWebhookLogsResponse, error) {
	query := url.Values{}
	if params.EndpointID != "" {
		query.Set("webhook_endpoint_id", params.EndpointID)
	}
	if params.Status != "" {
		query.Set("status", params.Status)
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}

	path := "/v1/webhooks/logs"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out ListWebhookLogsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

type ReplayWebhookLogResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Data    WebhookLog `json:"data"`
}

func (c *APIClient) ReplayWebhookLog(ctx context.Context, id string) (ReplayWebhookLogResponse, error) {
	var out ReplayWebhookLogResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/webhooks/logs/"+id+"/replay", nil, &out)
	return out, err
}
