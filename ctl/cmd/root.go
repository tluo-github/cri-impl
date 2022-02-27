package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var OptHost string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "crictl",
	Short: "crictl - 与cri-impl守护进程 sock 通信的CLI 工具",
	Long:  `crictl - 与cri-impl守护进程 sock 通信的CLI 工具`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Missed or unknown command.\n\n")
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(RootCmd.Execute())
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&OptHost,
		"host", "H",
		"/var/run/cri-impl.sock",
		"cri-impl 守护进程 sock")
}
