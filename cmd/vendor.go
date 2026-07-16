package cmd

import (
	"fmt"

	"github.com/boone-studios/forgebit-cli/internal/config"
	"github.com/spf13/cobra"
)

var vendorCmd = &cobra.Command{
	Use:   "vendor",
	Short: "Manage which vendor the CLI acts as",
}

func init() {
	rootCmd.AddCommand(vendorCmd)
}

var vendorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List vendors the CLI has stored credentials for",
	RunE: func(c *cobra.Command, args []string) error {
		if len(cfg.Profiles) == 0 {
			if cfg.Token != "" {
				fmt.Println("Logged in from before multi-vendor support was added — run `forgebit login` again to name and track this vendor, or add another.")
				return nil
			}
			fmt.Println("No stored vendors. Run `forgebit login` to add one.")
			return nil
		}

		for _, p := range cfg.Profiles {
			marker := " "
			if p.VendorID == cfg.ActiveVendorID {
				marker = "*"
			}
			fmt.Printf("%s %s  %s\n", marker, p.VendorID, p.VendorName)
		}
		return nil
	},
}

var vendorSwitchCmd = &cobra.Command{
	Use:   "switch <id|name>",
	Short: "Set the active vendor for future commands",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		profile, ok := cfg.ProfileForVendor(args[0])
		if !ok {
			return fmt.Errorf("no stored credential for vendor %q; run `forgebit login` for it first", args[0])
		}

		newCfg := cfg
		newCfg.ActiveVendorID = profile.VendorID
		if err := config.Save(newCfg); err != nil {
			return err
		}

		fmt.Printf("Switched to %s (vendor %s).\n", profile.VendorName, profile.VendorID)
		return nil
	},
}

func init() {
	vendorCmd.AddCommand(vendorListCmd, vendorSwitchCmd)
}
