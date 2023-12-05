package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootFlags = struct {
	verbose bool
}{}

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&rootFlags.verbose, "verbose", "v", false, "Verbose client output",
	)
}

var rootCmd = &cobra.Command{
	Use: "a0feed",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
