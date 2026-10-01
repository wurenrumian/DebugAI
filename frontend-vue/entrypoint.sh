#!/bin/sh
# DebugAI 前端容器入口：渲染 Nginx 配置后启动 Nginx
#
# 环境变量：
#   BACKEND_HOST   后端服务名（默认 backend）
#   BACKEND_PORT   后端端口（默认 8080）
#   SERVER_NAME    server_name（默认 _ 即匹配任意域名）
#   TLS_CERT_PATH  证书（含中间证书的 fullchain）路径，默认 /etc/nginx/certs/fullchain.pem
#   TLS_KEY_PATH   私钥路径，默认 /etc/nginx/certs/privkey.pem
#   HSTS_MAX_AGE   TLS 模式下 HSTS 的 max-age 秒数（默认 31536000；设为 0 可撤销 HSTS）
#
# 证书和私钥文件都存在且非空 -> TLS 模式：443 提供服务，80 只做 /healthz 与 301 跳转 https，并发送 HSTS
# 否则                         -> HTTP 模式：仅监听 80，并在日志中告警
set -eu

export BACKEND_HOST="${BACKEND_HOST:-backend}"
export BACKEND_PORT="${BACKEND_PORT:-8080}"
export SERVER_NAME="${SERVER_NAME:-_}"
TLS_CERT_PATH="${TLS_CERT_PATH:-/etc/nginx/certs/fullchain.pem}"
TLS_KEY_PATH="${TLS_KEY_PATH:-/etc/nginx/certs/privkey.pem}"
HSTS_MAX_AGE="${HSTS_MAX_AGE:-31536000}"

TEMPLATE=/etc/nginx/nginx.conf.template
SNIPPETS=/etc/nginx/snippets
OUTPUT=/etc/nginx/conf.d/default.conf

case "$HSTS_MAX_AGE" in
	'' | *[!0-9]*)
		echo "[entrypoint] ERROR: HSTS_MAX_AGE 必须是非负整数，当前值: $HSTS_MAX_AGE" >&2
		exit 1
		;;
esac

mkdir -p "$SNIPPETS"

if [ -s "$TLS_CERT_PATH" ] && [ -s "$TLS_KEY_PATH" ]; then
	echo "[entrypoint] 检测到证书 $TLS_CERT_PATH，启用 HTTPS（443）+ HTTP->HTTPS 跳转 + HSTS"

	cat >"$SNIPPETS/listen.conf" <<EOF
listen 443 ssl;
http2 on;

ssl_certificate     $TLS_CERT_PATH;
ssl_certificate_key $TLS_KEY_PATH;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 1d;
ssl_session_tickets off;

add_header Strict-Transport-Security "max-age=$HSTS_MAX_AGE; includeSubDomains" always;
EOF

	# 带引号的 heredoc：$host / $request_uri 保持为 Nginx 变量
	cat >"$SNIPPETS/redirect.conf" <<'EOF'
server {
    listen 80 default_server;
    server_name _;
    server_tokens off;

    location = /healthz {
        access_log off;
        default_type text/plain;
        return 200 "ok\n";
    }

    location / {
        return 301 https://$host$request_uri;
    }
}
EOF
else
	echo "[entrypoint] WARNING: 未找到 TLS 证书（$TLS_CERT_PATH / $TLS_KEY_PATH），仅以 HTTP（80）提供服务。" >&2
	echo "[entrypoint] WARNING: 生产环境请配置 HTTPS，或确保前面有终止 TLS 的反向代理，详见 deploy.md。" >&2

	printf 'listen 80;\n' >"$SNIPPETS/listen.conf"
	printf '# HTTP 模式：无跳转 server\n' >"$SNIPPETS/redirect.conf"
fi

# 只替换列出的变量，Nginx 自身的 $host、$uri 等保持不变
# shellcheck disable=SC2016
envsubst '${BACKEND_HOST} ${BACKEND_PORT} ${SERVER_NAME}' <"$TEMPLATE" >"$OUTPUT"

exec "$@"
