package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/spf13/cobra"
)

const (
	releasePollInterval = 2 * time.Second
	releaseWaitTimeout  = 10 * time.Minute
)

var productsReleaseCmd = &cobra.Command{
	Use:   "release <product-id>",
	Short: "Upload a new product version and wait for it to finish processing",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		flags := c.Flags()
		path, _ := flags.GetString("file")
		version, _ := flags.GetString("version")
		channel, _ := flags.GetString("channel")
		requiresPlatform, _ := flags.GetString("requires-platform")
		testedUpTo, _ := flags.GetString("tested-up-to")
		requiresRuntime, _ := flags.GetString("requires-runtime")
		changelog, _ := flags.GetString("changelog")
		security, _ := flags.GetBool("security")
		rollout, _ := flags.GetInt("rollout")
		noWait, _ := flags.GetBool("no-wait")
		asJSON, _ := flags.GetBool("json")

		params := forgebit.CreateVersionParams{
			Version:          version,
			Channel:          channel,
			RequiresPlatform: requiresPlatform,
			TestedUpTo:       testedUpTo,
			RequiresRuntime:  requiresRuntime,
			Changelog:        changelog,
			IsSecurity:       security,
		}
		if flags.Changed("rollout") {
			params.RolloutPercent = new(rollout)
		}

		// Progress goes to stderr so --json output stays clean on stdout
		result, err := releaseVersion(c.Context(), client, args[0], path, params, !noWait, releasePollInterval, os.Stderr)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		if result.Status == "pending" {
			fmt.Printf("→ %s is still processing (version %s)\n", result.Version, result.ID)
			return nil
		}
		fmt.Printf("✓ %s is ready on %s (version %s)\n", result.Version, result.Channel, result.ID)
		return nil
	},
}

func releaseVersion(ctx context.Context, client *forgebit.APIClient, productID, path string, params forgebit.CreateVersionParams, wait bool, pollEvery time.Duration, progress io.Writer) (forgebit.ProductVersion, error) {
	file, err := os.Open(path)
	if err != nil {
		return forgebit.ProductVersion{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return forgebit.ProductVersion{}, err
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return forgebit.ProductVersion{}, fmt.Errorf("hashing %s: %w", path, err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return forgebit.ProductVersion{}, err
	}

	created, err := client.CreateProductVersion(ctx, productID, params)
	if err != nil {
		return forgebit.ProductVersion{}, err
	}
	version := created.Data
	if version.Upload == nil {
		return forgebit.ProductVersion{}, errors.New("the API didn't return an upload URL for this version")
	}
	fmt.Fprintf(progress, "→ created %s (version %s)\n", version.Version, version.ID)

	// The version now exists, so a failure past here can't be fixed by re-running with the same --version
	if err := forgebit.UploadVersionArtifact(ctx, *version.Upload, file, info.Size()); err != nil {
		return forgebit.ProductVersion{}, fmt.Errorf("%w (version %s was created but has no artifact)", err, version.ID)
	}
	fmt.Fprintf(progress, "→ uploaded %s (%d bytes, sha256 %s)\n", path, info.Size(), checksum)

	completed, err := client.CompleteProductVersion(ctx, productID, version.ID, checksum)
	if err != nil {
		return forgebit.ProductVersion{}, fmt.Errorf("%w (version %s)", err, version.ID)
	}
	if !wait {
		return completed.Data, nil
	}
	fmt.Fprintln(progress, "→ finalizing…")

	ctx, cancel := context.WithTimeout(ctx, releaseWaitTimeout)
	defer cancel()
	for {
		shown, err := client.ShowProductVersion(ctx, productID, version.ID)
		if err != nil {
			return forgebit.ProductVersion{}, err
		}

		switch shown.Data.Status {
		case "pending":
		case "failed":
			// The show endpoint doesn't include a reason; the product_version.failed webhook does
			return shown.Data, fmt.Errorf("version %s failed processing (checksum or plugin Version: header mismatch); see the product_version.failed webhook for the reason", version.ID)
		default:
			return shown.Data, nil
		}

		select {
		case <-ctx.Done():
			return forgebit.ProductVersion{}, fmt.Errorf("timed out waiting for version %s to finish processing", version.ID)
		case <-time.After(pollEvery):
		}
	}
}

func init() {
	productsReleaseCmd.Flags().String("file", "", "artifact to upload, e.g. plugin.zip (required)")
	productsReleaseCmd.Flags().String("version", "", "version string, unique per product + channel (required)")
	productsReleaseCmd.Flags().String("channel", "", "release channel: stable or beta (server default: stable)")
	productsReleaseCmd.Flags().String("requires-platform", "", "minimum host platform version, e.g. WordPress 6.0")
	productsReleaseCmd.Flags().String("tested-up-to", "", "highest host platform version tested against")
	productsReleaseCmd.Flags().String("requires-runtime", "", "minimum runtime version, e.g. PHP 8.2")
	productsReleaseCmd.Flags().String("changelog", "", "plain-text changelog for this release")
	productsReleaseCmd.Flags().Bool("security", false, "flag this release as a security fix")
	productsReleaseCmd.Flags().Int("rollout", 0, "percent of licenses to serve this version to (server default: 100)")
	productsReleaseCmd.Flags().Bool("no-wait", false, "return after upload instead of waiting for processing to finish")
	_ = productsReleaseCmd.MarkFlagRequired("file")
	_ = productsReleaseCmd.MarkFlagRequired("version")

	productsCmd.AddCommand(productsReleaseCmd)
}
