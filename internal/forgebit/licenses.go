package forgebit

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Body holds the raw failure response since its shape varies per endpoint
type APIError struct {
	StatusCode int
	Message    string
	Body       json.RawMessage
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("unexpected status: %d", e.StatusCode)
}

func (c *APIClient) doJSON(ctx context.Context, method, path string, body, out any) error {
	return c.doJSONWithHeaders(ctx, method, path, body, out, nil)
}

// Issue and renew require an Idempotency-Key header, the rest don't
func (c *APIClient) doJSONIdempotent(ctx context.Context, method, path string, body, out any) error {
	return c.doJSONWithHeaders(ctx, method, path, body, out, map[string]string{
		"Idempotency-Key": randomIdempotencyKey(),
	})
}

func randomIdempotencyKey() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func (c *APIClient) doJSONWithHeaders(ctx context.Context, method, path string, body, out any, headers map[string]string) error {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("reaching %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	if out != nil && len(raw) > 0 {
		// Only a bad 2xx body is a real bug, failure shapes vary per endpoint
		if unmarshalErr := json.Unmarshal(raw, out); unmarshalErr != nil && success {
			return fmt.Errorf("decoding response: %w", unmarshalErr)
		}
	}

	if !success {
		var failure struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &failure)
		return &APIError{StatusCode: resp.StatusCode, Message: failure.Message, Body: raw}
	}

	return nil
}

type License struct {
	ID            string  `json:"id"`
	VendorID      string  `json:"vendor_id"`
	ProductID     string  `json:"product_id"`
	LicenseType   string  `json:"license_type"`
	LicenseTypeID string  `json:"license_type_id"`
	Tier          string  `json:"tier"`
	Environment   string  `json:"environment"`
	IsOffline     bool    `json:"is_offline"`
	ExpiresAt     *string `json:"expires_at"`
	CreatedAt     string  `json:"created_at"`
}

