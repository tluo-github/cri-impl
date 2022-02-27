# cri-impl
目标: 仿照 ori-o,实现 ori 接口,能给 kubelet 调用

# 版本
* go version go1.17 linux/amd64
* CentOS 7
* Docker 1.13.1

```bash

git clone  https://github.com/tluo-github/cri-impl.git
cd cri-impl
# 导入alpine 镜像 rootfs
make test/data/rootfs_alpine
# 预创建目录
make pre_mkdir
# 构建命令
make linux

# 启动守护进程
./bin/cri-impl-linux


# 创建 containers
sudo bin/crictl-linux container create --image test/data/rootfs_alpine/ cont1 -- sleep 100
sudo bin/crictl-linux container create --image test/data/rootfs_alpine/ cont2 -- sleep 200

# 查询遍历 containers
sudo bin/crictl-linux container list

# 启动 container 
sudo bin/crictl-linux container start <container_id>

# 停止 container 
sudo bin/crictl-linux container stop <container_id>

# 查询 container 状态
sudo bin/crictl-linux container status <container_id>

# 删除 container 
sudo bin/crictl-linux container remove <container_id>

```