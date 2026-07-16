package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/boone-studios/forgebit-cli/internal/config"
	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the CLI's stored credentials",
	RunE: func(c *cobra.Command, args []string) error {
		all, _ := c.Flags().GetBool("all")

		if all {
			return logoutAll(c.Context())
		}
		return logoutOne(c.Context())
	},
}

func init() {
	logoutCmd.Flags().Bool("all", false, "log out of every stored vendor, not just the target")
	rootCmd.AddCommand(logoutCmd)
}

func revokeBestEffort(ctx context.Context, token string) {
	if cfg.APIBaseURL == "" || token == "" {
		return
	}
	client := forgebit.NewAPIClient(cfg.APIBaseURL, token)
	if err := client.RevokeCurrentToken(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "warning: failed to revoke token on the server:", err)
	}
}

func logoutAll(ctx context.Context) error {
	for _, p := range cfg.Profiles {
		revokeBestEffort(ctx, p.Token)
	}
	if len(cfg.Profiles) == 0 && cfg.Token != "" {
		revokeBestEffort(ctx, cfg.Token)
	}

	cleared := cfg
	cleared.Token = ""
	cleared.ActiveVendorID = ""
	cleared.Profiles = nil
	if err := config.Save(cleared); err != nil {
		return err
	}

	fmt.Println("Logged out of all vendors.")
	return nil
}

func logoutOne(ctx context.Context) error {
	newCfg := cfg

	var target config.VendorProfile
	var haveTarget bool

	if vendorFlag != "" {
		target, haveTarget = cfg.ProfileForVendor(vendorFlag)
		if !haveTarget {
			fmt.Printf("No stored credential for vendor %q; nothing to log out of.\n", vendorFlag)
			return nil
		}
	} else if profile, ok := cfg.ActiveProfile(); ok {
		target, haveTarget = profile, true
	}

	if !haveTarget {
		if cfg.Token == "" {
			fmt.Println("Already logged out.")
			return nil
		}
		revokeBestEffort(ctx, cfg.Token)
		newCfg.Token = ""
		return config.Save(newCfg)
	}

	revokeBestEffort(ctx, target.Token)
	newCfg.RemoveProfile(target.VendorID)

	if newCfg.ActiveVendorID == target.VendorID {
		newCfg.ActiveVendorID = ""
		if len(newCfg.Profiles) > 0 {
			newCfg.ActiveVendorID = newCfg.Profiles[0].VendorID
			fmt.Printf("Logged out of %s. %s is now your active vendor.\n", target.VendorName, newCfg.Profiles[0].VendorName)
			return config.Save(newCfg)
		}
	}

	if newCfg.Token == target.Token {
		newCfg.Token = ""
	}

	fmt.Printf("Logged out of %s.\n", target.VendorName)
	return config.Save(newCfg)
}
