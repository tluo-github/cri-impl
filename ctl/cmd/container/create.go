package container

import (
	"context"
	"github.com/spf13/cobra"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [command options] <container-name> -- <command> [args...]",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.CreateContainer(
			context.Background(),
			&server.CreateContainerRequest{
				Name:           args[0],
				RootfsPath:     opts.Rootfs,
				RootfsReadonly: opts.RootfsReadonly,
				Command:        args[1],
				Args:           args[2:],
				Stdin:          opts.Stdin,
				StdinOnce:      !opts.LeaveStdinOpen,
			},
		)
		if err != nil {
			klog.Fatal("Command failed with err:%v", err)
		}
		cmdutil.Print(resp)
	},
}

func init() {
	createCmd.PersistentFlags().StringVarP(&opts.Rootfs,
		"image", "I",
		"",
		"Container rootfs image (必须)")
	createCmd.PersistentFlags().BoolVarP(&opts.RootfsReadonly,
		"rootfs-readonly", "R",
		true,
		"容器是否可以修改其 rootfs")

	createCmd.PersistentFlags().BoolVarP(&opts.Stdin,
		"stdin", "i",
		false,
		"保持容器的 STDIN 打开（交互模式）")
	createCmd.PersistentFlags().BoolVarP(&opts.LeaveStdinOpen,
		"leave-stdin-open", "",
		false,
		"在第一个attach session 完成后保持容器的 STDIN 打开")

	baseCmd.AddCommand(createCmd)
}
