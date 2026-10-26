package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
)

// Fake API + fake storage in one server; the version settles to finalStatus after one pending poll
func fakeReleaseServer(t *testing.T, finalStatus string) (*httptest.Server, *[]byte, *string) {
	var uploaded []byte
	var sentChecksum string
	polls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/products/prod_1/versions":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
				"id": "ver_1", "version": "1.2.0", "status": "pending",
				"upload": map[string]any{"url": server.URL + "/storage/ver_1.zip", "headers": map[string]string{"Content-Type": "application/zip"}},
			}})
		case "PUT /storage/ver_1.zip":
			if r.Header.Get("Authorization") != "" {
				t.Fatal("API token leaked to storage upload")
			}
			if r.Header.Get("Content-Type") != "application/zip" || r.ContentLength < 0 {
				t.Fatalf("bad upload request: content-type %q, length %d", r.Header.Get("Content-Type"), r.ContentLength)
			}
			uploaded, _ = io.ReadAll(r.Body)
		case "POST /v1/products/prod_1/versions/ver_1/complete":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentChecksum = body["checksum"]
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "ver_1", "status": "pending"}})
		case "GET /v1/products/prod_1/versions/ver_1":
			polls++
			status := "pending"
			if polls > 1 {
				status = finalStatus
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"id": "ver_1", "version": "1.2.0", "channel": "stable", "status": status}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server, &uploaded, &sentChecksum
}

func writeArtifact(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "plugin.zip")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReleaseVersionUploadsAndWaitsForReady(t *testing.T) {
	server, uploaded, checksum := fakeReleaseServer(t, "ready")
	client := forgebit.NewAPIClient(server.URL, "secret")

	result, err := releaseVersion(context.Background(), client, "prod_1", writeArtifact(t), forgebit.CreateVersionParams{Version: "1.2.0"}, true, 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "ready" {
		t.Fatalf("expected ready, got %+v", result)
	}
	if string(*uploaded) != "hello" {
		t.Fatalf("unexpected upload body: %q", *uploaded)
	}
	// sha256("hello")
	if *checksum != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected checksum: %q", *checksum)
	}
}

func TestReleaseVersionReportsFailure(t *testing.T) {
	server, _, _ := fakeReleaseServer(t, "failed")
	client := forgebit.NewAPIClient(server.URL, "secret")

	_, err := releaseVersion(context.Background(), client, "prod_1", writeArtifact(t), forgebit.CreateVersionParams{Version: "1.2.0"}, true, 0, io.Discard)
	if err == nil {
		t.Fatal("expected an error for a failed version")
	}
}

func TestReleaseVersionNoWaitSkipsPolling(t *testing.T) {
	server, _, _ := fakeReleaseServer(t, "ready")
	client := forgebit.NewAPIClient(server.URL, "secret")

	result, err := releaseVersion(context.Background(), client, "prod_1", writeArtifact(t), forgebit.CreateVersionParams{Version: "1.2.0"}, false, 0, io.Discard)
	if err != nil || result.Status != "pending" {
		t.Fatalf("expected pending without polling, got %+v, %v", result, err)
	}
}
