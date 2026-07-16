package forgebit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateWebhookIncludesSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Fatal("expected Idempotency-Key header")
		}
		var body CreateWebhookParams
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.URL != "https://example.com/webhook" {
			t.Fatalf("unexpected body: %+v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "wh_1", "url": "https://example.com/webhook", "has_secret": true, "secret": "shh"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.CreateWebhook(context.Background(), CreateWebhookParams{URL: "https://example.com/webhook", Events: []string{"license.revoked"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Secret == nil || *result.Data.Secret != "shh" {
		t.Fatalf("expected secret to be present: %+v", result.Data)
	}
}

func TestListWebhooksOmitsSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		nextCursor := "abc123"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "wh_1", "url": "https://example.com/webhook", "has_secret": true}},
			"meta":    map[string]any{"pagination": map[string]any{"per_page": 10, "next_cursor": nextCursor}},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ListWebhooks(context.Background(), ListWebhooksParams{PerPage: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].Secret != nil {
		t.Fatalf("expected no secret on list: %+v", result.Data)
	}
	if !result.Data[0].HasSecret {
		t.Fatal("expected has_secret to be true")
	}
}

func TestShowWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh_1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "wh_1", "url": "https://example.com/webhook"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ShowWebhook(context.Background(), "wh_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "wh_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUpdateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh_1" || r.Method != http.MethodPatch {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "wh_1", "url": "https://example.com/new"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.UpdateWebhook(context.Background(), "wh_1", UpdateWebhookParams{URL: "https://example.com/new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.URL != "https://example.com/new" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDeleteWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh_1" || r.Method != http.MethodDelete {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "Webhook endpoint deleted successfully"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.DeleteWebhook(context.Background(), "wh_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRotateWebhookSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh_1/rotate-secret" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "wh_1", "has_secret": true, "secret": "new-secret"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.RotateWebhookSecret(context.Background(), "wh_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Secret == nil || *result.Data.Secret != "new-secret" {
		t.Fatalf("expected new secret: %+v", result.Data)
	}
}

func TestTestWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/wh_1/test" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["event"] != "webhook.test" {
			t.Fatalf("unexpected body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"indicator": "success", "http_code": 200, "message": "ok"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.TestWebhook(context.Background(), "wh_1", "webhook.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Indicator != "success" || result.Data.HTTPCode == nil || *result.Data.HTTPCode != 200 {
		t.Fatalf("unexpected result: %+v", result.Data)
	}
}

func TestListWebhookLogsBuildsQueryString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("webhook_endpoint_id") != "wh_1" || q.Get("status") != "failed" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "whk_1", "event": "license.revoked", "status": "failed"}},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ListWebhookLogs(context.Background(), ListWebhookLogsParams{EndpointID: "wh_1", Status: "failed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "whk_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReplayWebhookLog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/logs/whk_1/replay" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Webhook replay queued",
			"data":    map[string]any{"id": "whk_1", "status": "pending"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ReplayWebhookLog(context.Background(), "whk_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Status != "pending" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
