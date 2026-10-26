package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup",
	Short: "Find every license a customer holds for a product, by exact email",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		flags := c.Flags()
		productID, _ := flags.GetString("product-id")
		email, _ := flags.GetString("email")
		asJSON, _ := flags.GetBool("json")

		result, err := client.LookupCustomerLicenses(c.Context(), productID, email)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Printf("No licenses for %s on this product.\n", email)
			return nil
		}
		for _, license := range result.Data {
			seats := "1"
			if license.IsFloating {
				seats = "unlimited"
				if license.MaxConcurrentUsers != nil {
					seats = strconv.Itoa(*license.MaxConcurrentUsers)
				}
			}
			fmt.Printf("%s  status:%s  tier:%s  type:%s  seats:%s  expires:%s\n",
				license.ID, license.Status, license.Tier, license.LicenseType, seats, formatExpiry(license.ExpiresAt))
			if license.Key != nil {
				fmt.Printf("  key: %s\n", *license.Key)
			}
		}
		return nil
	},
}

func init() {
	lookupCmd.Flags().String("product-id", "", "product ID to search (required)")
	lookupCmd.Flags().String("email", "", "customer email, matched exactly and case-insensitively (required)")
	_ = lookupCmd.MarkFlagRequired("product-id")
	_ = lookupCmd.MarkFlagRequired("email")

	licensesCmd.AddCommand(lookupCmd)
}
