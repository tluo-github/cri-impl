package server

import (
	"errors"
	"github.com/tluo-github/cri-impl/pkg/cri"
	"google.golang.org/grpc"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubelet/cri/streaming"
	"net"
	"os"
	"path/filepath"
)

type Server interface {
	CriServer
	Serve(network, addr string) error
}
type criServer struct {
	runtimeSrv   cri.RuntimeService
	streamingSrv streaming.Server
}

func New(
	runtimeSrv cri.RuntimeService,
	streamingSrv streaming.Server,
) Server {
	return &criServer{
		runtimeSrv:   runtimeSrv,
		streamingSrv: streamingSrv,
	}
}

func (s *criServer) Serve(network, addr string) error {
	lis, err := listen(network, addr)
	if err != nil {
		return err
	}

	gsrv := grpc.NewServer()
	RegisterCriServer(gsrv, s)
	return gsrv.Serve(lis)
}

func listen(network, addr string) (net.Listener, error) {
	if network != "unix" {
		return nil, errors.New("Only UNIX sockets supported")
	}
	if err := os.MkdirAll(filepath.Dir(addr), 0755); err != nil {
		return nil, err
	}
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return net.Listen("unix", addr)
}

func traceRequest(name string, req interface{}) {
	klog.Infof("Request [%s], body:%v", name, req)
}

func traceResponse(name string, resp interface{}, err error) {
	klog.Infof("Response [%s], body:%s with err:%v", name, resp, err)
}
