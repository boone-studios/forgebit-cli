package forgebit

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type APIClient struct {
	BaseURL string
	Token   string
	http    *http.Client
}

func NewAPIClient(baseURL, token string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) Ping(ctx context.Context) (Status, error) {
	// Placeholder endpoint, not real yet
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/health", nil)
	if err != nil {
		return Status{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return Status{}, fmt.Errorf("reaching %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("unexpected status from %s: %s", c.BaseURL, resp.Status)
	}

	return Status{Source: "api", Details: c.BaseURL}, nil
}

func (c *APIClient) RevokeCurrentToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/cli/logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("reaching %s: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status from %s: %s", c.BaseURL, resp.Status)
	}

	return nil
}
