package forgebit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
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

type UploadedLicenseFile struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

// Replaces the built-in generator's file for this license. The server accepts
// .lic, .json, .enc, or .xml up to 5 MB, so buffering in memory is fine
func (c *APIClient) UploadLicenseFile(ctx context.Context, licenseID, filename string, content io.Reader) (UploadedLicenseFile, error) {
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return UploadedLicenseFile{}, err
	}
	if _, err := io.Copy(part, content); err != nil {
		return UploadedLicenseFile{}, err
	}
	if err := form.Close(); err != nil {
		return UploadedLicenseFile{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/licenses/"+licenseID+"/upload-license", &buf)
	if err != nil {
		return UploadedLicenseFile{}, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())

	var out UploadedLicenseFile
	err = c.send(req, &out)
	return out, err
}

func (c *APIClient) DeleteUploadedLicenseFile(ctx context.Context, licenseID string) (MessageResponse, error) {
	var out MessageResponse
	err := c.doJSON(ctx, http.MethodDelete, "/v1/licenses/"+licenseID+"/uploaded-license", nil, &out)
	return out, err
}
