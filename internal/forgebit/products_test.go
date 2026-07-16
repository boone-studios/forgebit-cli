package forgebit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/products" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Fatal("expected Idempotency-Key header")
		}
		var body CreateProductParams
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "Test Product" {
			t.Fatalf("unexpected body: %+v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"message": "Product created successfully",
			"data":    map[string]any{"id": "prod_1", "vendor_id": "vnd_1", "name": "Test Product", "slug": "test-product"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.CreateProduct(context.Background(), CreateProductParams{Name: "Test Product"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "prod_1" || result.Data.Slug != "test-product" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListProductsBuildsQueryString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("search") != "test" || q.Get("per_page") != "10" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		nextCursor := "abc123"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    []map[string]any{{"id": "prod_1", "name": "Test Product"}},
			"meta":    map[string]any{"pagination": map[string]any{"per_page": 10, "next_cursor": nextCursor}},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ListProducts(context.Background(), ListProductsParams{Search: "test", PerPage: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].ID != "prod_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Meta.Pagination.NextCursor == nil || *result.Meta.Pagination.NextCursor != "abc123" {
		t.Fatalf("unexpected pagination meta: %+v", result.Meta.Pagination)
	}
}

func TestShowProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/products/prod_1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "prod_1", "name": "Test Product"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.ShowProduct(context.Background(), "prod_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.ID != "prod_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUpdateProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/products/prod_1" || r.Method != http.MethodPatch {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "" {
			t.Fatal("did not expect an Idempotency-Key header on update")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"id": "prod_1", "name": "Updated Name"},
		})
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")
	result, err := client.UpdateProduct(context.Background(), "prod_1", UpdateProductParams{Name: "Updated Name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.Name != "Updated Name" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestArchiveAndRestoreProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/products/prod_1/archive":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "prod_1", "archived_at": "2026-01-01T00:00:00Z"}})
		case "/v1/products/prod_1/restore":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "prod_1", "archived_at": nil}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "secret")

	archived, err := client.ArchiveProduct(context.Background(), "prod_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if archived.Data.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}

	restored, err := client.RestoreProduct(context.Background(), "prod_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restored.Data.ArchivedAt != nil {
		t.Fatal("expected archived_at to be cleared")
	}
}
