# -------------------- 构建阶段 --------------------
FROM oven/bun:alpine AS builder

WORKDIR /app

COPY package*.json bun.lockb* ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build

# -------------------- 运行阶段 --------------------
FROM nginx:alpine-slim

# 安装 envsubst（用于环境变量替换）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
	apk add --no-cache gettext

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/nginx.conf.template
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]