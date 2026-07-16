package cmd

import (
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Create, inspect, and manage Forgebit webhook endpoints",
}

func init() {
	rootCmd.AddCommand(webhooksCmd)
}

func formatWebhookName(name *string) string {
	if name == nil || *name == "" {
		return "-"
	}
	return *name
}

func formatHTTPCode(code *int) string {
	if code == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *code)
}

func printRevealedSecret(secret *string) {
	if secret == nil {
		return
	}
	fmt.Printf("secret: %s\n", *secret)
	fmt.Println("This secret won't be shown again — store it now.")
}

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook endpoints",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		perPage, _ := c.Flags().GetInt("per-page")
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ListWebhooks(c.Context(), forgebit.ListWebhooksParams{PerPage: perPage})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Println("No webhook endpoints found.")
			return nil
		}
		for _, endpoint := range result.Data {
			fmt.Printf("%s  %s  %s  enabled:%v\n", endpoint.ID, formatWebhookName(endpoint.Name), endpoint.URL, endpoint.IsEnabled)
		}
		if result.Meta.Pagination.NextCursor != nil {
			fmt.Printf("(more results — next cursor: %s)\n", *result.Meta.Pagination.NextCursor)
		}
		return nil
	},
}

var webhooksShowCmd = &cobra.Command{
	Use:   "show <webhook-id>",
	Short: "Show a single webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ShowWebhook(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		endpoint := result.Data
		fmt.Printf("%s  %s  %s  enabled:%v  events:%v  has_secret:%v\n",
			endpoint.ID, formatWebhookName(endpoint.Name), endpoint.URL, endpoint.IsEnabled, endpoint.Events, endpoint.HasSecret)
		return nil
	},
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new webhook endpoint",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		name, _ := flags.GetString("name")
		url, _ := flags.GetString("url")
		events, _ := flags.GetStringArray("event")
		disabled, _ := flags.GetBool("disabled")
		asJSON, _ := flags.GetBool("json")

		if len(events) == 0 {
			return fmt.Errorf("at least one --event is required")
		}

		result, err := client.CreateWebhook(c.Context(), forgebit.CreateWebhookParams{
			Name:      name,
			URL:       url,
			Events:    events,
			IsEnabled: !disabled,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Created %s  %s\n", result.Data.ID, result.Data.URL)
		printRevealedSecret(result.Data.Secret)
		return nil
	},
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update <webhook-id>",
	Short: "Update a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		name, _ := flags.GetString("name")
		url, _ := flags.GetString("url")
		events, _ := flags.GetStringArray("event")
		enabled, _ := flags.GetBool("enabled")
		disabled, _ := flags.GetBool("disabled")
		asJSON, _ := flags.GetBool("json")

		if enabled && disabled {
			return fmt.Errorf("--enabled and --disabled are mutually exclusive")
		}

		params := forgebit.UpdateWebhookParams{URL: url, Events: events}
		if flags.Changed("name") {
			params.Name = &name
		}
		if enabled {
			t := true
			params.IsEnabled = &t
		} else if disabled {
			f := false
			params.IsEnabled = &f
		}

		result, err := client.UpdateWebhook(c.Context(), args[0], params)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Updated %s  %s\n", result.Data.ID, result.Data.URL)
		return nil
	},
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-id>",
	Short: "Delete a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.DeleteWebhook(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println(result.Message)
		return nil
	},
}

var webhooksRotateSecretCmd = &cobra.Command{
	Use:   "rotate-secret <webhook-id>",
	Short: "Rotate a webhook endpoint's signing secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.RotateWebhookSecret(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("Rotated secret for %s\n", result.Data.ID)
		printRevealedSecret(result.Data.Secret)
		return nil
	},
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test <webhook-id>",
	Short: "Send a test event to a webhook endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		event, _ := c.Flags().GetString("event")
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.TestWebhook(c.Context(), args[0], event)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("%s  http_code:%s  %s\n", result.Data.Indicator, formatHTTPCode(result.Data.HTTPCode), result.Data.Message)
		return nil
	},
}

var webhooksLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Inspect and replay webhook delivery logs",
}

var webhooksLogsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhook delivery logs",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		endpointID, _ := flags.GetString("endpoint")
		status, _ := flags.GetString("status")
		perPage, _ := flags.GetInt("per-page")
		asJSON, _ := flags.GetBool("json")

		result, err := client.ListWebhookLogs(c.Context(), forgebit.ListWebhookLogsParams{
			EndpointID: endpointID,
			Status:     status,
			PerPage:    perPage,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if len(result.Data) == 0 {
			fmt.Println("No webhook logs found.")
			return nil
		}
		for _, log := range result.Data {
			fmt.Printf("%s  event:%s  status:%s  http_code:%s  retries:%d\n", log.ID, log.Event, log.Status, formatHTTPCode(log.HTTPCode), log.RetryCount)
		}
		if result.Meta.Pagination.NextCursor != nil {
			fmt.Printf("(more results — next cursor: %s)\n", *result.Meta.Pagination.NextCursor)
		}
		return nil
	},
}

var webhooksLogsReplayCmd = &cobra.Command{
	Use:   "replay <log-id>",
	Short: "Replay a webhook delivery",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ReplayWebhookLog(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println(result.Message)
		return nil
	},
}

func init() {
	webhooksListCmd.Flags().Int("per-page", 15, "results per page (max 100)")
	webhooksListCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksShowCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksCreateCmd.Flags().String("name", "", "webhook endpoint name")
	webhooksCreateCmd.Flags().String("url", "", "webhook endpoint URL (required)")
	webhooksCreateCmd.Flags().StringArray("event", nil, "event to subscribe to, repeatable (required)")
	webhooksCreateCmd.Flags().Bool("disabled", false, "create the endpoint disabled")
	webhooksCreateCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	_ = webhooksCreateCmd.MarkFlagRequired("url")

	webhooksUpdateCmd.Flags().String("name", "", "webhook endpoint name")
	webhooksUpdateCmd.Flags().String("url", "", "webhook endpoint URL")
	webhooksUpdateCmd.Flags().StringArray("event", nil, "event to subscribe to, repeatable")
	webhooksUpdateCmd.Flags().Bool("enabled", false, "enable the endpoint")
	webhooksUpdateCmd.Flags().Bool("disabled", false, "disable the endpoint")
	webhooksUpdateCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksDeleteCmd.Flags().Bool("json", false, "print the raw API response as JSON")
	webhooksRotateSecretCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksTestCmd.Flags().String("event", "webhook.test", "event to simulate")
	webhooksTestCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksLogsListCmd.Flags().String("endpoint", "", "filter by webhook endpoint ID")
	webhooksLogsListCmd.Flags().String("status", "", "filter by status (pending, success, failed, permanent_failure)")
	webhooksLogsListCmd.Flags().Int("per-page", 15, "results per page (max 100)")
	webhooksLogsListCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksLogsReplayCmd.Flags().Bool("json", false, "print the raw API response as JSON")

	webhooksLogsCmd.AddCommand(webhooksLogsListCmd, webhooksLogsReplayCmd)

	webhooksCmd.AddCommand(webhooksListCmd, webhooksShowCmd, webhooksCreateCmd, webhooksUpdateCmd, webhooksDeleteCmd, webhooksRotateSecretCmd, webhooksTestCmd, webhooksLogsCmd)
}
