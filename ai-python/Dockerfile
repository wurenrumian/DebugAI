# ============================================================
# 多阶段构建说明:
# - 第一阶段(builder): 安装依赖并构建
# - 第二阶段(runtime): 运行应用
# - 最终镜像约 100-150MB
# ============================================================
# Python 3.11 slim 版本安全且轻量
# -------------------- 第一阶段: 构建 --------------------
FROM python:3.11-slim-bookworm AS builder

RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources \
	&& sed -i 's/security.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources

WORKDIR /app

# 安装系统依赖（仅构建需要）
RUN apt-get update && apt-get install -y --no-install-recommends \
	gcc \
	&& rm -rf /var/lib/apt/lists/*

# 复制依赖文件
COPY requirements.txt .

# 安装依赖到虚拟环境
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir -r requirements.txt

# -------------------- 第二阶段: 运行 --------------------
FROM python:3.11-slim-bookworm

# 设置工作目录
WORKDIR /app

# 安装运行时依赖（仅 HTTPS 证书）
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

# 从构建阶段复制虚拟环境
COPY --from=builder /opt/venv /opt/venv

# 激活虚拟环境
ENV PATH="/opt/venv/bin:$PATH"

# 复制应用源码
COPY . .

# 创建非 root 用户（安全最佳实践）
RUN addgroup --system --gid 1001 appgroup && \
	adduser --system --uid 1001 --gid 1001 appuser
USER appuser

# 运行参数（可在 docker run / docker-compose 中覆盖）
# 注意：LLM 并发上限 LLM_MAX_CONCURRENCY 按 worker 进程生效，总并发 ≈ UVICORN_WORKERS × LLM_MAX_CONCURRENCY
ENV AI_PORT=8000 \
	UVICORN_WORKERS=2 \
	PYTHONUNBUFFERED=1 \
	PYTHONDONTWRITEBYTECODE=1

# 健康检查（/health 无需鉴权）
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD python -c "import os, urllib.request; urllib.request.urlopen('http://localhost:%s/health' % os.environ.get('AI_PORT', '8000'))"

# 暴露端口（与 AI_PORT 默认值一致）
EXPOSE 8000

# 启动命令：使用 exec 让 uvicorn 成为 PID 1，正确接收停止信号
CMD ["sh", "-c", "exec uvicorn main:app --host 0.0.0.0 --port \"${AI_PORT}\" --workers \"${UVICORN_WORKERS}\" --no-server-header"]
