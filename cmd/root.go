package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "drive-syncer",
	Short: "Sync local folders with Google Drive",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'drive-syncer sync' to start syncing.")
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
