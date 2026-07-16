package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/boone-studios/forgebit-cli/internal/config"
	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var cfg config.Config

var vendorFlag string

var rootCmd = &cobra.Command{
	Use:   "forgebit",
	Short: "forgebit-cli is a standalone tool for working with Forgebit",
	Long: `forgebit-cli talks to the Forgebit API when a connection and
credentials are available, and falls back to local offline data otherwise.`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Order matters: explicit token, then --vendor, then active profile, then legacy token
func resolveAPIClient() (*forgebit.APIClient, error) {
	if cfg.Offline {
		return nil, errors.New("--offline is set; remove it to make API calls")
	}
	if cfg.APIBaseURL == "" {
		return nil, errors.New("no API URL configured; pass --api-url or set it in config")
	}

	if rootCmd.PersistentFlags().Changed("token") {
		return forgebit.NewAPIClient(cfg.APIBaseURL, cfg.Token), nil
	}

	if vendorFlag != "" {
		profile, ok := cfg.ProfileForVendor(vendorFlag)
		if !ok {
			return nil, fmt.Errorf("no stored credential for vendor %q; run `forgebit login` for it first", vendorFlag)
		}
		return forgebit.NewAPIClient(cfg.APIBaseURL, profile.Token), nil
	}

	if profile, ok := cfg.ActiveProfile(); ok {
		return forgebit.NewAPIClient(cfg.APIBaseURL, profile.Token), nil
	}

	if cfg.Token != "" {
		return forgebit.NewAPIClient(cfg.APIBaseURL, cfg.Token), nil
	}

	return nil, errors.New("not logged in; run `forgebit login` first")
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.APIBaseURL, "api-url", "", "Forgebit API base URL (overrides config file, default "+config.DefaultAPIBaseURL+")")
	rootCmd.PersistentFlags().StringVar(&cfg.Token, "token", "", "Forgebit API token (overrides config file and any stored vendor profile)")
	rootCmd.PersistentFlags().BoolVar(&cfg.Offline, "offline", false, "force offline mode, skipping any API calls")
	rootCmd.PersistentFlags().StringVar(&vendorFlag, "vendor", "", "act as a specific already-authenticated vendor (ID or name), without changing the active default")

	cobra.OnInitialize(func() {
		loaded, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "warning: failed to load config:", err)
			return
		}
		cfg.MergeDefaults(loaded)
	})
}
