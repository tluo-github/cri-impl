package container

import (
	"context"
	"github.com/spf13/cobra"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status <container-id>",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.ContainerStatus(
			context.Background(),
			&server.ContainerStatusRequest{
				ContainerId: args[1],
			},
		)
		if err != nil {
			klog.Fatal("Command failed with err:%v", err)
		}
		cmdutil.Print(resp)
	},
}

func init() {
	baseCmd.AddCommand(statusCmd)
}
