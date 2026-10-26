package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var activationsCmd = &cobra.Command{
	Use:   "activations <license-id>",
	Short: "List a license's active sessions and seat usage",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ListActivations(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Println("No active sessions.")
		}
		for _, a := range result.Data {
			lastCheckIn := "-"
			if a.LastCheckIn != nil {
				lastCheckIn = *a.LastCheckIn
			}
			fmt.Printf("%s  client:%s  activated:%s  last_check_in:%s\n", a.ID, a.ClientIdentifier, a.ActivatedAt, lastCheckIn)
		}

		meta := result.Meta
		if meta.MaxAllowed == nil {
			fmt.Printf("seats: %d in use (unlimited)\n", meta.ActiveCount)
		} else {
			fmt.Printf("seats: %d of %d in use\n", meta.ActiveCount, *meta.MaxAllowed)
		}
		return nil
	},
}

var deactivateCmd = &cobra.Command{
	Use:   "deactivate <activation-id>",
	Short: "End a single activation session",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		clientID, _ := c.Flags().GetString("client-id")
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.DeactivateActivation(c.Context(), args[0], clientID)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println("→ deactivated")
		return nil
	},
}

var deactivateAllCmd = &cobra.Command{
	Use:   "deactivate-all <license-id>",
	Short: "End every active session for a license",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.DeactivateAllActivations(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("→ deactivated %d session(s)\n", result.Data.Count)
		return nil
	},
}

func init() {
	deactivateCmd.Flags().String("client-id", "", "only deactivate if the session's client identifier matches")

	licensesCmd.AddCommand(activationsCmd, deactivateCmd, deactivateAllCmd)
}
