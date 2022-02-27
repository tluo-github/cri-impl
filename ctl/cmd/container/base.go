package container

import (
	"fmt"
	"github.com/tluo-github/cri-impl/ctl/cmd"

	"github.com/spf13/cobra"
)

type Options struct {
	Rootfs         string
	RootfsReadonly bool
	Command        string
	Stdin          bool
	LeaveStdinOpen bool
}

var opts Options

// baseCmd represents the base command
var baseCmd = &cobra.Command{
	Use:   "container",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Missed or unknown container command.\n\n")
		cmd.Help()
	},
}

func init() {
	cmd.RootCmd.AddCommand(baseCmd)
}
