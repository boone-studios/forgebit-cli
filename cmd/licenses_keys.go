package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Anything but y/yes (including no input, e.g. a non-interactive run) is a no
func confirm(c *cobra.Command, question string) bool {
	fmt.Fprintf(c.ErrOrStderr(), "%s [y/N] ", question)
	answer, _ := bufio.NewReader(c.InOrStdin()).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}
	return false
}

const vendorIDFlagUsage = "vendor ID (default: the vendor the CLI is acting as)"

// Defaults to the vendor the CLI is acting as, via --vendor or the active profile
func resolveVendorID(c *cobra.Command) (string, error) {
	if id, _ := c.Flags().GetString("vendor-id"); id != "" {
		return id, nil
	}
	// An explicit --token could belong to any vendor, so don't guess
	if cfg.Token == "" {
		if vendorFlag != "" {
			if profile, ok := cfg.ProfileForVendor(vendorFlag); ok {
				return profile.VendorID, nil
			}
		} else if profile, ok := cfg.ActiveProfile(); ok {
			return profile.VendorID, nil
		}
	}
	return "", errors.New("--vendor-id is required (find it in the output of `licenses issue`, `show`, `list`, or `verify`)")
}

var publicKeyCmd = &cobra.Command{
	Use:   "public-key",
	Short: "Fetch a vendor's Ed25519 public key for offline verification",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		vendorID, err := resolveVendorID(c)
		if err != nil {
			return err
		}

		flags := c.Flags()
		kid, _ := flags.GetString("kid")
		out, _ := flags.GetString("out")

		if kid == "" {
			keys, err := client.VendorKeys(c.Context(), vendorID)
			if err != nil {
				return err
			}

			var active []string
			for _, k := range keys {
				if k.IsActive {
					active = append(active, k.Kid)
				}
			}
			switch len(active) {
			case 0:
				return errors.New("no active vendor keys found; pass --kid explicitly")
			case 1:
				kid = active[0]
			default:
				return fmt.Errorf("multiple active vendor keys found (%v); pass --kid to pick one", active)
			}
		}

		pem, err := client.VendorPublicKeyPEM(c.Context(), vendorID, kid)
		if err != nil {
			return err
		}

		if out == "" {
			fmt.Print(pem)
			return nil
		}

		if err := os.WriteFile(out, []byte(pem), 0o644); err != nil {
			return err
		}
		fmt.Printf("→ wrote %s (kid %s)\n", out, kid)
		return nil
	},
}

func init() {
	publicKeyCmd.Flags().String("vendor-id", "", vendorIDFlagUsage)
	publicKeyCmd.Flags().String("kid", "", "specific key ID; auto-selected if there's exactly one active key")
	publicKeyCmd.Flags().String("out", "", "write the PEM to a file instead of stdout")

	keysCreateCmd.Flags().String("vendor-id", "", vendorIDFlagUsage)
	keysCreateCmd.Flags().Bool("yes", false, "skip the confirmation prompt")
	keysListCmd.Flags().String("vendor-id", "", vendorIDFlagUsage)
	keysCmd.AddCommand(keysListCmd, keysCreateCmd)

	licensesCmd.AddCommand(publicKeyCmd, keysCmd)
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List and generate the vendor's Ed25519 signing keys",
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the vendor's signing keys",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		vendorID, err := resolveVendorID(c)
		if err != nil {
			return err
		}
		asJSON, _ := c.Flags().GetBool("json")

		keys, err := client.VendorKeys(c.Context(), vendorID)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(keys)
		}

		if len(keys) == 0 {
			fmt.Println("No signing keys. Generate one with `forgebit licenses keys create`.")
			return nil
		}
		for _, k := range keys {
			state := "inactive"
			if k.IsActive {
				state = "active"
			}
			fmt.Printf("%s  %s  %s  created:%s\n", k.Kid, k.Algorithm, state, k.CreatedAt)
		}
		return nil
	},
}

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Generate a new signing key (limited to one per 10 minutes)",
	Args:  cobra.NoArgs,
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		vendorID, err := resolveVendorID(c)
		if err != nil {
			return err
		}
		asJSON, _ := c.Flags().GetBool("json")
		yes, _ := c.Flags().GetBool("yes")

		if !yes {
			fmt.Fprintln(c.ErrOrStderr(), "warning: apps that verify licenses offline may need this new key's public key,")
			fmt.Fprintln(c.ErrOrStderr(), "and `licenses public-key` will need --kid once more than one key is active.")
			if !confirm(c, fmt.Sprintf("Generate a new signing key for vendor %s?", vendorID)) {
				return errors.New("aborted; no key was generated (pass --yes to skip this prompt)")
			}
		}

		result, err := client.CreateVendorKey(c.Context(), vendorID)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("→ generated key %s\n", result.Kid)
		fmt.Printf("  fetch its public key with `forgebit licenses public-key --kid %s`\n", result.Kid)
		return nil
	},
}
