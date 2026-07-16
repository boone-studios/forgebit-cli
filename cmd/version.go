package cmd

// Set via -ldflags "-X github.com/boone-studios/forgebit-cli/cmd.version=1.2.3" at release build time
var version = "dev"

func init() {
	rootCmd.Version = version
}
