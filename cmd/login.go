package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/boone-studios/forgebit-cli/internal/config"
	"github.com/boone-studios/forgebit-cli/internal/forgebit"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with your Forgebit account",
	RunE: func(c *cobra.Command, args []string) error {
		if cfg.APIBaseURL == "" {
			return errors.New("no API URL configured; pass --api-url or set it in config")
		}

		client := forgebit.NewDeviceAuthClient(cfg.APIBaseURL)

		auth, err := client.RequestAuthorization(c.Context(), "forgebit-cli")
		if err != nil {
			return err
		}

		fmt.Printf("First, confirm this code: %s\n", auth.UserCode)
		fmt.Printf("Opening %s in your browser...\n", auth.VerificationURIComplete)
		if err := browser.OpenURL(auth.VerificationURIComplete); err != nil {
			fmt.Println("Couldn't open a browser automatically — open the URL above manually.")
		}
		fmt.Println("Waiting for approval...")

		result, err := pollForResult(c.Context(), client, auth)
		if err != nil {
			return err
		}

		previous, hadPrevious := cfg.ProfileForVendor(result.VendorID)

		newCfg := cfg
		newCfg.Token = result.Token // Legacy field, kept in sync for back-compat
		newCfg.UpsertProfile(config.VendorProfile{
			VendorID:   result.VendorID,
			VendorName: result.VendorName,
			Token:      result.Token,
		})
		newCfg.ActiveVendorID = result.VendorID
		if err := config.Save(newCfg); err != nil {
			return err
		}

		if hadPrevious && previous.Token != result.Token {
			revokeBestEffort(c.Context(), previous.Token)
		}

		fmt.Printf("Logged in as %s (vendor %s). This is now your active vendor.\n", result.VendorName, result.VendorID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func pollForResult(ctx context.Context, client *forgebit.DeviceAuthClient, auth forgebit.DeviceAuthorization) (forgebit.PollResult, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(auth.ExpiresIn*float64(time.Second)))
	defer cancel()

	interval := time.Duration(auth.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	for {
		result, err := client.Poll(ctx, auth.DeviceCode)
		if err != nil {
			var rateLimited *forgebit.RateLimitedError
			if errors.As(err, &rateLimited) {
				select {
				case <-ctx.Done():
					return forgebit.PollResult{}, errors.New("timed out waiting for approval; run `forgebit login` again")
				case <-time.After(interval):
					continue
				}
			}
			return forgebit.PollResult{}, err
		}

		switch result.Status {
		case forgebit.PollStatusPending:
			select {
			case <-ctx.Done():
				return forgebit.PollResult{}, errors.New("timed out waiting for approval; run `forgebit login` again")
			case <-time.After(interval):
				continue
			}
		case forgebit.PollStatusApproved:
			return result, nil
		case forgebit.PollStatusDenied:
			return forgebit.PollResult{}, errors.New("login was denied")
		case forgebit.PollStatusExpired:
			return forgebit.PollResult{}, errors.New("the code expired; run `forgebit login` again")
		default:
			return forgebit.PollResult{}, fmt.Errorf("unexpected authorization status: %s", result.Status)
		}
	}
}
