package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/boone-studios/forgebit-cli/internal/licenseverify"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [key]",
	Short: "Verify a license key",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		flags := c.Flags()
		licenseID, _ := flags.GetString("license-id")
		productID, _ := flags.GetString("product-id")
		offline, _ := flags.GetBool("offline")
		publicKeyPath, _ := flags.GetString("public-key")
		asJSON, _ := flags.GetBool("json")

		var key string
		if len(args) == 1 {
			key = args[0]
		}
		if key == "" && licenseID == "" {
			return errors.New("provide a license key or --license-id")
		}

		if offline {
			return runOfflineVerify(key, publicKeyPath, asJSON)
		}

		client, err := requireAuth()
		if err != nil {
			return err
		}

		result, err := client.VerifyLicenseOnline(c.Context(), forgebit.VerifyLicenseParams{
			Key:       key,
			LicenseID: licenseID,
			ProductID: productID,
		})
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if result.Valid {
			fmt.Printf("✓ valid — tier %s — expires %s\n", result.Tier, formatExpiry(result.ExpiresAt))
			return nil
		}

		reason := "unknown"
		if result.Reason != nil {
			reason = *result.Reason
		}
		fmt.Printf("✗ invalid — reason: %s\n", reason)
		return nil
	},
}

func runOfflineVerify(key, publicKeyPath string, asJSON bool) error {
	if key == "" {
		return errors.New("--offline requires a license key argument (not --license-id, which needs the API)")
	}
	if publicKeyPath == "" {
		return errors.New("--offline requires --public-key <file>; fetch one first with `forgebit licenses public-key`")
	}

	pub, err := licenseverify.LoadPublicKeyPEM(publicKeyPath)
	if err != nil {
		return fmt.Errorf("loading public key: %w", err)
	}

	result := licenseverify.Verify(key, pub)

	if asJSON {
		return printJSON(result)
	}

	if !result.Valid {
		fmt.Printf("✗ invalid — reason: %s\n", result.Reason)
		return nil
	}

	switch result.Format {
	case licenseverify.FormatJWT:
		fmt.Printf("✓ valid (offline, jwt) — tier %s — product %s — expires %s\n",
			result.JWTClaims.Tier, result.JWTClaims.Product, formatTime(result.JWTClaims.ExpiresAt))
	case licenseverify.FormatForgebit:
		fmt.Printf("✓ valid (offline, forgebit) — license %s — vendor %s — expires %s\n",
			result.ForgebitFields.LicenseID, result.ForgebitFields.VendorID, formatTime(result.ForgebitFields.ExpiresAt))
	}
	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.Format(time.RFC3339)
}

func init() {
	verifyCmd.Flags().String("license-id", "", "verify by license ID instead of a key (online only)")
	verifyCmd.Flags().String("product-id", "", "restrict the online check to a specific product")
	verifyCmd.Flags().Bool("offline", false, "verify locally with no network call (jwt/forgebit key types only)")
	verifyCmd.Flags().String("public-key", "", "vendor public key PEM file, required with --offline")
	verifyCmd.Flags().Bool("json", false, "print the raw result as JSON")

	licensesCmd.AddCommand(verifyCmd)
}
