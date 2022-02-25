/*
Copyright © 2022 tluo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tluo-github/cri-impl/config"
	"github.com/tluo-github/cri-impl/pkg/cri"
	"github.com/tluo-github/cri-impl/pkg/fsutil"
	"github.com/tluo-github/cri-impl/pkg/oci"
	"github.com/tluo-github/cri-impl/pkg/storage"
	"github.com/tluo-github/cri-impl/server"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
)

var cfg config.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cri-impl",
	Short: "cri-impl 是一个简单的 container manager",
	Long:  `cri-impl 是一个简单的 container manager,像CRI-O or containerd, 仅用于学习研究`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		klog.Infof("cri-impl here!")

		runtime := oci.NewRuntime(
			fsutil.AssertExists(cfg.ShimmyPath),
			fsutil.AssertExists(cfg.RuntimePath),
			fsutil.AssertExists(cfg.RuntimeRoot),
		)
		cstore := storage.NewContainerStore(fsutil.EnsureExists(cfg.LibRoot))
		logDir := fsutil.EnsureExists(cfg.ContainerLogRoot)
		exitDir := fsutil.EnsureExists(cfg.RunRoot, "exits")
		attachDir := fsutil.EnsureExists(cfg.RunRoot, "attach")

		rs, err := cri.NewRuntimeService(runtime, cstore, logDir, exitDir, attachDir)
		if err != nil {
			klog.Fatalf("%v", err)
		}

		sscfg := streaming.DefaultConfig
		sscfg.Addr = cfg.StreamingAddr
		ss, err := streaming.NewServer(sscfg, rs)
		if err != nil {
			klog.Fatalf("%v", err)
		}

		go ss.Start(true)

		criServer := server.New(rs, ss)
		if err := criServer.Serve("unix", cfg.Listen); err != nil {
			klog.Fatalf("criserver serve error %v", err)
		}
		klog.Infof("cri-impl start ok! ")
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().StringVarP(&cfg.Listen, "listen", "l", config.DefaultListen, "守护进程监听 sock 地址")
	rootCmd.Flags().StringVarP(&cfg.LibRoot, "lib-root", "b", config.DefaultLibRoot, "持久数据的根目录,如 container bundles 等.")
	rootCmd.Flags().StringVarP(&cfg.RunRoot, "run-root", "n", config.DefaultRunRoot, "运行时数据的根目录,如 sock 和 pid 文件")
	rootCmd.Flags().StringVarP(&cfg.ContainerLogRoot, "container-logs", "L", config.DefaultContainerLogRoot, "容器日志根目录")
	rootCmd.Flags().StringVarP(&cfg.StreamingAddr, "streaming-addr", "S", config.DefaultStreaminAddr, "流服务 host:port( for attach,exec,port-forwarding)")
	rootCmd.Flags().StringVarP(&cfg.ShimmyPath, "shimmy-path", "s", config.DefaultShimmyPath, "OCI 运行时 shim 可执行文件(shimmy)")
	rootCmd.Flags().StringVarP(&cfg.RuntimePath, "runtime-path", "r", config.DefaultRuntimePath, "OCI 运行时可执行文件(runc)")
	rootCmd.Flags().StringVarP(&cfg.RuntimeRoot, "runtime-root", "t", config.DefaultRuntimeRoot, "OCI 运行时根目录")
}
