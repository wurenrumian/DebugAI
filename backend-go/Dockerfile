# ============================================================
# DebugAI Go 后端镜像（多阶段构建）
# - builder : 编译静态链接的 Go 二进制
# - runtime : 精简 alpine，以非 root 用户运行
#
# 构建参数（均可通过 --build-arg 或 docker-compose.yml 的 build.args 覆盖）：
#   ALPINE_MIRROR  apk 镜像域名，例如 mirrors.aliyun.com；留空则使用官方 dl-cdn.alpinelinux.org
#   GOPROXY        Go 模块代理，例如 https://goproxy.cn,direct（国内）
#
# 使用方法: docker build -t debugai-backend --build-arg ALPINE_MIRROR=mirrors.aliyun.com .
# ============================================================

# -------------------- 第一阶段: 构建 --------------------
FROM golang:1.25-alpine AS builder

ARG ALPINE_MIRROR=
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

RUN if [ -n "$ALPINE_MIRROR" ]; then \
		sed -i "s#dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g" /etc/apk/repositories; \
	fi \
	&& apk add --no-cache git

WORKDIR /src

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
# CGO_ENABLED=0 确保静态编译；不写死 GOARCH，跟随构建平台（amd64 / arm64 均可）
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# -------------------- 第二阶段: 运行 --------------------
FROM alpine:3.22

ARG ALPINE_MIRROR=

# 运行时依赖：HTTPS 根证书 + 时区数据（busybox 自带 wget，用于健康检查）
RUN if [ -n "$ALPINE_MIRROR" ]; then \
		sed -i "s#dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g" /etc/apk/repositories; \
	fi \
	&& apk add --no-cache ca-certificates tzdata \
	&& addgroup -g 1001 -S appgroup \
	&& adduser -u 1001 -S -D -H -G appgroup -s /sbin/nologin appuser

ENV TZ=Asia/Shanghai \
	GIN_MODE=release

# 注意：不要放在 /root 下（权限 700，非 root 用户无法进入）
WORKDIR /app
COPY --from=builder /out/server /app/server

USER appuser:appgroup

EXPOSE 8080

# 端口与 docker-compose.yml 中的 BACKEND_PORT=8080 保持一致
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=3 \
	CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

CMD ["/app/server"]
