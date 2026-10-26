package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <license-id>",
	Short: "Download a license's signed license file",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}

		out, _ := c.Flags().GetString("out")
		force, _ := c.Flags().GetBool("force")

		body, filename, err := client.DownloadLicenseFile(c.Context(), args[0])
		if err != nil {
			return err
		}
		defer body.Close()

		if out == "-" {
			_, err := io.Copy(os.Stdout, body)
			return err
		}
		if out == "" {
			out = filename
		}
		if out == "" {
			out = args[0] + ".lic"
		}

		n, err := writeNewFile(out, body, force)
		if err != nil {
			return err
		}
		fmt.Printf("→ saved %s (%d bytes)\n", out, n)
		return nil
	},
}

// Refuses to clobber an existing file unless force is set, and removes a partial file on failure
func writeNewFile(path string, r io.Reader, force bool) (int64, error) {
	mode := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		mode = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, mode, 0o600)
	if os.IsExist(err) {
		return 0, fmt.Errorf("%s already exists; pass --force to overwrite it or -o to pick another name", path)
	}
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(file, r)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return 0, err
	}
	return n, nil
}

var fileLinkCmd = &cobra.Command{
	Use:   "file-link <license-id>",
	Short: "Create a temporary web link to download a license's file",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		asJSON, _ := c.Flags().GetBool("json")

		link, err := client.LicenseFileLink(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(link)
		}

		fmt.Println(link.URL)
		fmt.Fprintf(os.Stderr, "expires %s\n", link.ExpiresAt)
		return nil
	},
}

var downloadLinkCmd = &cobra.Command{
	Use:   "download-link <license-id>",
	Short: "Create a temporary link to the product artifact (e.g. plugin ZIP) this license covers",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		channel, _ := c.Flags().GetString("channel")
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.ProductDownloadLink(c.Context(), args[0], channel)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println(result.Data.DownloadURL)
		fmt.Fprintf(os.Stderr, "version %s (%s), expires %s\n", result.Data.Version, result.Data.Channel, result.Data.ExpiresAt)
		return nil
	},
}

var uploadFileCmd = &cobra.Command{
	Use:   "upload-file <license-id> <file>",
	Short: "Attach your own license file (.lic, .json, .enc, or .xml, max 5 MB) instead of the generated one",
	Args:  cobra.ExactArgs(2),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		asJSON, _ := c.Flags().GetBool("json")

		file, err := os.Open(args[1])
		if err != nil {
			return err
		}
		defer file.Close()

		result, err := client.UploadLicenseFile(c.Context(), args[0], args[1], file)
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Printf("→ uploaded %s\n", args[1])
		return nil
	},
}

var deleteFileCmd = &cobra.Command{
	Use:   "delete-file <license-id>",
	Short: "Remove a previously uploaded license file",
	Args:  cobra.ExactArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		client, err := resolveAPIClient()
		if err != nil {
			return err
		}
		asJSON, _ := c.Flags().GetBool("json")

		result, err := client.DeleteUploadedLicenseFile(c.Context(), args[0])
		if err != nil {
			return err
		}

		if asJSON {
			return printJSON(result)
		}

		fmt.Println("→ deleted uploaded license file")
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringP("out", "o", "", "file to write, or - for stdout (default: the server's filename)")
	downloadCmd.Flags().Bool("force", false, "overwrite the output file if it exists")

	downloadLinkCmd.Flags().String("channel", "", "release channel: stable or beta (server default: stable)")

	licensesCmd.AddCommand(downloadCmd, fileLinkCmd, downloadLinkCmd, uploadFileCmd, deleteFileCmd)
}
