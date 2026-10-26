package forgebit

import (
	"context"
	"net/http"
)

type Activation struct {
	ID               string  `json:"id"`
	ClientIdentifier string  `json:"client_identifier"`
	ActivatedAt      string  `json:"activated_at"`
	LastCheckIn      *string `json:"last_check_in"`
}

// MaxAllowed and AvailableSeats are nil when the license has unlimited seats
type ListActivationsResponse struct {
	Success bool         `json:"success"`
	Message string       `json:"message"`
	Data    []Activation `json:"data"`
	Meta    struct {
		ActiveCount    int  `json:"active_count"`
		MaxAllowed     *int `json:"max_allowed"`
		AvailableSeats *int `json:"available_seats"`
		IsFloating     bool `json:"is_floating"`
	} `json:"meta"`
}

func (c *APIClient) ListActivations(ctx context.Context, licenseID string) (ListActivationsResponse, error) {
	var out ListActivationsResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/licenses/"+licenseID+"/activations", nil, &out)
	return out, err
}

// clientID is optional; when set, the server rejects the call unless it matches the session
func (c *APIClient) DeactivateActivation(ctx context.Context, activationID, clientID string) (MessageResponse, error) {
	body := map[string]string{}
	if clientID != "" {
		body["client_id"] = clientID
	}

	var out MessageResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/activations/"+activationID+"/deactivate", body, &out)
	return out, err
}

type DeactivateAllResult struct {
	Count int `json:"count"`
}

func (c *APIClient) DeactivateAllActivations(ctx context.Context, licenseID string) (Response[DeactivateAllResult], error) {
	var out Response[DeactivateAllResult]
	err := c.doJSON(ctx, http.MethodPost, "/v1/licenses/"+licenseID+"/deactivate-all", nil, &out)
	return out, err
}
