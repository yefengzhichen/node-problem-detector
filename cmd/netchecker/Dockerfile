# docker build --platform linux/amd64 -t docker.io/altonzhu/dlv:go1.23.1 -f Dockerfile .
# 基于Golang官方镜像（Debian Bookworm系统）
FROM golang:1.23.1-bullseye

# 切换到 root 用户安装系统工具
USER root

# # 使用阿里云镜像
# RUN sed -i 's|http://deb.debian.org|http://mirrors.aliyun.com/debian|g' /etc/apt/sources.list

# # 添加阿里云 Ceph 镜像（需确认仓库路径）
# RUN echo "deb http://mirrors.aliyun.com/ceph/debian-octopus/ bullseye main" | tee /etc/apt/sources.list.d/ceph.list

# 1. 安装 Python 3.6 和兼容性工具
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    iputils-ping \
    ceph-common \
    vim

RUN go env -w GOPROXY=https://goproxy.cn,direct
# 2. 安装Delve调试工具（Go调试器）
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 3. 配置环境变量
ENV PATH="${PATH}:/go/bin"

# 4. 验证工具安装
RUN dlv version && \
    ping -c 4 127.0.0.1 && \
    ceph --version
# 启动应用程序
CMD ["./sleep 100d"]
