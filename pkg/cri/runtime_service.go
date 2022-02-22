package cri

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/tluo-github/cri-impl/pkg/container"
	"github.com/tluo-github/cri-impl/pkg/oci"
	"github.com/tluo-github/cri-impl/pkg/rollback"
	"github.com/tluo-github/cri-impl/pkg/shimutil"
	"github.com/tluo-github/cri-impl/pkg/storage"
	"io/ioutil"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
	"path"
	"sort"
	"sync"
	"syscall"
	"time"
)

// RuntimeService 是管理 manager container 和 sandbox runtimes 的服务
// 类似于CRI runtime interface,但不严格遵循它
type RuntimeService interface {
	// CreateContainer 在disk 上准备一个新的 container bundle,并启动runc init，但不启动指定进程
	CreateContainer(options ContainerOptions) (*container.Container, error)

	// StartContainer 实际上通过CreateContainer()创建的容器中启动一个预定义的进程
	StartContainer(id container.ID) error

	// StopContainer 向container发出信号优雅停机
	StopContainer(id container.ID, timeout time.Duration) error

	// RemoveContainer 从 cri-impl 和 runc storages 中删除 container，
	// 如果container 没有停止,必须设置强制标志.
	// 如果container 已经被移出,则不返回错误,保持 幂等行为
	RemoveContainer(id container.ID) error

	ListContainers() ([]*container.Container, error)
	// GetContainer 从 OCI 获得 container
	GetContainer(id container.ID) (*container.Container, error)

	streaming.Runtime
}

type ContainerOptions struct {
	Name            string
	Command         string
	Args            []string
	RootfsPath      string
	RootsfsReadOnly bool
	Stdin           bool
	StdinOnce       bool
}

// runtimeService 实现 RuntimeService
// 一些设计注意事项
// - runtimeService 方法是线程安全的,有一个公共 lock 防止并发的容器修改,
//     有了这个 lock 像 container.Map、storage.ContainerStore 这样依赖可以被简化并省略他们的锁
// - runtimeService 自行跟踪容器状态,它使用 ContainerStore 在容器基础目录中写入 JSON 保存容器状态.
//      由于状态和 runc 执行写入不是原子的，首先发生状态修改(乐观锁),然后是runc 命令,如果出现 runc error ,
//      就回滚状态，但是在级联故障期间,保存在容器目录中的状态和容器根据runc 的状态可能会出现分歧。状态恢复逻辑应该尝试修复映入的差异。
// - ContainerStore 是唯一的事实来源。只跟踪 由 cri-impl 管理的容器,如果有人使用相同的配置用 runc 创建额外的容器，cri-impl 将看不见更改。
// 有三层存储
// 第一层 in memory map
// 第二次 in disk store
// 第三层 runc
type runtimeService struct {
	lock      sync.Mutex
	runtime   oci.Runtime
	cstore    storage.ContainerStore
	logDir    string
	exitDir   string
	attachDir string

	cmap *container.Map
}

func NewRuntimeService(
	runtime oci.Runtime,
	cstore storage.ContainerStore,
	logDir string,
	exitDir string,
	attachDir string) (RuntimeService, error) {
	rs := &runtimeService{
		runtime:   runtime,
		cstore:    cstore,
		logDir:    logDir,
		exitDir:   exitDir,
		attachDir: attachDir,
		cmap:      container.NewMap(),
	}
	if err := rs.restore(); err != nil {
		return nil, err
	}
	return rs, nil
}

func (rs *runtimeService) CreateContainer(options ContainerOptions) (cont *container.Container, err error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	rb := rollback.New()
	defer func() { _ = err == nil || rb.Execute() }()

	// UUID 生产容器ID
	contID := container.RandID()
	// 创建容器
	cont, err = container.New(
		contID,
		options.Name,
		rs.containerLogFile(contID),
	)
	if err != nil {
		return
	}
	// 添加进缓存
	if err = rs.cmap.Add(cont, rb); err != nil {
		return
	}
	// 在磁盘上创建容器目录
	hcont, err := rs.cstore.CreateContainer(cont.ID(), rb)
	if err != nil {
		return
	}

	// 生产容器 spec
	spec, err := oci.NewSpec(oci.SpecOptions{
		Command:      options.Command,
		Args:         options.Args,
		RootPath:     hcont.RootfsDir(),
		RootReadonly: options.RootsfsReadOnly,
	})

	if err != nil {
		return
	}

	// 在磁盘创建容器 bundle
	if err = rs.cstore.CreateContainerBundle(cont.ID(), spec, options.RootfsPath); err != nil {
		return
	}
	// 乐观的修改容器状态
	if err = rs.optimisticChangeContainerStatus(cont, container.Created); err != nil {
		return
	}

	_, err = rs.runtime.CreateContainer(
		cont.ID(),
		hcont.BundleDir(),
		cont.LogPath(),
		rs.containerExitFile(cont.ID()),
		rs.containerAttachFile(cont.ID()),
		options.Stdin,
		options.StdinOnce,
		10*time.Second,
	)

	if err != nil {
		return
	}
	err = cont.SetCreatedAt(time.Now())
	return
}

