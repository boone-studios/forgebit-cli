package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
)

func TestPollForResultApproved(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"status": "authorization_pending"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
			"status": "approved", "api_key_id": "01J", "token": "fb_secret",
			"vendor_id": "vnd_1", "vendor_name": "Acme",
		}})
	}))
	defer server.Close()

	client := forgebit.NewDeviceAuthClient(server.URL)
	auth := forgebit.DeviceAuthorization{DeviceCode: "abc123", ExpiresIn: 5, Interval: 1}

	result, err := pollForResult(context.Background(), client, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "fb_secret" {
		t.Fatalf("unexpected token: %q", result.Token)
	}
	if result.VendorID != "vnd_1" || result.VendorName != "Acme" {
		t.Fatalf("expected vendor fields to be threaded through, got %+v", result)
	}
	if attempts < 2 {
		t.Fatalf("expected at least 2 polls, got %d", attempts)
	}
}

func TestPollForResultDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": map[string]any{"status": "denied"}})
	}))
	defer server.Close()

	client := forgebit.NewDeviceAuthClient(server.URL)
	auth := forgebit.DeviceAuthorization{DeviceCode: "abc123", ExpiresIn: 5, Interval: 1}

	_, err := pollForResult(context.Background(), client, auth)
	if err == nil {
		t.Fatal("expected an error for denied authorization")
	}
}

func TestPollForResultExpires(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"status": "authorization_pending"}})
	}))
	defer server.Close()

	client := forgebit.NewDeviceAuthClient(server.URL)
	auth := forgebit.DeviceAuthorization{DeviceCode: "abc123", ExpiresIn: 1, Interval: 5}

	_, err := pollForResult(context.Background(), client, auth)
	if err == nil {
		t.Fatal("expected a timeout error when the context deadline is exceeded")
	}
}
