package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/boone-studios/forgebit-cli/internal/config"
	"github.com/spf13/cobra"
)

func TestResolveVendorID(t *testing.T) {
	savedCfg, savedVendor := cfg, vendorFlag
	t.Cleanup(func() { cfg, vendorFlag = savedCfg, savedVendor })

	profiles := []config.VendorProfile{{VendorID: "vnd_1", VendorName: "Acme"}, {VendorID: "vnd_2", VendorName: "Globex"}}
	cases := []struct {
		name, flag, vendor, token, want string
	}{
		{name: "explicit flag wins", flag: "vnd_9", vendor: "Globex", token: "t", want: "vnd_9"},
		{name: "--vendor by name", vendor: "globex", want: "vnd_2"},
		{name: "active profile", want: "vnd_1"},
		{name: "explicit token can't guess", token: "t", want: ""},
		{name: "unknown --vendor", vendor: "nope", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg = config.Config{Profiles: profiles, ActiveVendorID: "vnd_1", Token: tc.token}
			vendorFlag = tc.vendor
			c := &cobra.Command{}
			c.Flags().String("vendor-id", tc.flag, "")

			got, err := resolveVendorID(c)
			if got != tc.want || (tc.want == "") != (err != nil) {
				t.Fatalf("got %q, err %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	for input, want := range map[string]bool{"y\n": true, "YES\n": true, " yes ": true, "n\n": false, "\n": false, "": false, "yep\n": false} {
		c := &cobra.Command{}
		c.SetIn(strings.NewReader(input))
		c.SetErr(io.Discard)
		if got := confirm(c, "ok?"); got != want {
			t.Errorf("confirm(%q) = %v, want %v", input, got, want)
		}
	}
}
