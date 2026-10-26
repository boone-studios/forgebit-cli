package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the CLI is set up to talk to the API",
	RunE: func(c *cobra.Command, args []string) error {
		// No credential just means offline, not an error
		if _, err := resolveAPIClient(); err != nil {
			fmt.Println("mode:    offline")
			fmt.Printf("details: %s\n", err)
			return nil
		}

		fmt.Println("mode:    api")
		fmt.Printf("details: %s\n", cfg.APIBaseURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