func (rs *runtimeService) StartContainer(id container.ID) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	cont := rs.cmap.Get(id)
	if cont == nil {
		return errors.New("container not found")
	}
	// 检查容器状态是否为 created
	if err := assertStatus(cont.Status(), container.Created); err != nil {
		return err
	}

	// 乐观的修改容器状态为 Running
	if err := rs.optimisticChangeContainerStatus(cont, container.Running); err != nil {
		return err
	}
	// 调用 runc start container
	if err := rs.runtime.StartContainer(cont.ID()); err != nil {
		return err
	}
	// 等待容器运行成功
	if err := rs.waitContainerStartedNoLock(id); err != nil {
		return nil
	}
	return cont.SetStartedAt(time.Now())

}

func (rs *runtimeService) StopContainer(id container.ID, timeout time.Duration) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	cont := rs.cmap.Get(id)
	if cont == nil {
		return errors.New("container not found")
	}
	// 检查容器状态
	if err := assertStatus(cont.Status(), container.Created, container.Running); err != nil {
		return err
	}
	// todo 实现一个合适的算法,等待超时
	// 如果容器 proc 存在, rs.runtime.KillContainer(cont.ID(),syscall.SIGKILL) 等待一些默认超时时间
	// 如果容器 proc 任然存在,os.kill(PID)

	// todo 测试这个逻辑

	// 乐观的修改容器状态为 Stopped
	if err := rs.optimisticChangeContainerStatus(cont, container.Stopped); err != nil {
		return err
	}
	// 先发送 -15 信号
	if err := rs.runtime.KillContainer(cont.ID(), syscall.SIGTERM); err != nil {
		return err
	}
	// 等待 -15 信号删除情况
	if err := rs.waitContainerStopedNoLock(cont.ID()); err != nil {
		// 15 失败,在用 -9 强杀
		if err := rs.runtime.KillContainer(cont.ID(), syscall.SIGKILL); err != nil {
			return err
		}
		// 再次等待
		if err := rs.waitContainerStopedNoLock(cont.ID()); err != nil {
			return err
		}
	}
	return nil

}

func (rs *runtimeService) RemoveContainer(id container.ID) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	cont := rs.cmap.Get(id)
	if cont == nil {
		return errors.New("container not found")
	}
	// 在磁盘上删除容器状态文件state.json
	if err := rs.cstore.ContainerStateDeleteAtomic(id); err != nil {
		return err
	}
	// runc 开始 remove
	if err := rs.runtime.DeleteContainer(cont.ID()); err != nil {
		return err
	}
	// cleanup
	rs.cmap.Del(id)
	return rs.cstore.DeleteContainer(id)
}

func (rs *runtimeService) ListContainers() ([]*container.Container, error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	var cs []*container.Container
	for _, c := range rs.cmap.All() {
		cont, err := rs.getContainerNoLock(c.ID())
		if err != nil {
			return nil, err
		}
		cs = append(cs, cont)
	}

	// 按照 createat 时间倒排序
	sort.SliceStable(cs, func(i, j int) bool {
		iat := cs[i].CreatedAtNano()
		jat := cs[j].CreatedAtNano()
		if iat == jat {
			return cs[i].ID() < cs[j].ID()
		}
		return iat < jat
	})
	return cs, nil
}

func (rs *runtimeService) GetContainer(id container.ID) (*container.Container, error) {
	rs.lock.Lock()
	defer rs.lock.Unlock()
	return rs.getContainerNoLock(id)
}

