package cmd

import (
	"context"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

// expiring and expired differ only in endpoint and default window
func newExpiryCmd(use, short string, defaultDays int, fetch func(*forgebit.APIClient, context.Context, int) (forgebit.Response[[]forgebit.License], error)) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			client, err := resolveAPIClient()
			if err != nil {
				return err
			}

			days, _ := c.Flags().GetInt("days")
			asJSON, _ := c.Flags().GetBool("json")

			result, err := fetch(client, c.Context(), days)
			if err != nil {
				return err
			}

			if asJSON {
				return printJSON(result)
			}

			printLicenseRows(result.Data)
			return nil
		},
	}
	cmd.Flags().Int("days", defaultDays, "window in days (1–90)")
	return cmd
}

func init() {
	licensesCmd.AddCommand(
		newExpiryCmd("expiring", "List licenses expiring within the next N days", 7, (*forgebit.APIClient).ExpiringLicenses),
		newExpiryCmd("expired", "List licenses that expired within the last N days", 30, (*forgebit.APIClient).ExpiredLicenses),
	)
}
