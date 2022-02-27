package container

import (
	"context"
	"github.com/spf13/cobra"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start <container-id>",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.StartContainer(
			context.Background(),
			&server.StartContainerRequest{
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
	baseCmd.AddCommand(startCmd)
}
