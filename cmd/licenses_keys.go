package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var publicKeyCmd = &cobra.Command{
	Use:   "public-key",
	Short: "Fetch a vendor's Ed25519 public key for offline verification",
	RunE: func(c *cobra.Command, args []string) error {
		client, err := requireAuth()
		if err != nil {
			return err
		}

		flags := c.Flags()
		vendorID, _ := flags.GetString("vendor-id")
		kid, _ := flags.GetString("kid")
		out, _ := flags.GetString("out")

		if vendorID == "" {
			return errors.New("--vendor-id is required (find it in the output of `licenses issue`, `show`, `list`, or `verify`)")
		}

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
	publicKeyCmd.Flags().String("vendor-id", "", "vendor ID (required)")
	publicKeyCmd.Flags().String("kid", "", "specific key ID; auto-selected if there's exactly one active key")
	publicKeyCmd.Flags().String("out", "", "write the PEM to a file instead of stdout")
	_ = publicKeyCmd.MarkFlagRequired("vendor-id")

	licensesCmd.AddCommand(publicKeyCmd)
}
