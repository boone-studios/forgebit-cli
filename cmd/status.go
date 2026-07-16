package cmd

import (
	"context"
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the CLI is running against the API or offline data",
	RunE: func(c *cobra.Command, args []string) error {
		backend, err := resolveBackend()
		if err != nil {
			return err
		}

		status, err := backend.Ping(context.Background())
		if err != nil {
			return err
		}

		fmt.Printf("mode:    %s\n", status.Source)
		fmt.Printf("details: %s\n", status.Details)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// Unlike requireAuth, no credential just means offline, not an error
func resolveBackend() (forgebit.Backend, error) {
	if client, err := resolveAPIClient(); err == nil {
		return client, nil
	}
	return forgebit.NewOfflineStore()
}
