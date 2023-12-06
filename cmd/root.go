package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootFlags = struct {
	verbose    bool
	configFile string
}{}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.verbose, "verbose", "v", false, "Verbose client output")
	rootCmd.PersistentFlags().StringVarP(&rootFlags.configFile, "config", "c", "file:///$(pwd)/config.yml", "Config file URI. Supported schemes: file://, gs://")
	// _ = rootCmd.MarkPersistentFlagRequired("config")
}

var rootCmd = &cobra.Command{
	Use:  "a0feed",
	Long: `A0 Feed a news translation service`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
