package cmd

import (
	"errors"
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke <license-id>",
	Short: "Revoke a license",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		reason, _ := c.Flags().GetString("reason")
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.RevokeLicense(c.Context(), args[0], reason)
		if err != nil {
			var apiErr *forgebit.APIError
			if errors.As(err, &apiErr) && result.Errors != nil {
				if asJSON {
					return printJSON(result)
				}
				fmt.Printf("✗ not revoked — reason: %s\n", result.Errors.Reason)
				return nil
			}
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println("→ revoked")
		return nil
	},
}

func init() {
	revokeCmd.Flags().String("reason", "", "reason for revocation")
	revokeCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	licensesCmd.AddCommand(revokeCmd)
}
