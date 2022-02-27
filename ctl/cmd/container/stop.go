package container

import (
	"context"
	"github.com/spf13/cobra"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop <container-id>",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.StopContainer(
			context.Background(),
			&server.StopContainerRequest{
				ContainerId: args[0],
			},
		)
		if err != nil {
			klog.Fatal("Command failed with err:%v", err)
		}
		cmdutil.Print(resp)
	},
}

func init() {
	baseCmd.AddCommand(stopCmd)
}
