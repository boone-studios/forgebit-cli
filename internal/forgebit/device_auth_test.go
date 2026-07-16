package forgebit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestAuthorization(t *testing.T) {
	var handler http.HandlerFunc
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/device/cli/authorizations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Device authorization created",
			"data": map[string]any{
				"device_code":               "abc123",
				"user_code":                 "WDJB-MJHT",
				"verification_uri":          server.URL + "/cli/authorize",
				"verification_uri_complete": server.URL + "/cli/authorize?user_code=WDJB-MJHT",
				"expires_in":                600,
				"interval":                  1,
			},
		})
	}

	client := NewDeviceAuthClient(server.URL)
	auth, err := client.RequestAuthorization(context.Background(), "forgebit-cli")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.DeviceCode != "abc123" || auth.UserCode != "WDJB-MJHT" {
		t.Fatalf("unexpected auth: %+v", auth)
	}
	if auth.ExpiresIn != 600 || auth.Interval != 1 {
		t.Fatalf("unexpected timing fields: %+v", auth)
	}
}

func TestPollStatuses(t *testing.T) {
	cases := []struct {
		name       string
		httpStatus int
		body       map[string]any
		want       PollStatus
	}{
		{
			name:       "pending",
			httpStatus: http.StatusAccepted,
			body:       map[string]any{"success": true, "data": map[string]any{"status": "authorization_pending"}},
			want:       PollStatusPending,
		},
		{
			name:       "approved",
			httpStatus: http.StatusOK,
			body:       map[string]any{"success": true, "data": map[string]any{"status": "approved", "api_key_id": "01J", "token": "fb_secret"}},
			want:       PollStatusApproved,
		},
		{
			name:       "denied",
			httpStatus: http.StatusGone,
			body:       map[string]any{"success": false, "errors": map[string]any{"status": "denied"}},
			want:       PollStatusDenied,
		},
		{
			name:       "expired",
			httpStatus: http.StatusGone,
			body:       map[string]any{"success": false, "errors": map[string]any{"status": "expired"}},
			want:       PollStatusExpired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.httpStatus)
				_ = json.NewEncoder(w).Encode(tc.body)
			}))
			defer server.Close()

			client := NewDeviceAuthClient(server.URL)
			result, err := client.Poll(context.Background(), "abc123")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != tc.want {
				t.Fatalf("got status %q, want %q", result.Status, tc.want)
			}
			if tc.want == PollStatusApproved && result.Token == "" {
				t.Fatalf("expected token to be set on approval")
			}
		})
	}
}

func TestPollRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewDeviceAuthClient(server.URL)
	_, err := client.Poll(context.Background(), "abc123")

	var rateLimited *RateLimitedError
	if err == nil {
		t.Fatal("expected an error")
	}
	if !isRateLimitedError(err, &rateLimited) {
		t.Fatalf("expected RateLimitedError, got %T: %v", err, err)
	}
	if rateLimited.RetryAfter != "3" {
		t.Fatalf("unexpected Retry-After: %q", rateLimited.RetryAfter)
	}
}

func isRateLimitedError(err error, target **RateLimitedError) bool {
	if rl, ok := err.(*RateLimitedError); ok {
		*target = rl
		return true
	}
	return false
}

func TestPollUnknownDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "Unknown or already-used device code"})
	}))
	defer server.Close()

	client := NewDeviceAuthClient(server.URL)
	_, err := client.Poll(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected an error for unknown device code")
	}
}

// Ensures the polling contract's timing fields survive a round trip fast
// enough to be usable in a tight test loop.
func TestPollTimingIsFast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"status": "authorization_pending"}})
	}))
	defer server.Close()

	client := NewDeviceAuthClient(server.URL)
	start := time.Now()
	if _, err := client.Poll(context.Background(), "abc123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("poll took too long: %s", elapsed)
	}
}
