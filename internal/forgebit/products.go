package forgebit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type Product struct {
	ID                       string   `json:"id"`
	VendorID                 string   `json:"vendor_id"`
	Name                     string   `json:"name"`
	Slug                     string   `json:"slug"`
	Description              *string  `json:"description"`
	DefaultExpiry            *string  `json:"default_expiry"`
	AllowedLicenseTypes      []string `json:"allowed_license_types"`
	CustomerPortalManaged    bool     `json:"customer_portal_managed"`
	RequireDeviceFingerprint bool     `json:"require_device_fingerprint"`
	IsStaleDetectionEnabled  bool     `json:"is_stale_detection_enabled"`
	StaleThresholdDays       *int     `json:"stale_threshold_days"`
	ArchivedAt               *string  `json:"archived_at"`
	CreatedAt                string   `json:"created_at"`
}

type CreateProductParams struct {
	Name                     string   `json:"name"`
	Slug                     string   `json:"slug,omitempty"`
	Description              string   `json:"description,omitempty"`
	DefaultExpiry            string   `json:"default_expiry,omitempty"`
	AllowedLicenseTypes      []string `json:"allowed_license_types,omitempty"`
	CustomerPortalManaged    bool     `json:"customer_portal_managed,omitempty"`
	RequireDeviceFingerprint bool     `json:"require_device_fingerprint,omitempty"`
	IsStaleDetectionEnabled  bool     `json:"is_stale_detection_enabled,omitempty"`
	StaleThresholdDays       int      `json:"stale_threshold_days,omitempty"`
}

type CreateProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    Product `json:"data"`
}

func (c *APIClient) CreateProduct(ctx context.Context, params CreateProductParams) (CreateProductResponse, error) {
	var out CreateProductResponse
	err := c.doJSONIdempotent(ctx, http.MethodPost, "/v1/products", params, &out)
	return out, err
}

type ListProductsParams struct {
	Search  string
	PerPage int
}

type ListProductsResponse struct {
	Success bool      `json:"success"`
	Message string    `json:"message"`
	Data    []Product `json:"data"`
	Meta    struct {
		Pagination struct {
			PerPage    int     `json:"per_page"`
			NextCursor *string `json:"next_cursor"`
			PrevCursor *string `json:"prev_cursor"`
		} `json:"pagination"`
	} `json:"meta"`
}

func (c *APIClient) ListProducts(ctx context.Context, params ListProductsParams) (ListProductsResponse, error) {
	query := url.Values{}
	if params.Search != "" {
		query.Set("search", params.Search)
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}

	path := "/v1/products"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out ListProductsResponse
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

type ShowProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    Product `json:"data"`
}

func (c *APIClient) ShowProduct(ctx context.Context, id string) (ShowProductResponse, error) {
	var out ShowProductResponse
	err := c.doJSON(ctx, http.MethodGet, "/v1/products/"+id, nil, &out)
	return out, err
}

type UpdateProductParams struct {
	Name                     string   `json:"name"`
	Description              string   `json:"description,omitempty"`
	DefaultExpiry            string   `json:"default_expiry,omitempty"`
	AllowedLicenseTypes      []string `json:"allowed_license_types,omitempty"`
	CustomerPortalManaged    bool     `json:"customer_portal_managed,omitempty"`
	RequireDeviceFingerprint bool     `json:"require_device_fingerprint,omitempty"`
	IsStaleDetectionEnabled  bool     `json:"is_stale_detection_enabled,omitempty"`
	StaleThresholdDays       int      `json:"stale_threshold_days,omitempty"`
}

type UpdateProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    Product `json:"data"`
}

func (c *APIClient) UpdateProduct(ctx context.Context, id string, params UpdateProductParams) (UpdateProductResponse, error) {
	var out UpdateProductResponse
	err := c.doJSON(ctx, http.MethodPatch, "/v1/products/"+id, params, &out)
	return out, err
}

type ArchiveProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    Product `json:"data"`
}

func (c *APIClient) ArchiveProduct(ctx context.Context, id string) (ArchiveProductResponse, error) {
	var out ArchiveProductResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/products/"+id+"/archive", nil, &out)
	return out, err
}

type RestoreProductResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Data    Product `json:"data"`
}

func (c *APIClient) RestoreProduct(ctx context.Context, id string) (RestoreProductResponse, error) {
	var out RestoreProductResponse
	err := c.doJSON(ctx, http.MethodPost, "/v1/products/"+id+"/restore", nil, &out)
	return out, err
}
