# 多阶段构建 Dockerfile
# 第一阶段:构建阶段,使用完整的 Go 环境编译代码
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
# git: 用于下载 Go 依赖
# ca-certificates: HTTPS 证书,用于访问外部 API
RUN apk add --no-cache git ca-certificates

# 复制 go.mod 和 go.sum(如果存在)
# 利用 Docker 缓存层,依赖不变时不重新下载
COPY go.mod go.sum* ./

# 下载依赖
# 如果网络有问题,可以设置 GOPROXY
# 国内推荐使用 goproxy.cn,国际推荐使用官方 proxy.golang.org
ENV GOPROXY=https://goproxy.cn,direct

# 尽早下载依赖,充分利用 Docker 缓存
# 这样当源代码改变时,不需要重新下载所有依赖
RUN go mod download && go mod verify

# 复制源代码
# 这一步放在依赖下载之后,可以缓存依赖层
COPY . .

# 编译应用
# CGO_ENABLED=0: 禁用 CGO,生成纯静态二进制文件,便于在任何 Linux 环境运行
# GOOS=linux: 目标系统 Linux
# GOARCH=amd64: 目标架构 amd64(x86_64)
# -ldflags="-s -w": 去除调试信息和符号表,减小二进制文件大小
# -trimpath: 从构建路径中移除路径前缀,有利于可重复构建
# -o: 输出文件路径
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /app/dailyhot-api-go \
    ./cmd/api

# 第二阶段:运行阶段,使用最小镜像运行编译好的二进制文件
FROM alpine:latest

# 安装 ca-certificates(访问 HTTPS 需要)
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为上海(东八区)
ENV TZ=Asia/Shanghai

# 创建非 root 用户运行应用(安全最佳实践)
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/dailyhot-api-go .

# 复制配置文件(可选)
COPY --from=builder /app/config.yaml .

# 创建日志目录
RUN mkdir -p /app/logs && chown -R app:app /app

# 切换到非 root 用户
USER app

# 暴露端口
EXPOSE 6688

# 健康检查
# 每 30 秒检查一次,超时 3 秒,重试 3 次
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:6688/health || exit 1

# 启动应用
CMD ["./dailyhot-api-go"]
