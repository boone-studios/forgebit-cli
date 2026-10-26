package forgebit

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"path/filepath"
)

// Body is the raw signed file, not JSON. filename comes from Content-Disposition,
// reduced to a bare name so a hostile header can't point outside the target dir
func (c *APIClient) DownloadLicenseFile(ctx context.Context, licenseID string) (body io.ReadCloser, filename string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/licenses/"+licenseID+"/download", nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		var failure struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &failure)
		return nil, "", &APIError{StatusCode: resp.StatusCode, Message: failure.Message, Body: raw}
	}

	if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); err == nil {
		if name := filepath.Base(params["filename"]); name != "." && name != ".." && name != string(filepath.Separator) {
			filename = name
		}
	}
	return resp.Body, filename, nil
}

// No success envelope on this endpoint
type LicenseFileLink struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

func (c *APIClient) LicenseFileLink(ctx context.Context, licenseID string) (LicenseFileLink, error) {
	var out LicenseFileLink
	err := c.doJSON(ctx, http.MethodPost, "/v1/licenses/"+licenseID+"/generate-token", nil, &out)
	return out, err
}

type ProductDownloadLink struct {
	Version     string `json:"version"`
	Channel     string `json:"channel"`
	ExpiresAt   string `json:"expires_at"`
	DownloadURL string `json:"download_url"`
}

// channel is stable or beta; empty leaves it to the server default (stable)
func (c *APIClient) ProductDownloadLink(ctx context.Context, licenseID, channel string) (Response[ProductDownloadLink], error) {
	body := map[string]string{}
	if channel != "" {
		body["channel"] = channel
	}

	var out Response[ProductDownloadLink]
	err := c.doJSON(ctx, http.MethodPost, "/v1/licenses/"+licenseID+"/download-link", body, &out)
	return out, err
}
