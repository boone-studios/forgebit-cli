package forgebit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownloadLicenseFileSanitizesFilename(t *testing.T) {
	cases := map[string]string{
		`attachment; filename="license-lic_1.lic"`: "license-lic_1.lic",
		`attachment; filename="../../.bashrc"`:     ".bashrc",
		`attachment; filename="/etc/passwd"`:       "passwd",
		`attachment; filename=".."`:                "",
		``:                                         "",
	}
	for header, want := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer secret" {
				t.Fatal("missing auth")
			}
			if header != "" {
				w.Header().Set("Content-Disposition", header)
			}
			_, _ = w.Write([]byte{0x00, 0x01, 0x02})
		}))

		body, name, err := NewAPIClient(server.URL, "secret").DownloadLicenseFile(context.Background(), "lic_1")
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", header, err)
		}
		data, _ := io.ReadAll(body)
		body.Close()
		server.Close()

		if name != want || len(data) != 3 {
			t.Errorf("%q: got name %q (%d bytes), want %q", header, name, len(data), want)
		}
	}
}

func TestDownloadLicenseFileErrorIsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "This action is unauthorized."})
	}))
	defer server.Close()

	_, _, err := NewAPIClient(server.URL, "secret").DownloadLicenseFile(context.Background(), "lic_1")
	if err == nil || err.Error() != "This action is unauthorized." {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestProductDownloadLinkSendsChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/v1/licenses/lic_1/download-link" || body["channel"] != "beta" {
			t.Fatalf("unexpected request: %s %v", r.URL.Path, body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"version": "1.2.0", "download_url": "https://storage/x.zip"}})
	}))
	defer server.Close()

	result, err := NewAPIClient(server.URL, "secret").ProductDownloadLink(context.Background(), "lic_1", "beta")
	if err != nil || result.Data.DownloadURL != "https://storage/x.zip" {
		t.Fatalf("unexpected result: %+v, %v", result, err)
	}
}
