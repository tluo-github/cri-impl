package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印 cri-impl 版本",
	Long:  `打印 cri-impl 版本`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("cri-impl version 0.0.1")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
