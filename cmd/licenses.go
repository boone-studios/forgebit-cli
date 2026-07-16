package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

func printJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

var licensesCmd = &cobra.Command{
	Use:   "licenses",
	Short: "Issue, inspect, and manage Forgebit licenses",
}

func init() {
	rootCmd.AddCommand(licensesCmd)
}

func requireAuth() (*forgebit.APIClient, error) {
	return resolveAPIClient()
}

func toAnyMap(m map[string]string) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func formatExpiry(expiresAt *string) string {
	if expiresAt == nil || *expiresAt == "" {
		return "never"
	}
	return *expiresAt
}

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue a new license",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		productID, _ := flags.GetString("product-id")
		customerEmail, _ := flags.GetString("customer-email")
		customerName, _ := flags.GetString("customer-name")
		tier, _ := flags.GetString("tier")
		licenseType, _ := flags.GetString("license-type")
		licenseTypeID, _ := flags.GetString("license-type-id")
		durationType, _ := flags.GetString("duration-type")
		expiresAt, _ := flags.GetString("expires-at")
		isOffline, _ := flags.GetBool("offline")
		isFloating, _ := flags.GetBool("floating")
		seats, _ := flags.GetInt("seats")
		metadata, _ := flags.GetStringToString("metadata")
		features, _ := flags.GetStringToString("features")
		orderReference, _ := flags.GetString("order-reference")
		asJSON, _ := flags.GetBool("json")

		if licenseType == "" && licenseTypeID == "" {
			return errors.New("one of --license-type or --license-type-id is required")
		}
		if isFloating && seats <= 0 {
			return errors.New("--seats is required when --floating is set")
		}

		params := forgebit.IssueLicenseParams{
			CustomerEmail:      customerEmail,
			CustomerName:       customerName,
			ProductID:          productID,
			Tier:               tier,
			LicenseType:        licenseType,
			LicenseTypeID:      licenseTypeID,
			LicenseDuration:    durationType,
			ExpiresAt:          expiresAt,
			IsOffline:          isOffline,
			IsFloating:         isFloating,
			MaxConcurrentUsers: seats,
			Metadata:           toAnyMap(metadata),
			Features:           toAnyMap(features),
			OrderReference:     orderReference,
		}

		result, err := client.IssueLicense(c.Context(), params)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("→ %s\n", result.Key)
		fmt.Printf("  license: %s  type: %s  tier: %s  vendor: %s  expires: %s\n",
			result.License.ID, result.License.LicenseType, result.License.Tier, result.License.VendorID, formatExpiry(result.License.ExpiresAt))
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List licenses",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		productID, _ := flags.GetString("product-id")
		email, _ := flags.GetString("email")
		tier, _ := flags.GetString("tier")
		licenseType, _ := flags.GetString("license-type")
		active, _ := flags.GetBool("active")
		inactive, _ := flags.GetBool("inactive")
		perPage, _ := flags.GetInt("per-page")
		asJSON, _ := flags.GetBool("json")

		if active && inactive {
			return errors.New("--active and --inactive are mutually exclusive")
		}

		var isActive *bool
		if active {
			t := true
			isActive = &t
		} else if inactive {
			f := false
			isActive = &f
		}

		result, err := client.ListLicenses(c.Context(), forgebit.ListLicensesParams{
			ProductID:   productID,
			Email:       email,
			Tier:        tier,
			LicenseType: licenseType,
			IsActive:    isActive,
			PerPage:     perPage,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Println("No licenses found.")
			return nil
		}
		for _, license := range result.Data {
			fmt.Printf("%s  tier:%s  type:%s  env:%s  expires:%s\n",
				license.ID, license.Tier, license.LicenseType, license.Environment, formatExpiry(license.ExpiresAt))
		}
		if result.Meta.Pagination.NextCursor != nil {
			fmt.Printf("(more results — next cursor: %s)\n", *result.Meta.Pagination.NextCursor)
		}
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show <license-id>",
	Short: "Show a single license",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ShowLicense(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		license := result.License
		fmt.Printf("%s  tier:%s  type:%s  env:%s  vendor:%s  expires:%s\n",
			license.ID, license.Tier, license.LicenseType, license.Environment, license.VendorID, formatExpiry(license.ExpiresAt))
		return nil
	},
}

func init() {
	issueCmd.Flags().String("product-id", "", "product ID the license belongs to (required)")
	issueCmd.Flags().String("customer-email", "", "customer email (required)")
	issueCmd.Flags().String("customer-name", "", "customer name")
	issueCmd.Flags().String("tier", "", "license tier (required)")
	issueCmd.Flags().String("license-type", "", "license type slug (jwt, forgebit, serial, hmac, hwid, encfile)")
	issueCmd.Flags().String("license-type-id", "", "license type ID, alternative to --license-type")
	issueCmd.Flags().String("duration-type", "", "trial, subscription, or perpetual (required)")
	issueCmd.Flags().String("expires-at", "", "explicit expiry (RFC3339)")
	issueCmd.Flags().Bool("offline", false, "mark the license as offline-capable")
	issueCmd.Flags().Bool("floating", false, "issue a floating (seat-based) license")
	issueCmd.Flags().Int("seats", 0, "max concurrent users; required with --floating")
	issueCmd.Flags().StringToString("metadata", nil, "metadata key=value pairs, repeatable")
	issueCmd.Flags().StringToString("features", nil, "feature flag key=value pairs, repeatable")
	issueCmd.Flags().String("order-reference", "", "external order/purchase reference")
	issueCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	_ = issueCmd.MarkFlagRequired("product-id")
	_ = issueCmd.MarkFlagRequired("customer-email")
	_ = issueCmd.MarkFlagRequired("tier")
	_ = issueCmd.MarkFlagRequired("duration-type")

	listCmd.Flags().String("product-id", "", "filter by product ID")
	listCmd.Flags().String("email", "", "filter by customer email (substring match)")
	listCmd.Flags().String("tier", "", "filter by tier")
	listCmd.Flags().String("license-type", "", "filter by license type slug")
	listCmd.Flags().Bool("active", false, "only active licenses")
	listCmd.Flags().Bool("inactive", false, "only inactive/expired/revoked licenses")
	listCmd.Flags().Int("per-page", 15, "results per page (max 100)")
	listCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	showCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	licensesCmd.AddCommand(issueCmd, listCmd, showCmd)
}
