# ============================================================
# DebugAI 前端镜像（多阶段构建）
# - builder : Node 20 + npm ci（依赖锁定在 package-lock.json）构建 Vite 产物
# - runtime : nginx:alpine 提供静态文件，并反向代理 /api、/auth 到 Go 后端
#
# 构建参数：
#   NPM_REGISTRY  npm 源，默认使用 lockfile 中的官方源；国内可用 https://registry.npmmirror.com
#                 （docker-compose.yml 默认传入 npmmirror，可在 .env 中覆盖）
# ============================================================

# -------------------- 构建阶段 --------------------
FROM node:20-alpine AS builder

ARG NPM_REGISTRY=

WORKDIR /app

# 先只复制依赖清单，充分利用缓存层
COPY package.json package-lock.json ./
RUN if [ -n "$NPM_REGISTRY" ]; then npm config set registry "$NPM_REGISTRY"; fi \
	&& npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# -------------------- 运行阶段 --------------------
# nginx:alpine 自带 envsubst（gettext）和 busybox wget（用于健康检查），无需额外安装软件包
FROM nginx:1.28-alpine

# 删除镜像自带的默认站点，配置由 entrypoint.sh 在启动时渲染
RUN rm -f /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/nginx.conf.template
COPY --chmod=755 entrypoint.sh /entrypoint.sh

EXPOSE 80 443

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
	CMD wget -q -O /dev/null http://127.0.0.1/healthz || exit 1

ENTRYPOINT ["/entrypoint.sh"]
CMD ["nginx", "-g", "daemon off;"]
