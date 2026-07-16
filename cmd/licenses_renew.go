package cmd

import (
	"errors"
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var renewCmd = &cobra.Command{
	Use:   "renew <license-id>",
	Short: "Renew a license",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		duration, _ := flags.GetString("duration")
		reactivate, _ := flags.GetBool("reactivate")
		reference, _ := flags.GetString("reference")
		force, _ := flags.GetBool("force")
		minDaysBeforeExpiry, _ := flags.GetInt("min-days-before-expiry")
		minDaysBetweenRenewals, _ := flags.GetInt("min-days-between-renewals")
		asJSON, _ := flags.GetBool("json")

		result, err := client.RenewLicense(c.Context(), args[0], forgebit.RenewLicenseParams{
			Duration:               duration,
			Reactivate:             reactivate,
			Reference:              reference,
			Force:                  force,
			MinDaysBeforeExpiry:    minDaysBeforeExpiry,
			MinDaysBetweenRenewals: minDaysBetweenRenewals,
		})
		if err != nil {
			var apiErr *forgebit.APIError
			if errors.As(err, &apiErr) && result.Errors != nil {
				if asJSON {
					return printJSON(result)
				}
				fmt.Printf("✗ not renewed — reason: %s\n", result.Errors.Reason)
				return nil
			}
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("→ renewed — new expiration: %s\n", result.Meta.NewExpiration)
		return nil
	},
}

func init() {
	renewCmd.Flags().String("duration", "", "renewal duration, e.g. '30_days', '1_year' (required)")
	renewCmd.Flags().Bool("reactivate", false, "reactivate the license if it was expired")
	renewCmd.Flags().String("reference", "", "external reference for this renewal")
	renewCmd.Flags().Bool("force", false, "bypass the min-days guards")
	renewCmd.Flags().Int("min-days-before-expiry", 30, "reject renewal if more than this many days remain")
	renewCmd.Flags().Int("min-days-between-renewals", 7, "reject renewal if renewed this recently")
	renewCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	_ = renewCmd.MarkFlagRequired("duration")

	licensesCmd.AddCommand(renewCmd)
}
