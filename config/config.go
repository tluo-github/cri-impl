package config

const (
	DefaultListen           = "/var/run/cri-impl.sock"
	DefaultLibRoot          = "/var/lib/cri-impl"
	DefaultRunRoot          = "/var/run/cri-impl"
	DefaultContainerLogRoot = "/var/log/cri-impl/containers"
	DefaultStreaminAddr     = "127.0.0.1:8881"
	DefaultShimmyPath       = "/usr/local/bin/shimmy"
	DefaultRuntimePath      = "/usr/bin/runc"
	DefaultRuntimeRoot      = "/var/run/cri-impl-runc"
)

type Config struct {
	Listen string
	// LibRoot 用于存储长期存在的数据
	LibRoot string
}
