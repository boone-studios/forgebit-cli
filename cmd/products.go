package cmd

import (
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var productsCmd = &cobra.Command{
	Use:   "products",
	Short: "Create, inspect, and manage Forgebit products",
}

func init() {
	rootCmd.AddCommand(productsCmd)
}

func formatDescription(description *string) string {
	if description == nil || *description == "" {
		return "-"
	}
	return *description
}

var productsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List products",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		search, _ := flags.GetString("search")
		perPage, _ := flags.GetInt("per-page")
		asJSON, _ := flags.GetBool("json")

		result, err := client.ListProducts(c.Context(), forgebit.ListProductsParams{
			Search:  search,
			PerPage: perPage,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Println("No products found.")
			return nil
		}
		for _, product := range result.Data {
			fmt.Printf("%s  %s  slug:%s\n", product.ID, product.Name, product.Slug)
		}
		if result.Meta.Pagination.NextCursor != nil {
			fmt.Printf("(more results — next cursor: %s)\n", *result.Meta.Pagination.NextCursor)
		}
		return nil
	},
}

var productsShowCmd = &cobra.Command{
	Use:   "show <product-id>",
	Short: "Show a single product",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ShowProduct(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		product := result.Data
		fmt.Printf("%s  %s  slug:%s  description:%s\n", product.ID, product.Name, product.Slug, formatDescription(product.Description))
		return nil
	},
}

var productsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new product",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		name, _ := flags.GetString("name")
		slug, _ := flags.GetString("slug")
		description, _ := flags.GetString("description")
		defaultExpiry, _ := flags.GetString("default-expiry")
		allowedLicenseTypes, _ := flags.GetStringArray("allowed-license-type")
		customerPortalManaged, _ := flags.GetBool("customer-portal-managed")
		requireDeviceFingerprint, _ := flags.GetBool("require-device-fingerprint")
		staleDetection, _ := flags.GetBool("stale-detection")
		staleThresholdDays, _ := flags.GetInt("stale-threshold-days")
		asJSON, _ := flags.GetBool("json")

		if staleDetection && staleThresholdDays <= 0 {
			return fmt.Errorf("--stale-threshold-days is required when --stale-detection is set")
		}

		result, err := client.CreateProduct(c.Context(), forgebit.CreateProductParams{
			Name:                     name,
			Slug:                     slug,
			Description:              description,
			DefaultExpiry:            defaultExpiry,
			AllowedLicenseTypes:      allowedLicenseTypes,
			CustomerPortalManaged:    customerPortalManaged,
			RequireDeviceFingerprint: requireDeviceFingerprint,
			IsStaleDetectionEnabled:  staleDetection,
			StaleThresholdDays:       staleThresholdDays,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Created %s  %s  slug:%s\n", result.Data.ID, result.Data.Name, result.Data.Slug)
		return nil
	},
}

var productsUpdateCmd = &cobra.Command{
	Use:   "update <product-id>",
	Short: "Update a product",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		name, _ := flags.GetString("name")
		description, _ := flags.GetString("description")
		defaultExpiry, _ := flags.GetString("default-expiry")
		allowedLicenseTypes, _ := flags.GetStringArray("allowed-license-type")
		customerPortalManaged, _ := flags.GetBool("customer-portal-managed")
		requireDeviceFingerprint, _ := flags.GetBool("require-device-fingerprint")
		staleDetection, _ := flags.GetBool("stale-detection")
		staleThresholdDays, _ := flags.GetInt("stale-threshold-days")
		asJSON, _ := flags.GetBool("json")

		if staleDetection && staleThresholdDays <= 0 {
			return fmt.Errorf("--stale-threshold-days is required when --stale-detection is set")
		}

		result, err := client.UpdateProduct(c.Context(), args[0], forgebit.UpdateProductParams{
			Name:                     name,
			Description:              description,
			DefaultExpiry:            defaultExpiry,
			AllowedLicenseTypes:      allowedLicenseTypes,
			CustomerPortalManaged:    customerPortalManaged,
			RequireDeviceFingerprint: requireDeviceFingerprint,
			IsStaleDetectionEnabled:  staleDetection,
			StaleThresholdDays:       staleThresholdDays,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Updated %s  %s  slug:%s\n", result.Data.ID, result.Data.Name, result.Data.Slug)
		return nil
	},
}

var productsArchiveCmd = &cobra.Command{
	Use:   "archive <product-id>",
	Short: "Archive a product",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ArchiveProduct(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Archived %s  %s\n", result.Data.ID, result.Data.Name)
		return nil
	},
}

var productsRestoreCmd = &cobra.Command{
	Use:   "restore <product-id>",
	Short: "Restore an archived product",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.RestoreProduct(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Restored %s  %s\n", result.Data.ID, result.Data.Name)
		return nil
	},
}

func init() {
	productsListCmd.Flags().String("search", "", "filter by product name (substring match)")
	productsListCmd.Flags().Int("per-page", 15, "results per page (max 100)")
	productsListCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	productsShowCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	productsCreateCmd.Flags().String("name", "", "product name (required)")
	productsCreateCmd.Flags().String("slug", "", "product slug, auto-generated from name if omitted")
	productsCreateCmd.Flags().String("description", "", "product description")
	productsCreateCmd.Flags().String("default-expiry", "", "30, 90, 365, 730, or perpetual")
	productsCreateCmd.Flags().StringArray("allowed-license-type", nil, "allowed license type slug, repeatable")
	productsCreateCmd.Flags().Bool("customer-portal-managed", false, "let customers manage licenses for this product in the portal")
	productsCreateCmd.Flags().Bool("require-device-fingerprint", false, "require a device fingerprint on activation")
	productsCreateCmd.Flags().Bool("stale-detection", false, "enable stale license detection")
	productsCreateCmd.Flags().Int("stale-threshold-days", 0, "days of inactivity before a license is stale; required with --stale-detection")
	productsCreateCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	_ = productsCreateCmd.MarkFlagRequired("name")

	productsUpdateCmd.Flags().String("name", "", "product name (required)")
	productsUpdateCmd.Flags().String("description", "", "product description")
	productsUpdateCmd.Flags().String("default-expiry", "", "30, 90, 365, 730, or perpetual")
	productsUpdateCmd.Flags().StringArray("allowed-license-type", nil, "allowed license type slug, repeatable")
	productsUpdateCmd.Flags().Bool("customer-portal-managed", false, "let customers manage licenses for this product in the portal")
	productsUpdateCmd.Flags().Bool("require-device-fingerprint", false, "require a device fingerprint on activation")
	productsUpdateCmd.Flags().Bool("stale-detection", false, "enable stale license detection")
	productsUpdateCmd.Flags().Int("stale-threshold-days", 0, "days of inactivity before a license is stale; required with --stale-detection")
	productsUpdateCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	_ = productsUpdateCmd.MarkFlagRequired("name")

	productsArchiveCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	productsRestoreCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	productsCmd.AddCommand(productsListCmd, productsShowCmd, productsCreateCmd, productsUpdateCmd, productsArchiveCmd, productsRestoreCmd)
}
