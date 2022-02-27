package container

import (
	"context"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove <container-id>",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.RemoveContainer(
			context.Background(),
			&server.RemoveContainerRequest{
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
	baseCmd.AddCommand(removeCmd)
}
