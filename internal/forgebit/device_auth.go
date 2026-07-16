package forgebit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Separate from APIClient since login has no token yet
type DeviceAuthClient struct {
	BaseURL string
	http    *http.Client
}

func NewDeviceAuthClient(baseURL string) *DeviceAuthClient {
	return &DeviceAuthClient{
		BaseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type DeviceAuthorization struct {
	DeviceCode              string  `json:"device_code"`
	UserCode                string  `json:"user_code"`
	VerificationURI         string  `json:"verification_uri"`
	VerificationURIComplete string  `json:"verification_uri_complete"`
	ExpiresIn               float64 `json:"expires_in"`
	Interval                int     `json:"interval"`
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *DeviceAuthClient) RequestAuthorization(ctx context.Context, clientName string) (DeviceAuthorization, error) {
	body, err := json.Marshal(map[string]string{"client_name": clientName})
	if err != nil {
		return DeviceAuthorization{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/device/cli/authorizations", bytes.NewReader(body))
	if err != nil {
		return DeviceAuthorization{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return DeviceAuthorization{}, fmt.Errorf("requesting device authorization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DeviceAuthorization{}, fmt.Errorf("unexpected status requesting device authorization: %s", resp.Status)
	}

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return DeviceAuthorization{}, err
	}

	var auth DeviceAuthorization
	if err := json.Unmarshal(envelope.Data, &auth); err != nil {
		return DeviceAuthorization{}, err
	}

	return auth, nil
}

type PollStatus string

const (
	PollStatusPending  PollStatus = "authorization_pending"
	PollStatusApproved PollStatus = "approved"
	PollStatusDenied   PollStatus = "denied"
	PollStatusExpired  PollStatus = "expired"
)

type PollResult struct {
	Status     PollStatus
	APIKeyID   string
	Token      string
	VendorID   string
	VendorName string
}

func (c *DeviceAuthClient) Poll(ctx context.Context, deviceCode string) (PollResult, error) {
	body, err := json.Marshal(map[string]string{"device_code": deviceCode})
	if err != nil {
		return PollResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/device/cli/authorizations/poll", bytes.NewReader(body))
	if err != nil {
		return PollResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return PollResult{}, fmt.Errorf("polling device authorization: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return PollResult{}, &RateLimitedError{RetryAfter: resp.Header.Get("Retry-After")}
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusGone, http.StatusNotFound:
	default:
		return PollResult{}, fmt.Errorf("unexpected status polling device authorization: %s", resp.Status)
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Status     PollStatus `json:"status"`
			APIKeyID   string     `json:"api_key_id"`
			Token      string     `json:"token"`
			VendorID   string     `json:"vendor_id"`
			VendorName string     `json:"vendor_name"`
		} `json:"data"`
		Errors struct {
			Status PollStatus `json:"status"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return PollResult{}, err
	}

	status := envelope.Data.Status
	if status == "" {
		status = envelope.Errors.Status
	}
	if status == "" && resp.StatusCode == http.StatusNotFound {
		return PollResult{}, fmt.Errorf("unknown or already-used device code")
	}

	return PollResult{
		Status:     status,
		APIKeyID:   envelope.Data.APIKeyID,
		Token:      envelope.Data.Token,
		VendorID:   envelope.Data.VendorID,
		VendorName: envelope.Data.VendorName,
	}, nil
}

type RateLimitedError struct {
	RetryAfter string
}

func (e *RateLimitedError) Error() string {
	return "rate limited by server"
}
