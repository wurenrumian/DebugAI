#!/bin/sh

# 生成 Nginx 配置文件，替换环境变量
# 使用 envsubst 替换模板中的环境变量占位符
envsubst '${BACKEND_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/conf.d/default.conf

# 启动 Nginx
exec "$@"