// getContainerNoLock 无锁获取容器
func (rs *runtimeService) getContainerNoLock(id container.ID) (*container.Container, error) {
	cont := rs.cmap.Get(id)
	if cont == nil {
		return nil, errors.New("container not found")
	}
	// 获取容器state
	state, err := rs.runtime.ContainerState(cont.ID())
	if err != nil {
		return nil, err
	}
	// 设置容器 status
	status, err := container.StatusFromString(state.Status)
	if err != nil {
		return nil, err
	}
	cont.SetStatus(status)
	// 设置容器 exit code
	if cont.Status() == container.Stopped {
		ts, err := rs.parseContainerExitFile(id)
		if err != nil {
			return nil, err
		}
		cont.SetFinishedAt(ts.At())

		if ts.IsSignaled() {
			cont.SetExitCode(127 + ts.Signal())
		} else {
			cont.SetExitCode(ts.ExitCode())
		}
	}
	blob, err := cont.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := rs.cstore.ContainerStateWriteAtomic(id, blob); err != nil {
		return nil, err
	}
	return cont, nil

}

// restore 同步一下 store 容器
func (rs *runtimeService) restore() error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	hconts, err := rs.cstore.FindContainers()
	if err != nil {
		return err
	}

	// purgeBrokenContainer 清理容器函数
	purgeBrokenContainer := func(id container.ID) {
		// 第一步清理缓存
		rs.cmap.Del(id)
		// 第二步情况磁盘
		if err := rs.cstore.DeleteContainer(id); err != nil {
			klog.Errorf("failed to purge broker container with err:%v", err)
		}
	}

	for _, h := range hconts {
		blob, err := rs.cstore.ContainerStateRead(h.ContainerID())
		if err != nil {
			klog.Warningf("failed to read container state with err:%v", err)
			purgeBrokenContainer(h.ContainerID())
			continue
		}

		cont := &container.Container{}
		if err := cont.UnmarshalJSON(blob); err != nil {
			klog.Warningf("failed to unmarshal container state with err:%v", err)
			continue
		}
		if err := rs.cmap.Add(cont, nil); err != nil {
			klog.Warningf("failed to in-memory store container with err:%v", err)
			continue
		}
		cont, err = rs.getContainerNoLock(h.ContainerID())
		if err != nil {
			klog.Warningf("failed to update container state")
			purgeBrokenContainer(h.ContainerID())
			continue
		}

	}
	return nil

}

// waitContainerStartedNoLock 等待容器启动情况
func (rs *runtimeService) waitContainerStartedNoLock(id container.ID) error {
	// 简单的退避算法
	delays := []time.Duration{
		250 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
		500 * time.Millisecond,
	}
	status := container.Unknown
	for _, d := range delays {
		time.Sleep(d)
		cont, err := rs.getContainerNoLock(id)
		if err != nil {
			return err
		}
		status = cont.Status()
		if status == container.Running {
			return nil
		}
		if status != container.Created {
			break
		}
	}
	//todo 处理容器快速退出的情况
	return errors.New(fmt.Sprintf("Failed to start container; status=%v.", status))
}

// waitContainerStopedNoLock 等待容器删除情况
func (rs *runtimeService) waitContainerStopedNoLock(id container.ID) error {
	// 简单的退避算法
	delays := []time.Duration{
		250 * time.Millisecond,
		250 * time.Millisecond,
	}
	status := container.Unknown
	for _, d := range delays {
		time.Sleep(d)
		cont, err := rs.getContainerNoLock(id)
		if err != nil {
			return err
		}
		status = cont.Status()
		if status == container.Stopped {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Cannot kill container status=%v.", status))
}

// optimisticChangeContainerStatus 乐观的修改容器 status
func (rs *runtimeService) optimisticChangeContainerStatus(c *container.Container, s container.Status) error {
	c.SetStatus(s)
	blob, err := c.MarshalJSON()
	if err != nil {
		return err
	}
	return rs.cstore.ContainerStateWriteAtomic(c.ID(), blob)
}

func (rs *runtimeService) containerAttachFile(id container.ID) string {
	return path.Join(rs.attachDir, string(id))
}

func (rs *runtimeService) containerLogFile(id container.ID) string {
	return path.Join(rs.logDir, string(id)+".log")
}

func (rs *runtimeService) containerExitFile(id container.ID) string {
	return path.Join(rs.exitDir, string(id))
}

func (rs *runtimeService) parseContainerExitFile(id container.ID) (*shimutil.TerminationStatus, error) {
	bytes, err := ioutil.ReadFile(rs.containerExitFile(id))
	if err != nil {
		return nil, errors.New("container exit file parsing failed")
	}
	return shimutil.ParseExitFile(bytes)
}

// assertStatus 判断容器状态是否符合预期
func assertStatus(actual container.Status, expected ...container.Status) error {
	for _, e := range expected {
		if actual == e {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Wrong container status \"%v\". Expected one of=%v", actual, expected))
}