type IssueLicenseParams struct {
	CustomerEmail      string         `json:"customer_email"`
	CustomerName       string         `json:"customer_name,omitempty"`
	ProductID          string         `json:"product_id"`
	Tier               string         `json:"tier"`
	LicenseType        string         `json:"license_type,omitempty"`
	LicenseTypeID      string         `json:"license_type_id,omitempty"`
	LicenseDuration    string         `json:"license_duration_type"`
	ExpiresAt          string         `json:"expires_at,omitempty"`
	IsOffline          bool           `json:"is_offline,omitempty"`
	IsFloating         bool           `json:"is_floating,omitempty"`
	MaxConcurrentUsers int            `json:"max_concurrent_users,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	Features           map[string]any `json:"features,omitempty"`
	OrderReference     string         `json:"order_reference,omitempty"`
}

type IssueLicenseResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	License License `json:"license"`
	Key     string  `json:"key"`
}

func (c *APIClient) IssueLicense(ctx context.Context, params IssueLicenseParams) (IssueLicenseResponse, error) {
	var out IssueLicenseResponse
	err := c.doJSONIdempotent(ctx, http.MethodPost, "/v1/licenses/issue", params, &out)
	return out, err
}

type ListLicensesParams struct {
	ProductID   string
	Email       string
	Tier        string
	LicenseType string
	IsActive    *bool
	PerPage     int
}

type ListLicensesResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    []License `json:"data"`
	Meta    struct {
		Pagination struct {
			PerPage    int     `json:"per_page"`
			NextCursor *string `json:"next_cursor"`
			PrevCursor *string `json:"prev_cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

func (c *APIClient) ListLicenses(ctx context.Context, params ListLicensesParams) (ListLicensesResponse, error) {
	query := url.Values{}
	if params.ProductID != "" {
		query.Set("product_id", params.ProductID)
	}
	if params.Email != "" {
		query.Set("email", params.Email)
	}
	if params.Tier != "" {
		query.Set("tier", params.Tier)
	}
	if params.LicenseType != "" {
		query.Set("license_type", params.LicenseType)
	}
	if params.IsActive != nil {
		query.Set("is_active", strconv.FormatBool(*params.IsActive))
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}

	path := "/v1/licenses"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out ListLicensesResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

type ShowLicenseResponse struct {
	Success bool    `json:"success"`
	License License `json:"license"`
}

func (c *APIClient) ShowLicense(ctx context.Context, id string) (ShowLicenseResponse, error) {
	var out ShowLicenseResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/licenses/"+id, nil, &out)
	return out, err
}

type VerifyLicenseParams struct {
	Key       string `json:"key,omitempty"`
	LicenseID string `json:"license_id,omitempty"`
	ProductID string `json:"product_id,omitempty"`
}

type VerifyLicenseResponse struct {
	Success     bool     `json:"success"`
	Valid       bool     `json:"valid"`
	Reason      *string  `json:"reason"`
	Environment string   `json:"environment"`
	License     *License `json:"license"`
	ExpiresAt   *string  `json:"expires_at"`
	Tier        string   `json:"tier"`
	Product     *struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"product"`
}

// A 422 here just means invalid, not an error
func (c *APIClient) VerifyLicenseOnline(ctx context.Context, params VerifyLicenseParams) (VerifyLicenseResponse, error) {
	var out VerifyLicenseResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/verify/license", params, &out)

	var apiErr *APIError
	if err != nil {
		if isAPIError(err, &apiErr) && apiErr.StatusCode == 422 {
			return out, nil
		}
		return out, err
	}

	return out, nil
}

func isAPIError(err error, target **APIError) bool {
	if apiErr, ok := err.(*APIError); ok {
		*target = apiErr
		return true
	}
	return false
}

type RevokeLicenseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		RevokedAt           *string `json:"revoked_at"`
		SessionsDeactivated int     `json:"sessions_deactivated"`
	} `json:"data"`
	Errors *struct {
		Reason string `json:"reason"`
	} `json:"errors"`
}

func (c *APIClient) RevokeLicense(ctx context.Context, id, reason string) (RevokeLicenseResponse, error) {
	body := map[string]string{}
	if reason != "" {
		body["reason"] = reason
	}

	var out RevokeLicenseResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/licenses/"+id+"/revoke", body, &out)
	return out, err
}

type RenewLicenseParams struct {
	Duration               string `json:"duration"`
	Reactivate             bool   `json:"reactivate,omitempty"`
	Reference              string `json:"reference,omitempty"`
	Force                  bool   `json:"force,omitempty"`
	MinDaysBeforeExpiry    int    `json:"min_days_before_expiry,omitempty"`
	MinDaysBetweenRenewals int    `json:"min_days_between_renewals,omitempty"`
}

type RenewLicenseResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    License `json:"data"`
	Meta    struct {
		NewExpiration string `json:"new_expiration"`
		RenewalCount  int    `json:"renewal_count"`
	} `json:"meta"`
	Errors *struct {
		Reason    string `json:"reason"`
		LicenseID string `json:"license_id"`
	} `json:"errors"`
}

func (c *APIClient) RenewLicense(ctx context.Context, id string, params RenewLicenseParams) (RenewLicenseResponse, error) {
	var out RenewLicenseResponse
	err := c.doJSONIdempotent(ctx, http.MethodPost, "/v1/licenses/"+id+"/renew", params, &out)
	return out, err
}

type VendorKey struct {
	ID        string    `json:"id"`
	Kid       string    `json:"kid"`
	Algorithm string    `json:"algorithm"`
	IsActive  looseBool `json:"is_active"`
	CreatedAt string    `json:"created_at"`
}

// The vendor keys endpoint sends is_active as 0/1, not a real bool
type looseBool bool

func (b *looseBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", "1":
		*b = true
	case "false", "0":
		*b = false
	default:
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*b = looseBool(v)
	}
	return nil
}

func (c *APIClient) VendorKeys(ctx context.Context, vendorID string) ([]VendorKey, error) {
	var out []VendorKey
	err := c.doJSON(ctx, http.MethodGet, "/v1/vendors/"+vendorID+"/keys", nil, &out)
	return out, err
}

// The PEM body is a raw Ed25519 key, not SPKI/DER, see internal/licenseverify
func (c *APIClient) VendorPublicKeyPEM(ctx context.Context, vendorID, kid string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/vendors/"+vendorID+"/keys/"+kid+".pem", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("reaching %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", &APIError{StatusCode: resp.StatusCode, Message: string(raw)}
	}

	return string(raw), nil
}
