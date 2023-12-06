package cmd

import (
	"broadcaster/utils/info"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints app version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(info.Print("\n"))
	},
}
