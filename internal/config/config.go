package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const DefaultAPIBaseURL = "https://api.forgebit.io"

type VendorProfile struct {
	VendorID   string `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
	Token      string `json:"token"`
}

type Config struct {
	APIBaseURL string `json:"api_base_url"`
	Token      string `json:"token"` // Legacy single-account fallback, kept for back-compat

	ActiveVendorID string          `json:"active_vendor_id,omitempty"`
	Profiles       []VendorProfile `json:"profiles,omitempty"`

	Offline bool `json:"-"` // Flag-only, never persisted
}

func (c Config) ActiveProfile() (VendorProfile, bool) {
	if c.ActiveVendorID == "" {
		return VendorProfile{}, false
	}
	return c.ProfileForVendor(c.ActiveVendorID)
}

// ID match wins over name match
func (c Config) ProfileForVendor(idOrName string) (VendorProfile, bool) {
	for _, p := range c.Profiles {
		if p.VendorID == idOrName {
			return p, true
		}
	}
	for _, p := range c.Profiles {
		if strings.EqualFold(p.VendorName, idOrName) {
			return p, true
		}
	}
	return VendorProfile{}, false
}

func (c *Config) UpsertProfile(p VendorProfile) {
	for i, existing := range c.Profiles {
		if existing.VendorID == p.VendorID {
			c.Profiles[i] = p
			return
		}
	}
	c.Profiles = append(c.Profiles, p)
}

func (c *Config) RemoveProfile(vendorID string) {
	filtered := c.Profiles[:0]
	for _, p := range c.Profiles {
		if p.VendorID != vendorID {
			filtered = append(filtered, p)
		}
	}
	c.Profiles = filtered
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "forgebit", "config.json"), nil
}

func Load() (Config, error) {
	var c Config

	p, err := path()
	if err != nil {
		return c, err
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

func Save(c Config) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (c *Config) MergeDefaults(file Config) {
	if c.APIBaseURL == "" {
		c.APIBaseURL = file.APIBaseURL
	}
	if c.APIBaseURL == "" {
		c.APIBaseURL = DefaultAPIBaseURL
	}
	if c.Token == "" {
		c.Token = file.Token
	}
	// No flags for these, they always come from the file
	c.ActiveVendorID = file.ActiveVendorID
	c.Profiles = file.Profiles
}
