package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := Connect()
		defer conn.Close()

		resp, err := client.Version(
			context.Background(),
			&server.VersionRequest{},
		)
		if err != nil {
			klog.Fatal("Command failed with err:%v", err)
		}
		Print(resp)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
