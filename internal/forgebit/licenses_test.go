package forgebit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIssueLicense(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/licenses/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("missing/incorrect Authorization header: %q", r.Header.Get("Authorization"))
		}
		var body IssueLicenseParams
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.CustomerEmail != "customer@example.com" {
			t.Fatalf("unexpected body: %+v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "License issued successfully",
			"license": map[string]any{"id": "lic_1", "vendor_id": "vnd_1", "tier": "pro", "license_type": "jwt"},
			"key":     "eyFAKE.eyFAKE.sig",
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.IssueLicense(context.Background(), IssueLicenseParams{
		CustomerEmail: "customer@example.com", ProductID: "prod_1", Tier: "pro", LicenseDuration: "trial",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Key != "eyFAKE.eyFAKE.sig" || result.License.ID != "lic_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListLicensesBuildsQueryString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("product_id") != "prod_1" || q.Get("is_active") != "true" || q.Get("per_page") != "10" || q.Get("cursor") != "c1" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		nextCursor := "abc123"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "lic_1", "tier": "pro"}},
			"meta":    map[string]any{"pagination": map[string]any{"per_page": 10, "next_cursor": nextCursor}},
		})
	}))
	defer server.Close()

	active := true
	client := NewAPIClient(server.URL, "secret")
	result, err := client.ListLicenses(context.Background(), ListLicensesParams{
		ProductID: "prod_1", IsActive: &active, PerPage: 10, Cursor: "c1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "lic_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Meta.Pagination.NextCursor == nil || *result.Meta.Pagination.NextCursor != "abc123" {
		t.Fatalf("unexpected pagination meta: %+v", result.Meta.Pagination)
	}
}

func TestVerifyLicenseOnlineValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "valid": true, "tier": "pro", "environment": "live",
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.VerifyLicenseOnline(context.Background(), VerifyLicenseParams{Key: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid || result.Tier != "pro" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVerifyLicenseOnlineInvalidIsNotAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		reason := "expired"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "valid": false, "reason": reason,
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.VerifyLicenseOnline(context.Background(), VerifyLicenseParams{Key: "abc"})
	if err != nil {
		t.Fatalf("expected no error for a well-formed invalid result, got: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected Valid=false")
	}
	if result.Reason == nil || *result.Reason != "expired" {
		t.Fatalf("unexpected reason: %+v", result.Reason)
	}
}

func TestVerifyLicenseOnlineHardFailureIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Invalid API key"})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "bad-token")
	_, err := client.VerifyLicenseOnline(context.Background(), VerifyLicenseParams{Key: "abc"})
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}
}

func TestRevokeLicenseSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/licenses/lic_1/revoke" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		revokedAt := "2026-07-15T00:00:00Z"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true, "message": "License revoked successfully",
			"data": map[string]any{"revoked_at": revokedAt, "sessions_deactivated": 2},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.RevokeLicense(context.Background(), "lic_1", "chargeback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.RevokedAt == nil || result.Data.SessionsDeactivated != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRevokeLicenseFailureReasonIsPopulated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false, "message": "Cannot revoke", "errors": map[string]any{"reason": "already_revoked"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.RevokeLicense(context.Background(), "lic_1", "")
	if err == nil {
		t.Fatal("expected an error for a 400 response")
	}
	if result.Errors == nil || result.Errors.Reason != "already_revoked" {
		t.Fatalf("expected Errors to still be populated from the failure body, got: %+v", result)
	}
}

func TestVendorPublicKeyPEM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/vendors/vnd_1/keys/kid_1.pem" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("-----BEGIN PUBLIC KEY-----\nZmFrZWtleQ==\n-----END PUBLIC KEY-----\n"))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	pem, err := client.VendorPublicKeyPEM(context.Background(), "vnd_1", "kid_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pem == "" {
		t.Fatal("expected a non-empty PEM")
	}
}

func TestExpiringAndExpiredLicensesSendWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := map[string]string{"/v1/licenses/expiring": "14", "/v1/licenses/expired": ""}
		days, ok := want[r.URL.Path]
		if !ok || r.URL.Query().Get("days") != days {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "lic_1", "expires_at": "2027-02-14T23:59:59.000000Z"}},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	expiring, err := client.ExpiringLicenses(context.Background(), 14)
	if err != nil || len(expiring.Data) != 1 || expiring.Data[0].ID != "lic_1" {
		t.Fatalf("expiring: %+v, %v", expiring, err)
	}
	if _, err := client.ExpiredLicenses(context.Background(), 0); err != nil {
		t.Fatalf("expired: %v", err)
	}
}

func TestLookupCustomerLicensesSendsEmailInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/products/prod_1/licenses/lookup" || r.URL.RawQuery != "" || body["customer_email"] != "jane@example.com" {
			t.Fatalf("unexpected request: %s %s body %v", r.Method, r.URL, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": []map[string]any{
			{"id": "lic_1", "tier": "pro", "status": "active", "is_floating": true, "max_concurrent_users": 2, "key": "KEY-1"},
		}})
	}))
	defer server.Close()

	result, err := NewAPIClient(server.URL, "secret").LookupCustomerLicenses(context.Background(), "prod_1", "jane@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.Data[0]
	if got.ID != "lic_1" || got.Tier != "pro" || got.Status != "active" || *got.MaxConcurrentUsers != 2 || *got.Key != "KEY-1" {
		t.Fatalf("unexpected result: %+v", got)
	}
}
