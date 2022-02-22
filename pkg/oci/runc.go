package oci

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tluo-github/cri-impl/pkg/container"
	"github.com/tluo-github/cri-impl/pkg/timeutil"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

// runcRuntime 实现 oci.Runtime
type runcRuntime struct {
	// shimmy 执行路径, eg: /usr/local/bin/shimmy
	shimmyPath string
	// runc 执行路径, eg: /usr/bin/runc
	runtimePath string
	// container 状态存储目录,eg /run/runc/
	rootPath string
}

func NewRuntime(shimmyPath string,
	runtimePath string,
	rootPath string,
) Runtime {
	return &runcRuntime{
		shimmyPath:  shimmyPath,
		runtimePath: runtimePath,
		rootPath:    rootPath,
	}
}

func (r runcRuntime) CreateContainer(
	id container.ID,
	bundleDir string,
	logfile string,
	exitfile string,
	attachfile string,
	stdin bool,
	stdinOnce bool,
	timeout time.Duration,
) (pid int, err error) {
	cmd := exec.Command(
		r.shimmyPath,
		"--shimmy-pidfile", path.Join(bundleDir, "shimmy.pid"),
		"--shimmy-log-level", strings.ToUpper("info"),
		"--runtime", r.runtimePath,
		fmt.Sprintf("--runtime-arg='--root=%s'", r.rootPath),
		"--bundle", bundleDir,
		"--container-id", string(id),
		"--container-pidfile", path.Join(bundleDir, "container.pid"),
		"--container-logfile", logfile,
		"--container-exitfile", exitfile,
		"--container-attachfile", attachfile,
	)
	if stdin {
		cmd.Args = append(cmd.Args, "--stdin")
	}
	if stdinOnce {
		cmd.Args = append(cmd.Args, "--stdin-once")
	}

	syncpipeRead, syncpipeWrite, err := os.Pipe()
	if err != nil {
		return 0, err
	}
	defer syncpipeRead.Close()
	defer syncpipeWrite.Close()

	cmd.ExtraFiles = append(cmd.ExtraFiles, syncpipeWrite)
	cmd.Args = append(
		cmd.Args,
		"--syncpipe-fd", strconv.Itoa(2+len(cmd.ExtraFiles)),
	)
	// 我们预计 shimmy 的执行几乎是即时的,因为它的主进程只是验证输入参数
	// fork shim 进程处理,将其  PID 保持在磁盘上,然后退出
	if _, err := runCommand(cmd); err != nil {
		return 0, err
	}
	syncpipeWrite.Close()

	type Report struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
		Stderr string `json:"stderr"`
		Pid    int    `json:"pid"`
	}
	err = timeutil.WithTimeout(timeout, func() error {
		bytes, err := ioutil.ReadAll(syncpipeRead)
		if err != nil {
			return err
		}
		syncpipeRead.Close()

		report := Report{}
		if err := json.Unmarshal(bytes, &report); err != nil {
			return errors.Wrap(
				err,
				fmt.Sprintf("Failed to decode report string [%v]. Raw[%v].",
					string(bytes), bytes),
			)
		}
		if report.Kind == "container_pid" && report.Pid > 0 {
			pid = report.Pid
			return nil
		}
		return errors.Errorf("%+v", report)
	})
	return pid, err
}

func (r runcRuntime) StartContainer(id container.ID) error {
	cmd := exec.Command(
		r.runtimePath,
		"--root", r.rootPath,
		"start", string(id),
	)

	_, err := runCommand(cmd)
	return err
}

func (r runcRuntime) KillContainer(id container.ID, sig os.Signal) error {
	sigstr, err := sigStr(sig)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		r.runtimePath,
		"--root", r.rootPath,
		"kill",
		string(id),
		sigstr,
	)
	_, err = runCommand(cmd)
	return err
}

func (r runcRuntime) DeleteContainer(id container.ID) error {
	cmd := exec.Command(
		r.runtimePath,
		"--root", r.rootPath,
		"delete",
		string(id),
	)
	_, err := runCommand(cmd)
	return err
}

func (r runcRuntime) ContainerState(id container.ID) (StateResp, error) {
	cmd := exec.Command(
		r.runtimePath,
		"--root", r.rootPath,
		"state",
		string(id),
	)
	output, err := runCommand(cmd)
	if err != nil {
		return StateResp{}, err
	}
	resp := StateResp{}
	return resp, json.Unmarshal(output, &resp)
}

func runCommand(cmd *exec.Cmd) ([]byte, error) {
	output, err := cmd.Output()
	debugLog(cmd, output, err)
	return output, wrappedError(err)
}

func debugLog(c *exec.Cmd, stdout []byte, err error) {
	stderr := []byte{}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
	}
	klog.Infof("stdout:%v stderr:%s error:%v exec %s", string(stdout), string(stderr), err, strings.Join(c.Args, " "))

}

func wrappedError(err error) error {
	if err == nil {
		return nil
	}
	msg := "OCI runtime (runc) execution failed"
	if ee, ok := err.(*exec.ExitError); ok {
		msg = fmt.Sprintf("%v,stderr=[%v]", msg, string(ee.Stderr))
	}
	return errors.Wrap(err, msg)
}
