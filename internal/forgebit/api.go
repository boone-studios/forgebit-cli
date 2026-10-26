package forgebit

import (
	"context"
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

// Response is the standard {success, message, data} envelope most endpoints return
type Response[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// MessageResponse is for endpoints that return no data, just a confirmation
type MessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ListResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
	Meta    struct {
		Pagination struct {
			PerPage    int     `json:"per_page"`
			NextCursor *string `json:"next_cursor"`
			PrevCursor *string `json:"prev_cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

func (c *APIClient) RevokeCurrentToken(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "/v1/cli/logout", nil, nil)
}
