package forgebit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListActivationsUnlimitedSeats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/licenses/lic_1/activations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "act_1", "client_identifier": "dev-1", "last_check_in": nil}},
			"meta":    map[string]any{"active_count": 1, "max_allowed": nil, "available_seats": nil},
		})
	}))
	defer server.Close()

	result, err := NewAPIClient(server.URL, "secret").ListActivations(context.Background(), "lic_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].LastCheckIn != nil || result.Meta.MaxAllowed != nil || result.Meta.ActiveCount != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDeactivateActivationSendsClientIDOnlyWhenSet(t *testing.T) {
	var got map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = nil
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "Activation deactivated successfully"})
	}))
	defer server.Close()
	client := NewAPIClient(server.URL, "secret")

	if _, err := client.DeactivateActivation(context.Background(), "act_1", "dev-1"); err != nil || got["client_id"] != "dev-1" {
		t.Fatalf("with client id: body %v, err %v", got, err)
	}
	if _, err := client.DeactivateActivation(context.Background(), "act_1", ""); err != nil || len(got) != 0 {
		t.Fatalf("without client id: body %v, err %v", got, err)
	}
}

func TestDeactivateAllReturnsCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/licenses/lic_1/deactivate-all" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"count": 3}})
	}))
	defer server.Close()

	result, err := NewAPIClient(server.URL, "secret").DeactivateAllActivations(context.Background(), "lic_1")
	if err != nil || result.Data.Count != 3 {
		t.Fatalf("unexpected result: %+v, %v", result, err)
	}
}

func TestDeactivateActivationNotFoundIsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "Activation not found", "errors": map[string]any{"reason": "activation_not_found"}})
	}))
	defer server.Close()

	_, err := NewAPIClient(server.URL, "secret").DeactivateActivation(context.Background(), "nope", "")
	if err == nil || err.Error() != "Activation not found" {
		t.Fatalf("expected API error message, got %v", err)
	}
}
