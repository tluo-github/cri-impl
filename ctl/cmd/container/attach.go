package container

import (
	"context"
	"github.com/spf13/cobra"
	cmdutil "github.com/tluo-github/cri-impl/ctl/cmd"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
	"net/url"
	"os"
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach <container-id>",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		client, conn := cmdutil.Connect()
		defer conn.Close()

		resp, err := client.Attach(
			context.Background(),
			&server.AttachRequest{
				ContainerId: args[0],
				Tty:         false,
				Stdin:       opts.Stdin,
				Stdout:      true,
				Stderr:      true,
			},
		)
		if err != nil {
			klog.Fatal("Command failed with err:%v", err)
		}

		url, err := url.Parse(resp.Url)
		if err != nil {
			klog.Fatal("Failed to parse stream URL with err:%v", err)
		}
		executor, err := remotecommand.NewSPDYExecutor(
			&rest.Config{
				TLSClientConfig: rest.TLSClientConfig{Insecure: true},
			},
			"POST",
			url,
		)
		if err != nil {
			klog.Fatal("Failed to create stream executor with err:%v", err)
		}

		streamOptions := remotecommand.StreamOptions{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Tty:    false,
		}

		if err := executor.Stream(streamOptions); err != nil {
			klog.Fatal("executor.Stream() failed")
		}

	},
}

func init() {
	attachCmd.PersistentFlags().BoolVarP(&opts.Stdin,
		"stdin", "i",
		false,
		"将 stdin 传递给容器")
	baseCmd.AddCommand(attachCmd)
}
