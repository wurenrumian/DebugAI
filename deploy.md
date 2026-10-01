# DebugAI 部署指南

本文档说明如何用 Docker Compose 在一台 Linux 服务器（如腾讯云 CVM / 轻量应用服务器）上部署 DebugAI，
以及 HTTPS、备份恢复、升级、日志和故障排查。

## 目录

- [1. 架构与端口](#1-架构与端口)
- [2. 前置条件](#2-前置条件)
- [3. 配置 .env](#3-配置-env)
- [4. 启动与验证](#4-启动与验证)
- [5. HTTPS](#5-https)
- [6. 客户端真实 IP](#6-客户端真实-ip)
- [7. 备份与恢复](#7-备份与恢复)
- [8. 升级与回滚](#8-升级与回滚)
- [9. 日志与监控](#9-日志与监控)
- [10. 资源限制](#10-资源限制)
- [11. 本地开发：暴露内部端口](#11-本地开发暴露内部端口)
- [12. 故障排查](#12-故障排查)
- [13. 上线检查清单](#13-上线检查清单)

---

## 1. 架构与端口

```
                 宿主机 80/443
                      │
              ┌───────▼────────┐
  浏览器 ───► │ frontend (Nginx)│  静态文件 + 反向代理 /api、/auth
              └───────┬────────┘
                      │ debugai-network
              ┌───────▼────────┐    X-Internal-Token    ┌──────────────────┐
              │ backend (Go)   │ ─────────────────────► │ ai-service (Py)  │ ──► DeepSeek API
              │ :8080 /health  │                        │ :8000 /health    │
              └───────┬────────┘                        └──────────────────┘
                      │ debugai-data（internal，无外网）
           ┌──────────┴──────────┐
     ┌─────▼─────┐         ┌─────▼─────┐
     │ postgres  │         │  redis    │
     │ :5432     │         │  :6379    │
     └───────────┘         └───────────┘
```

| 服务         | 镜像 / 构建              | 容器端口 | 宿主机端口                 | 健康检查                     |
| ------------ | ------------------------ | -------- | -------------------------- | ---------------------------- |
| `frontend`   | `frontend-vue/Dockerfile` | 80 / 443 | `FRONTEND_PORT` / `FRONTEND_HTTPS_PORT` | `GET /healthz`（Nginx 自身） |
| `backend`    | `backend-go/Dockerfile`  | 8080     | **不发布**                 | `GET /health`（DB 异常返回 503） |
| `ai-service` | `ai-python/Dockerfile`   | 8000     | **不发布**                 | `GET /health`                |
| `postgres`   | `postgres:16-alpine`     | 5432     | **不发布**                 | `pg_isready`                 |
| `redis`      | `redis:7-alpine`         | 6379     | **不发布**                 | `redis-cli ping`（带密码）   |

要点：

- **只有 frontend 对外发布端口**。数据库、Redis、后端、AI 服务都只能在 Docker 内部网络访问；
  postgres/redis 所在的 `debugai-data` 是 `internal` 网络，连外网也访问不了。
- 启动顺序由健康检查保证：postgres、redis、ai-service 健康 → backend 启动并健康 → frontend 启动。
- backend 调用 ai-service 时携带共享密钥 `AI_SERVICE_TOKEN`（请求头 `X-Internal-Token`），AI 服务在 production 下没有该密钥会拒绝启动。
- `/api/**/stream` 是 NDJSON 流式接口，Nginx 已对 `/api/` 关闭缓冲（`proxy_buffering off`）并把读超时设为 600 秒。

## 2. 前置条件

- Linux 服务器，推荐 **2 核 4G** 及以上（2G 内存需要加 swap，并调低资源限制，见第 10 节）
- Docker Engine 24+ 与 Docker Compose v2（`docker compose version` 能输出版本）

  ```bash
  # Ubuntu / Debian 示例（国内可用阿里云镜像脚本）
  curl -fsSL https://get.docker.com | sh -s -- --mirror Aliyun
  sudo usermod -aG docker $USER   # 重新登录后生效
  ```

- 安全组 / 防火墙只放行 **22、80、443**（不要放行 5432、6379、8000、8080）
- 已解析到服务器的域名（启用 HTTPS 需要）
- DeepSeek API Key：在 <https://platform.deepseek.com/api_keys> 创建，并确认账户有余额
- 若启用邮箱验证：一个可用的 SMTP 账号（如腾讯企业邮、QQ 邮箱授权码）

获取代码：

```bash
sudo mkdir -p /opt/DebugAI && sudo chown $USER /opt/DebugAI
git clone <仓库地址> /opt/DebugAI
cd /opt/DebugAI
```

## 3. 配置 .env

```bash
cp .env.production.example .env
chmod 600 .env
```

### 3.1 生成密钥

所有密钥都用随机值，**建议只使用十六进制字符**（数据库密码会拼进 DSN，特殊字符容易出问题）：

```bash
echo "JWT_SECRET=$(openssl rand -hex 32)"         # 64 字符，production 下后端要求 ≥ 64
echo "AI_SERVICE_TOKEN=$(openssl rand -hex 32)"
echo "POSTGRES_PASSWORD=$(openssl rand -hex 24)"
echo "REDIS_PASSWORD=$(openssl rand -hex 24)"
```

把输出填入 `.env` 对应位置（替换 `<...>` 占位符）。

### 3.2 必需变量

| 变量                | 说明                                                       |
| ------------------- | ---------------------------------------------------------- |
| `POSTGRES_PASSWORD` | 数据库密码。**仅在数据库首次初始化时生效**，之后修改需进 psql 执行 `ALTER USER` |
| `REDIS_PASSWORD`    | Redis `requirepass`，后端用同一个值连接                    |
| `JWT_SECRET`        | JWT 签名密钥，production 下至少 64 字符                    |
| `AI_SERVICE_TOKEN`  | backend ↔ ai-service 内部调用共享密钥                      |
| `DEEPSEEK_API_KEY`  | DeepSeek API 密钥                                          |

缺少任意一个，`docker compose` 会直接报错，例如：

```
required variable JWT_SECRET is missing a value: 请在 .env 中设置 JWT_SECRET（...）
```

### 3.3 其他常用变量

| 变量                       | 默认值        | 说明 |
| -------------------------- | ------------- | ---- |
| `ENV`                      | `production`  | compose 中默认 production |
| `SERVER_NAME`              | `_`           | Nginx `server_name`，填你的域名 |
| `FRONTEND_URL`             | 空            | 邮件中验证链接的前缀，如 `https://debugai.example.com` |
| `CORS_ALLOW_ORIGINS`       | 代码默认      | 同域部署填站点地址即可 |
| `SKIP_EMAIL_VERIFICATION`  | `false`       | 生产必须为 false |
| `SMTP_*`                   | 空            | 邮箱验证所需 |
| `DB_SSLMODE`               | `disable`     | 自带 postgres 不启用 SSL 且仅内网可达；换云数据库时改为 `require`/`verify-full` |
| `UVICORN_WORKERS`          | 代码默认      | AI 服务进程数，每个约 100MB+ 内存 |
| `LLM_MODEL` / `LLM_BASE_URL` / `LLM_TIMEOUT` / `LLM_MAX_RETRIES` / `LLM_TOTAL_TIMEOUT` / `LLM_MAX_CONCURRENCY` | 代码默认 | LLM 调用参数 |
| `AI_STREAM_MAX_CONCURRENT` / `AI_STREAM_TIMEOUT` | 代码默认 | 后端流式接口并发与超时（超时不要超过 Nginx 的 600s） |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME_MINUTES` | 代码默认 | 连接池；`max_connections=50`，保持明显小于它 |
| `MAX_BODY_BYTES`           | 代码默认      | 后端请求体上限（Nginx 另有 2MB 上限） |
| `TRUSTED_PROXIES`          | 私有网段      | 后端信任的代理网段，一般无需修改 |
| `ALPINE_MIRROR` / `GOPROXY` / `NPM_REGISTRY` | 国内镜像 | 构建时的软件源，海外服务器可设为空 |

可选变量注释掉即可，docker-compose.yml 只在设置了时才传给容器，未设置时使用代码内默认值。
完整列表和注释见 [`.env.production.example`](.env.production.example)。

## 4. 启动与验证

```bash
docker compose up -d --build
docker compose ps            # 所有服务 STATUS 应为 healthy
```

首次构建需要拉取镜像和依赖，国内服务器默认已使用阿里云 apk、goproxy.cn、npmmirror 镜像源。

验证：

```bash
curl -i http://localhost/healthz                 # Nginx：ok
docker compose exec backend wget -qO- http://127.0.0.1:8080/health   # 后端（含数据库检查）
docker compose exec ai-service python -c "import urllib.request;print(urllib.request.urlopen('http://127.0.0.1:8000/health').read())"
curl -sI http://localhost/ | grep -i content-security-policy          # 安全响应头
```

然后浏览器访问 `http://<服务器IP或域名>/` 注册账号。

> 启动日志中看到 `[entrypoint] WARNING: 未找到 TLS 证书` 表示当前只有 HTTP，生产环境请按第 5 节配置 HTTPS。

## 5. HTTPS

三种方式任选其一。

### 方案 A：容器内 Nginx 直接终止 TLS（推荐，最简单）

frontend 容器启动时检查证书：

- `TLS_CERT_PATH`（默认 `/etc/nginx/certs/fullchain.pem`）和 `TLS_KEY_PATH`（默认 `/etc/nginx/certs/privkey.pem`）**都存在且非空** →
  监听 443（TLS 1.2/1.3 + HTTP/2），80 端口只保留 `/healthz` 并 301 跳转到 https，所有响应附带 HSTS；
- 否则 → 只监听 80，并在日志中告警。

宿主机的 `./certs` 目录（可用 `TLS_CERT_DIR` 修改）以只读方式挂载到容器的 `/etc/nginx/certs`。

**1）用 acme.sh 申请 Let's Encrypt 证书（DNS 验证，以 DNSPod 为例）：**

```bash
curl https://get.acme.sh | sh -s email=you@example.com
export DP_Id="<DNSPod ID>" DP_Key="<DNSPod Token>"
~/.acme.sh/acme.sh --issue --dns dns_dp -d debugai.example.com --server letsencrypt

mkdir -p /opt/DebugAI/certs
~/.acme.sh/acme.sh --install-cert -d debugai.example.com \
  --fullchain-file /opt/DebugAI/certs/fullchain.pem \
  --key-file       /opt/DebugAI/certs/privkey.pem \
  --reloadcmd      "cd /opt/DebugAI && docker compose exec -T frontend nginx -s reload"
chmod 644 /opt/DebugAI/certs/*.pem   # 容器内 Nginx master 以 root 读取，600 也可以
```

acme.sh 会自动续期并执行 `--reloadcmd`。也可以使用腾讯云控制台申请的免费证书（下载 Nginx 格式，
把 `xxx_bundle.crt` 改名为 `fullchain.pem`、`xxx.key` 改名为 `privkey.pem` 放到 `certs/`）。

**2）设置域名并重建 frontend：**

```bash
# .env
SERVER_NAME=debugai.example.com
FRONTEND_URL=https://debugai.example.com
CORS_ALLOW_ORIGINS=https://debugai.example.com

docker compose up -d frontend
docker compose logs frontend | grep entrypoint   # 应看到「检测到证书 ... 启用 HTTPS」
curl -I https://debugai.example.com/              # 应含 strict-transport-security
```

> HSTS 一旦下发，浏览器在 max-age 内只会用 HTTPS 访问该域名。首次上线可先设置 `HSTS_MAX_AGE=300`，
> 确认 HTTPS 稳定后再删掉该行（默认一年）。HSTS 包含 `includeSubDomains`，请确认所有子域名都支持 HTTPS。
>
> 如果 `FRONTEND_HTTPS_PORT` 不是 443，80→443 的跳转地址不带端口，需要自行调整模板。

### 方案 B：宿主机上的 Caddy / Nginx 终止 TLS

适用于服务器上已有其他站点共用 80/443 的情况。让 frontend 只监听本机端口：

```bash
# .env
FRONTEND_PORT=127.0.0.1:8081
FRONTEND_HTTPS_PORT=127.0.0.1:8443   # 不使用，但避免占用宿主机 443
```

Caddyfile 示例（Caddy 会自动申请并续期证书）：

```
debugai.example.com {
    encode gzip
    reverse_proxy 127.0.0.1:8081 {
        # 流式接口：立即刷新，不缓冲
        flush_interval -1
        transport http {
            read_timeout 600s
        }
    }
    header Strict-Transport-Security "max-age=31536000; includeSubDomains"
}
```

此时 `certs/` 留空即可（容器内保持 HTTP 模式，日志中的 TLS 告警可以忽略）。
**注意：必须按第 6 节调整真实 IP 配置**，否则后端看到的所有客户端 IP 都是 Docker 网关地址，按 IP 限流会误伤所有用户。

### 方案 C：云负载均衡 / CDN 终止 TLS

腾讯云 CLB、CDN 等在前面终止 TLS 时，与方案 B 相同：后端只接收来自 CLB 的流量，
并且**必须**按第 6 节配置真实 IP；CDN 还需要关闭对 `/api/` 的缓存和响应缓冲，否则流式输出会被攒成一整块。

## 6. 客户端真实 IP

后端使用 Gin 的 `c.ClientIP()` 做限流和审计，只信任 `TRUSTED_PROXIES`（默认私有网段，即 Docker 网络中的 Nginx）
传来的 `X-Forwarded-For`。为防止客户端伪造 IP，`frontend-vue/nginx.conf.template` 中：

```nginx
proxy_set_header X-Real-IP       $remote_addr;
proxy_set_header X-Forwarded-For $remote_addr;   # 覆盖，而不是 $proxy_add_x_forwarded_for 追加
```

也就是说后端看到的 IP 永远是「直接连到 Nginx 的那个地址」。默认部署（浏览器直连 Nginx）下这就是真实客户端 IP。

**如果 Nginx 前面还有一层代理（方案 B / C）**，`$remote_addr` 会变成那层代理的地址。需要在
`nginx.conf.template` 的 `server { ... }` 内加入 realip 配置，让 Nginx 先从可信代理的请求头中还原真实 IP：

```nginx
# 只信任前面那层代理的地址，切勿写 0.0.0.0/0
set_real_ip_from 172.16.0.0/12;     # 方案 B：宿主机代理经 Docker 网关进入（前提：端口只绑定 127.0.0.1）
# set_real_ip_from 10.0.0.0/8;      # 方案 C：替换为 CLB / CDN 回源网段
real_ip_header    X-Forwarded-For;
real_ip_recursive on;
```

还原后 `$remote_addr` 就是真实客户端 IP，上面的覆盖写法仍然正确。若前置代理终止了 TLS，
还可以把 `X-Forwarded-Proto $scheme` 改为 `$http_x_forwarded_proto`（同样只在可信代理后面这样做）。
修改模板后执行 `docker compose up -d --build frontend`。

## 7. 备份与恢复

### 7.1 数据库备份

```bash
./scripts/backup.sh
```

- 通过 `docker compose exec -T postgres pg_dump` 导出整个库（纯 SQL，含 `DROP ... IF EXISTS`），gzip 压缩
- 保存为 `backups/debugai_<库名>_<时间戳>.sql.gz`（权限 600，`backups/` 已在 .gitignore 中）
- 校验压缩包完整性后才落盘，并删除超过保留天数的旧备份

可在 `.env` 或环境变量中配置：

| 变量                    | 默认          | 说明                       |
| ----------------------- | ------------- | -------------------------- |
| `BACKUP_DIR`            | `./backups`   | 备份目录（相对项目根目录） |
| `BACKUP_RETENTION_DAYS` | `14`          | 保留天数，`0` 表示不清理   |

### 7.2 定时备份（cron）

```bash
crontab -e
# 每天 03:00 备份
0 3 * * * /opt/DebugAI/scripts/backup.sh >> /var/log/debugai-backup.log 2>&1
```

运行 cron 的用户需要能执行 `docker`（在 docker 组中或使用 root）。

**备份一定要复制到服务器之外**，例如腾讯云 COS（`coscli`）：

```bash
30 3 * * * coscli sync /opt/DebugAI/backups/ cos://<bucket>/debugai-backups/ -r >> /var/log/debugai-backup.log 2>&1
```

Redis 中只有缓存、限流计数和 JWT 黑名单，开启了 AOF 持久化（`redis_data` 卷），一般不需要单独备份。

### 7.3 恢复

```bash
./scripts/backup.sh                                   # 先备份当前状态
./scripts/restore.sh backups/debugai_debugai_20250101_030000.sql.gz
```

脚本会：校验备份 → 交互确认（`--yes` 跳过）→ 停止 frontend/backend → 在单个事务中执行 SQL（出错整体回滚）→ 重新启动 backend/frontend。

注意：恢复会删除并重建**备份中包含的**表；备份之后新建的、备份中不存在的表会保留。
如需完全干净的恢复（例如迁移到新服务器）：

```bash
docker compose up -d postgres
docker compose exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" --if-exists "$POSTGRES_DB" && createdb -U "$POSTGRES_USER" "$POSTGRES_DB"'
./scripts/restore.sh backups/xxx.sql.gz --yes
docker compose up -d
```

建议每隔一段时间在测试机上演练一次恢复，确认备份可用。

## 8. 升级与回滚

```bash
cd /opt/DebugAI
./scripts/backup.sh                     # 1. 升级前先备份
git fetch && git checkout <新版本 tag 或 main> && git pull
diff <(grep -o '^[A-Z_]*=' .env.production.example | sort) <(grep -o '^[A-Z_]*=' .env | sort)   # 2. 检查是否有新增必需变量
docker compose build                    # 3. 先构建，服务不中断
docker compose up -d                    # 4. 只重建有变化的容器
docker compose ps                       # 5. 确认全部 healthy
docker image prune -f                   # 6. 清理旧镜像
```

后端启动时会执行 GORM AutoMigrate（只新增表/列，不删除）。回滚：

```bash
git checkout <旧版本>
docker compose up -d --build
# 若新版本的数据变更与旧版本不兼容，再用升级前的备份执行 ./scripts/restore.sh
```

## 9. 日志与监控

```bash
docker compose logs -f --tail=200 backend      # 某个服务
docker compose logs -f                         # 全部
docker stats                                   # 实时 CPU / 内存
```

- 所有容器使用 json-file 日志驱动，单文件 10MB、保留 5 个，不会撑满磁盘
- Nginx 错误日志级别为 `warn`；访问日志输出到容器标准输出
- PostgreSQL 开启了 `logging_collector`（csvlog），慢查询（>500ms）日志在数据卷内：
  `docker compose exec postgres ls /var/lib/postgresql/data/log`
- 可用外部监控探测 `https://<域名>/healthz`（只表示 Nginx 存活）；后端及数据库状态看 `docker compose ps` 中 backend 是否 healthy

## 10. 资源限制

`docker-compose.yml` 为每个服务设置了内存 / CPU 上限，可在 `.env` 中调整：

| 服务       | 内存变量             | 默认  | CPU 变量         | 默认 |
| ---------- | -------------------- | ----- | ---------------- | ---- |
| postgres   | `POSTGRES_MEM_LIMIT` | 1g    | `POSTGRES_CPUS`  | 1.0  |
| redis      | `REDIS_MEM_LIMIT`    | 512m  | `REDIS_CPUS`     | 0.5  |
| backend    | `BACKEND_MEM_LIMIT`  | 512m  | `BACKEND_CPUS`   | 1.0  |
| ai-service | `AI_MEM_LIMIT`       | 1g    | `AI_CPUS`        | 1.0  |
| frontend   | `FRONTEND_MEM_LIMIT` | 128m  | `FRONTEND_CPUS`  | 0.5  |

`backend-go/config/postgresql.conf` 中 `shared_buffers=384MB`，因此 postgres 至少需要约 1GB；
如果服务器只有 2GB 内存，请同时调低 `shared_buffers`（如 256MB）和各服务上限，并配置 swap。
`CPUS` 不能超过服务器的 CPU 核数，否则容器无法启动。

## 11. 本地开发：暴露内部端口

生产配置不向宿主机发布数据库等端口。本机调试需要 `psql`、`redis-cli` 或直接调用后端时：

```bash
cp docker-compose.override.example.yml docker-compose.override.yml   # 已在 .gitignore 中
docker compose up -d
psql -h 127.0.0.1 -U postgres debugai
```

覆盖文件只绑定 `127.0.0.1`，**不要在生产服务器上使用**。

## 12. 故障排查

| 现象 | 排查 |
| ---- | ---- |
| `required variable XXX is missing a value` | `.env` 缺少必需变量，或不在项目根目录执行命令 |
| backend 一直 `unhealthy` / 不断重启 | `docker compose logs backend`：常见原因是 `JWT_SECRET` 不足 64 字符、数据库密码不对、Redis 密码不对 |
| 修改 `POSTGRES_PASSWORD` 后后端连不上数据库 | 密码只在数据卷首次初始化时生效。进入 `docker compose exec postgres psql -U postgres` 执行 `ALTER USER postgres PASSWORD '<新密码>';` |
| ai-service `unhealthy` | `docker compose logs ai-service`：production 下缺少 `AI_SERVICE_TOKEN` 会拒绝启动；检查 `DEEPSEEK_API_KEY` |
| AI 请求返回 401/403 | backend 与 ai-service 的 `AI_SERVICE_TOKEN` 不一致（都来自同一个 .env，修改后需 `docker compose up -d` 重建两个容器） |
| 页面打开但接口 502 | backend 未就绪：`docker compose ps backend`；Nginx 通过 Docker DNS 动态解析 backend，backend 恢复后无需重启 frontend |
| AI 流式输出一次性出现 / 中途断开 | 前面有 CDN/代理在缓冲或超时；Nginx 内 `/api/` 读超时为 600s，`AI_STREAM_TIMEOUT` 不应超过它 |
| 上传/提交返回 413 | 请求体超过 Nginx `client_max_body_size 2m` 或后端 `MAX_BODY_BYTES` |
| 所有用户同时被限流 | 前面有代理但未配置真实 IP，见第 6 节 |
| 80/443 端口被占用 | `sudo ss -lntp | grep -E ':80|:443'`；修改 `FRONTEND_PORT` 或按方案 B 部署 |
| 容器被 OOM Kill | `docker inspect <容器> | grep OOMKilled`；调高对应的 `*_MEM_LIMIT` 或减少 `UVICORN_WORKERS` |
| 构建时下载依赖很慢/失败 | 检查 `ALPINE_MIRROR` / `GOPROXY` / `NPM_REGISTRY`；海外服务器可改为官方源 |
| 浏览器控制台报 CSP 错误 | 前端引入了外部资源（CDN 脚本、字体等），需要同步修改 `nginx.conf.template` 中的 `Content-Security-Policy` |

## 13. 上线检查清单

- [ ] `.env` 由 `.env.production.example` 生成，所有密钥均为随机值，权限 600，未提交到 git
- [ ] `ENV=production`，`SKIP_EMAIL_VERIFICATION=false`
- [ ] 安全组只放行 22 / 80 / 443
- [ ] HTTPS 已启用（`curl -I https://域名` 含 HSTS），或前置代理已终止 TLS 且配置了真实 IP
- [ ] `docker compose ps` 全部 healthy
- [ ] 定时备份已配置，并已异地保存、演练过一次恢复
- [ ] DeepSeek 账户余额与用量告警已设置
