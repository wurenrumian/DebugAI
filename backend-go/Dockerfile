# ============================================================
# 多阶段构建说明:
# - 第一阶段( builder): 编译 Go 程序
# - 第二阶段( runtime): 运行编译好的二进制
# - 最终镜像约 15-20MB
#
# 使用方法: docker build -t debugai-backend .
# ============================================================
# golang:1.21-alpine提示有高危漏洞，本地测试可用，部署时建议更新镜像
# -------------------- 第一阶段: 构建 --------------------
FROM golang:1.25-alpine AS builder
WORKDIR /app

# 安装 git（go mod download 需要）
RUN apk add --no-cache git

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
# CGO_ENABLED=0 确保静态编译，避免 musl/glibc 兼容问题
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main .

# -------------------- 第二阶段: 运行 --------------------
FROM alpine:3.19

# 安装运行时依赖（HTTPS 证书 + 时区数据）
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /root/
# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 创建非 root 用户（安全最佳实践）
RUN addgroup -g 1001 -S appgroup && \
	adduser -u 1001 -S appuser -G appgroup
USER appuser

EXPOSE 8080
CMD ["./main"]
