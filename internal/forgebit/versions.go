package forgebit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ProductVersion struct {
	ID             string         `json:"id"`
	ProductID      string         `json:"product_id"`
	Version        string         `json:"version"`
	Channel        string         `json:"channel"`
	Status         string         `json:"status"`
	IsCurrent      bool           `json:"is_current"`
	IsSecurity     bool           `json:"is_security"`
	RolloutPercent int            `json:"rollout_percent"`
	ReleasedAt     *string        `json:"released_at,omitempty"`
	Checksum       *string        `json:"checksum,omitempty"`
	DownloadCount  int            `json:"download_count"`
	Upload         *VersionUpload `json:"upload,omitempty"`
}

// Only returned on create; the URL is presigned and short-lived
type VersionUpload struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
}

type CreateVersionParams struct {
	Version          string `json:"version"`
	Channel          string `json:"channel,omitempty"`
	RequiresPlatform string `json:"requires_platform,omitempty"`
	TestedUpTo       string `json:"tested_up_to,omitempty"`
	RequiresRuntime  string `json:"requires_runtime,omitempty"`
	Changelog        string `json:"changelog,omitempty"`
	IsSecurity       bool   `json:"is_security,omitempty"`
	RolloutPercent   *int   `json:"rollout_percent,omitempty"` // nil = server default (100)
}

func (c *APIClient) CreateProductVersion(ctx context.Context, productID string, params CreateVersionParams) (Response[ProductVersion], error) {
	var out Response[ProductVersion]
	err := c.doJSONIdempotent(ctx, http.MethodPost, "/v1/products/"+productID+"/versions", params, &out)
	return out, err
}

// The upload goes straight to storage, so it must never carry the API token,
// and it has no client timeout since artifacts can be large (ctx bounds it)
func UploadVersionArtifact(ctx context.Context, upload VersionUpload, body io.Reader, size int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.URL, body)
	if err != nil {
		return err
	}
	req.ContentLength = size // Presigned PUTs reject chunked bodies
	for k, v := range upload.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// *url.Error embeds the full URL, and the presigned query string is a credential
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("uploading artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("uploading artifact: storage returned %s", resp.Status)
	}
	return nil
}

// checksum is the artifact's SHA-256 hex; the server fails the version on mismatch
func (c *APIClient) CompleteProductVersion(ctx context.Context, productID, versionID, checksum string) (Response[ProductVersion], error) {
	body := map[string]string{}
	if checksum != "" {
		body["checksum"] = checksum
	}

	var out Response[ProductVersion]
	err := c.doJSON(ctx, http.MethodPost, "/v1/products/"+productID+"/versions/"+versionID+"/complete", body, &out)
	return out, err
}

func (c *APIClient) ShowProductVersion(ctx context.Context, productID, versionID string) (Response[ProductVersion], error) {
	var out Response[ProductVersion]
	err := c.doJSON(ctx, http.MethodGet, "/v1/products/"+productID+"/versions/"+versionID, nil, &out)
	return out, err
}
