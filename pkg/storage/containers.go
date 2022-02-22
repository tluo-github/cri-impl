package storage

import (
	"github.com/pkg/errors"
	"github.com/tluo-github/cri-impl/pkg/container"
	"github.com/tluo-github/cri-impl/pkg/fsutil"
	"github.com/tluo-github/cri-impl/pkg/oci"
	"github.com/tluo-github/cri-impl/pkg/rollback"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"path"
)

const (
	DirAccessFailed string = "can't access container directory"
)

type ContainerStore interface {
	RootDir() string
	// CreateContainer 在非易失性的位置创建容器目录(它也可能在里面存储一些容器的元数据)
	CreateContainer(id container.ID, rollback *rollback.Rollback) (*ContainerHandler, error)

	CreateContainerBundle(id container.ID, spec oci.RuntimeSpec, rootfs string) error

	GetContainer(id container.ID) (*ContainerHandler, error)

	// DeleteContainer Removes <container_dir>
	DeleteContainer(id container.ID) error

	FindContainers() ([]*ContainerHandler, error)

	ContainerStateRead(id container.ID) (state []byte, err error)

	// ContainerStateWriteAtomic 更新磁盘上容器的状态(原子性的,使用 os.Rename)
	// 容器state 存储在 <container_dir>/state.json
	ContainerStateWriteAtomic(id container.ID, state []byte) error

	// ContainerStateDeleteAtomic 删除 <container_dir>/state.json, 将容器标记为准备好清理
	ContainerStateDeleteAtomic(id container.ID) error
}

func NewContainerStore(rootDir string) ContainerStore {
	return &containerStore{rootDir: rootDir}
}

type containerStore struct {
	rootDir string
}

func (s *containerStore) RootDir() string {
	return s.rootDir
}

func (s *containerStore) CreateContainer(id container.ID, rollback *rollback.Rollback) (*ContainerHandler, error) {
	if rollback != nil {
		rollback.Add(func() {
			s.DeleteContainer(id)
		})
	}

	dir := s.containerDir(id)
	if ok, err := fsutil.Exists(dir); ok || err != nil {
		if ok {
			return nil, errors.New("container directory already exists")
		}
		return nil, errors.New(DirAccessFailed)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, errors.New("can't create container directory")
	}
	return newContainerHandler(id, dir), nil

}

func (s *containerStore) CreateContainerBundle(
	id container.ID,
	spec oci.RuntimeSpec,
	rootfs string,
) error {
	h, err := s.GetContainer(id)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(h.BundleDir(), 0700); err != nil {
		return errors.Wrap(err, "can't create bundle directory")
	}
	if err := fsutil.CopyDir(rootfs, h.RootfsDir()); err != nil {
		return errors.Wrap(err, "can't copy rootfs directory")
	}
	if err := ioutil.WriteFile(h.RuntimeSpecFile(), spec, 0644); err != nil {
		return errors.Wrap(err, "can't write OCI runtime spec file")
	}
	return nil
}

func (s *containerStore) GetContainer(id container.ID) (*ContainerHandler, error) {
	dir := s.containerDir(id)
	ok, err := fsutil.Exists(dir)
	if err != nil {
		return nil, errors.Wrap(err, DirAccessFailed)
	}
	if ok {
		return newContainerHandler(id, dir), nil
	}
	return nil, nil
}

func (s *containerStore) DeleteContainer(id container.ID) error {
	err := os.RemoveAll(s.containerDir(id))
	if err != nil {
		return errors.Wrap(err, "can't remove container directory")
	}
	return nil
}

func (s *containerStore) FindContainers() ([]*ContainerHandler, error) {
	files, err := ioutil.ReadDir(s.containersDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var hconts []*ContainerHandler
	for _, f := range files {
		if f.IsDir() {
			cid, err := container.ParseId(f.Name())
			if err != nil {
				klog.Warningf("container store: unexpected dir %s with err:%v", f.Name(), err)
				continue
			}
			cdir := path.Join(s.RootDir(), f.Name())
			hconts = append(hconts, newContainerHandler(cid, cdir))
		}
	}
	return hconts, nil

}

func (s *containerStore) ContainerStateRead(id container.ID) (state []byte, err error) {
	h, err := s.GetContainer(id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(h.stateFile())
}

func (s *containerStore) ContainerStateWriteAtomic(id container.ID, state []byte) error {
	h, err := s.GetContainer(id)
	if err != nil {
		return err
	}

	staatefile := h.stateFile()
	tmpfile := staatefile + ".writing"
	if err := ioutil.WriteFile(tmpfile, state, 0600); err != nil {
		return err
	}
	return os.Rename(tmpfile, staatefile)
}

func (s *containerStore) ContainerStateDeleteAtomic(id container.ID) error {
	h, err := s.GetContainer(id)
	if err != nil {
		return err
	}
	return os.Remove(h.stateFile())
}

func (s *containerStore) containersDir() string {
	return path.Join(s.rootDir, "containers")
}

func (s *containerStore) containerDir(id container.ID) string {
	return path.Join(s.containersDir(), string(id))
}

type ContainerHandler struct {
	containerId  container.ID
	containerDir string
}

func newContainerHandler(id container.ID, containerDir string) *ContainerHandler {
	return &ContainerHandler{
		containerId:  id,
		containerDir: containerDir,
	}
}

func (h *ContainerHandler) ContainerID() container.ID {
	return h.containerId
}

func (h *ContainerHandler) ContainerDir() string {
	return h.containerDir
}

func (h *ContainerHandler) BundleDir() string {
	return path.Join(h.ContainerDir(), "bundle")
}

func (h *ContainerHandler) RootfsDir() string {
	return path.Join(h.BundleDir(), "rootfs")
}
func (h *ContainerHandler) RuntimeSpecFile() string {
	return path.Join(h.BundleDir(), "config.json")
}
func (h *ContainerHandler) stateFile() string {
	return path.Join(h.ContainerDir(), "state.json")
}
