package server

import (
	"context"
	"github.com/tluo-github/cri-impl/pkg/container"
	"github.com/tluo-github/cri-impl/pkg/cri"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"time"
)

func (c *criServer) Version(
	ctx context.Context,
	req *VersionRequest,
) (resp *VersionResponse, err error) {
	return &VersionResponse{
		Version:     "0.0.1",
		RuntimeName: "runc",
	}, nil
}

func (c *criServer) CreateContainer(
	ctx context.Context,
	req *CreateContainerRequest,
) (resp *CreateContainerResponse, err error) {
	traceRequest("CreateContainer", req)
	defer func() { traceResponse("CreateContainer", resp, err) }()

	cont, err := c.runtimeSrv.CreateContainer(
		cri.ContainerOptions{
			Name:            req.Name,
			Command:         req.Command,
			Args:            req.Args,
			RootfsPath:      req.RootfsPath,
			RootsfsReadOnly: req.RootfsReadonly,
			Stdin:           false,
			StdinOnce:       false,
		},
	)
	if err == nil {
		resp = &CreateContainerResponse{ContainerId: string(cont.ID())}
	}
	return
}

func (c *criServer) StartContainer(
	ctx context.Context,
	req *StartContainerRequest,
) (resp *StartContainerResponse, err error) {
	traceRequest("StartContainer", req)
	defer func() { traceResponse("StartContainer", resp, err) }()

	err = c.runtimeSrv.StartContainer(
		container.ID(req.ContainerId),
	)
	if err == nil {
		resp = &StartContainerResponse{}
	}
	return
}

func (c *criServer) StopContainer(
	ctx context.Context,
	req *StopContainerRequest,
) (resp *StopContainerResponse, err error) {
	traceRequest("StopContainer", req)
	defer func() { traceResponse("StopContainer", resp, err) }()

	err = c.runtimeSrv.StopContainer(
		container.ID(req.ContainerId),
		time.Duration(req.Timeout)*time.Second,
	)
	if err == nil {
		resp = &StopContainerResponse{}
	}
	return
}

func (c *criServer) RemoveContainer(
	ctx context.Context,
	req *RemoveContainerRequest,
) (resp *RemoveContainerResponse, err error) {
	traceRequest("RemoveContainer", req)
	defer func() { traceResponse("RemoveContainer", resp, err) }()

	err = c.runtimeSrv.RemoveContainer(
		container.ID(req.ContainerId),
	)
	if err == nil {
		resp = &RemoveContainerResponse{}
	}
	return
}

func (c *criServer) ListContainers(
	ctx context.Context,
	req *ListContainersRequest,
) (resp *ListContainersResponse, err error) {
	traceRequest("ListContainers", req)
	defer func() { traceResponse("ListContainers", resp, err) }()

	cs, err := c.runtimeSrv.ListContainers()
	if err != nil {
		return nil, err
	}
	return &ListContainersResponse{
		Containers: toPbContainers(cs),
	}, nil
}

func (c *criServer) ContainerStatus(
	ctx context.Context,
	req *ContainerStatusRequest,
) (resp *ContainerStatusResponse, err error) {
	traceRequest("ContainerStatus", req)
	defer func() { traceResponse("ContainerStatus", resp, err) }()

	cont, err := c.runtimeSrv.GetContainer(
		container.ID(req.ContainerId),
	)
	if err != nil {
		return nil, err
	}

	return &ContainerStatusResponse{
		Status: &ContainerStatus{
			ContainerId:   string(cont.ID()),
			ContainerName: string(cont.Name()),
			State:         toPbContainerState(cont.Status()),
			CreatedAt:     cont.CreatedAtNano(),
			StartedAt:     cont.StartedAtNano(),
			FinishedAt:    cont.FinishedAtNano(),
			ExitCode:      cont.ExitCode(),
			LogPath:       cont.LogPath(),
		},
	}, nil

}

func (c *criServer) Attach(
	ctx context.Context,
	req *AttachRequest,
) (resp *AttachResponse, err error) {
	traceRequest("Attach", req)
	defer func() { traceResponse("Attach", resp, err) }()

	r, err := c.streamingSrv.GetAttach(&criapi.AttachRequest{
		ContainerId: req.ContainerId,
		Stdin:       req.Stdin,
		Tty:         req.Tty,
		Stdout:      req.Stdout,
		Stderr:      req.Stderr,
	})
	if err != nil {
		return nil, err
	}
	return &AttachResponse{
		Url: r.Url,
	}, err
}

func (c *criServer) mustEmbedUnimplementedCriServer() {
	panic("implement me")
}

func toPbContainers(cs []*container.Container) (rv []*Container) {
	for _, c := range cs {
		rv = append(rv, &Container{
			Id:        string(c.ID()),
			Name:      string(c.Name()),
			CreatedAt: c.CreatedAtNano(),
			State:     toPbContainerState(c.Status()),
		})
	}
	return
}

func toPbContainerState(s container.Status) ContainerState {
	switch s {
	case container.Created:
		return ContainerState_CREATED
	case container.Running:
		return ContainerState_RUNNING
	case container.Stopped:
		return ContainerState_EXITED
	}
	return ContainerState_UNKNOW
}
