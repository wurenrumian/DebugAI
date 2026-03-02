# DebugAI 部署指南

## 目录
1. [服务器环境准备](#服务器环境准备)
2. [部署步骤](#部署步骤)
3. [生产环境配置](#生产环境配置)
4. [验证与监控](#验证与监控)
5. [运维任务](#运维任务)
6. [常见问题](#常见问题)
7. [部署前待办清单](#部署前待办清单)

---

## 服务器环境准备

### 系统要求

| 组件     | 最低配置                  | 推荐配置         |
| -------- | ------------------------- | ---------------- |
| CPU      | 2核                       | 4核+             |
| 内存     | 4GB                       | 8GB+             |
| 磁盘     | 20GB                      | 50GB+ (SSD)      |
| 操作系统 | Ubuntu 20.04+ / CentOS 8+ | Ubuntu 22.04 LTS |

### 安装必要软件

#### Ubuntu/Debian
```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 安装 Docker Compose
sudo apt install docker-compose-plugin -y

# 安装 Git
sudo apt install git -y

# 安装 Nginx（用于反向代理）
sudo apt install nginx -y

# 安装 Certbot（SSL 证书）
sudo apt install certbot python3-certbot-nginx -y
```

#### CentOS/RHEL
```bash
# 安装 Docker
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker

# 安装 Git
sudo yum install git -y

# 安装 Nginx
sudo yum install nginx -y

# 安装 Certbot
sudo yum install certbot python3-certbot-nginx -y
```

### 配置 Docker

```bash
# 添加当前用户到 docker 组（避免每次使用 sudo）
sudo usermod -aG docker $USER
# 需要重新登录生效

# 配置 Docker 开机自启
sudo systemctl enable docker

# 验证安装
docker --version
docker compose version
```

---

## 部署步骤

### 上传代码到服务器

```bash
# 方式1：克隆仓库
git clone <your-repo-url> /opt/debugai
cd /opt/debugai

# 方式2：从本地上传
# 使用 scp 或 rsync 上传整个项目目录
```

### 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env  # 如果存在 .env.example
# 或直接编辑 .env 文件

# 编辑 .env 文件，填入真实配置
nano .env
```

**必需配置项**：
```bash
# 数据库
POSTGRES_PASSWORD=your-strong-password-here

# JWT（必须修改！）
JWT_SECRET=$(openssl rand -base64 64)

# DeepSeek API
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 环境
ENV=production
```

### 下载依赖（可选）

Docker Compose 会自动构建镜像并下载依赖。如果需要手动预下载：

```bash
# Go 依赖
cd backend-go
go mod download
cd ..

# Python 依赖
cd ai-python
pip download -r requirements.txt -d ./deps
cd ..
```

### 启动服务

```bash
# 一键启动所有服务
docker compose up -d --build

# 查看启动状态
docker compose ps

# 查看日志
docker compose logs -f
```

### 初始化数据库

```bash
# 进入后端容器
docker compose exec backend /bin/bash

# 运行数据库迁移（如果有）
# 或等待自动初始化（Weak Point Keywords 会在首次启动时自动创建）

# 退出容器
exit
```

---

## 生产环境配置

### Nginx 反向代理配置

创建 `/etc/nginx/sites-available/debugai`：

```nginx
upstream backend {
    server backend:8080;
}

upstream ai {
    server ai-service:8000;
}

upstream frontend {
    server frontend:80;
}

server {
    listen 80;
    server_name your-domain.com;  # 修改为你的域名

    # 前端
    location / {
        proxy_pass http://frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 后端 API
    location /api/ {
        proxy_pass http://backend:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 认证相关
    location /auth/ {
        proxy_pass http://backend:8080/auth/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # AI 服务（如果需要直接访问）
    location /ai/ {
        proxy_pass http://ai:8000/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

启用配置：
```bash
sudo ln -s /etc/nginx/sites-available/debugai /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### HTTPS 配置

```bash
# 申请 SSL 证书（需要域名已解析到服务器）
sudo certbot --nginx -d your-domain.com

# 自动续期测试
sudo certbot renew --dry-run
```

### 修改 CORS 配置

编辑 `backend-go/main.go`，将 CORS 改为生产域名：
```go
r.Use(func(c *gin.Context) {
    c.Writer.Header().Set("Access-Control-Allow-Origin", "https://your-domain.com")
    // ... 其他配置
})
```

重新构建：
```bash
docker compose up -d --build backend
```

### 数据库连接池优化

编辑 `backend-go/config/db.go`：

```go
// 修改连接池参数
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    ConnMaxLifetime: time.Hour,
    MaxOpenConns:    25,  // 生产环境建议 25-50
    MaxIdleConns:    25,  // 建议与 MaxOpenConns 相同
})
```

### 调整 Worker Pool 配置

编辑 `backend-go/models/job.go` 中的 `PoolConfigs()` 函数：

```go
func PoolConfigs() map[string]PoolConfig {
    return map[string]PoolConfig{
        "debug": {
            WorkerCount:  10,  // 根据 CPU 核数调整
            QueueSize:    200,
            Timeout:      60 * time.Second,
        },
        "evaluate": {
            WorkerCount:  5,
            QueueSize:    100,
            Timeout:      30 * time.Second,
        },
        "recommend": {
            WorkerCount:  3,
            QueueSize:    50,
            Timeout:      20 * time.Second,
        },
    }
}
```

---

## 验证与监控

### 服务验证

```bash
# 检查所有容器
docker compose ps

# 健康检查
curl http://localhost/health
curl http://localhost:8000/health

# 测试注册接口
curl -X POST http://localhost/auth/register \
  -H "Content-Type: application/json" \
  -d '{"student_id":"test001","username":"test","password":"123456"}'
```

### 日志查看

```bash
# 实时查看所有日志
docker compose logs -f

# 查看特定服务
docker compose logs -f backend
docker compose logs -f ai-service

# 结构化日志（JSON 格式）
docker compose logs backend | jq .
```

### 资源监控

```bash
# 容器资源使用
docker stats

# 查看数据库连接
docker compose exec postgres psql -U postgres -d debugai -c "SELECT count(*) FROM pg_stat_activity;"
```

---

## 运维任务

### 数据库备份

```bash
# 创建备份脚本 /opt/debugai/backup.sh
#!/bin/bash
BACKUP_DIR="/opt/debugai/backups"
DATE=$(date +%Y%m%d_%H%M%S)
docker compose exec -T postgres pg_dump -U postgres debugai > $BACKUP_DIR/backup_$DATE.sql
gzip $BACKUP_DIR/backup_$DATE.sql
find $BACKUP_DIR -name "*.sql.gz" -mtime +7 -delete
```

添加到 crontab：
```bash
chmod +x /opt/debugai/backup.sh
crontab -e
# 每天凌晨 2 点备份
0 2 * * * /opt/debugai/backup.sh
```

### 日志轮转

创建 `/etc/logrotate.d/docker-debugai`：
```bash
/var/lib/docker/containers/*/*.log {
    rotate 7
    daily
    compress
    copytruncate
    missingok
    notifempty
}
```

### 防火墙配置

```bash
# Ubuntu (ufw)
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# CentOS (firewalld)
sudo firewall-cmd --permanent --add-port=22/tcp
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
```

### 更新代码

```bash
cd /opt/debugai
git pull origin main
docker compose up -d --build
docker compose logs -f  # 检查是否有错误
```

---

## 常见问题

| 问题              | 解决方案                                                |
| ----------------- | ------------------------------------------------------- |
| 端口 80 被占用    | `sudo systemctl stop nginx`（如果已有其他服务）         |
| DeepSeek API 失败 | 检查 `.env` 中的 API Key，确保服务器能访问外网          |
| 数据库连接失败    | `docker compose ps postgres` 确认容器运行，检查密码     |
| 日志占满磁盘      | `sudo docker system prune -a --volumes`，配置 logrotate |
| 容器无法启动      | `docker compose logs <service-name>` 查看具体错误       |

---

## 快速部署脚本

创建 `deploy.sh`：
```bash
#!/bin/bash
set -e

echo "=== DebugAI 部署脚本 ==="

# 1. 检查环境
echo "检查 Docker..."
docker --version || { echo "Docker 未安装"; exit 1; }

# 2. 检查 .env 文件
if [ ! -f .env ]; then
    echo "错误：.env 文件不存在，请先配置"
    exit 1
fi

# 3. 下载依赖（可选，Docker 会自动处理）
echo "下载 Go 依赖..."
cd backend-go && go mod download && cd ..

echo "下载 Python 依赖..."
cd ai-python && pip download -r requirements.txt -d deps && cd ..

# 4. 构建并启动
echo "启动服务..."
docker compose up -d --build

# 5. 等待服务就绪
echo "等待服务启动..."
sleep 10

# 6. 验证
echo "验证服务..."
curl -s http://localhost:8000/health || echo "AI 服务未就绪"
curl -s http://localhost/ || echo "前端未就绪"

echo "部署完成！"
```

使用：
```bash
chmod +x deploy.sh
./deploy.sh
```

---

## 部署前待办清单

### 必需完成项（未完成将无法正常运行）

- [ ] 申请 DeepSeek API Key（访问 https://platform.deepseek.com/）
- [ ] 生成 JWT_SECRET（`openssl rand -base64 64`）
- [ ] 填写 `.env` 文件中的配置（数据库密码、API Key、JWT Secret）
- [ ] 确认服务器已安装 Docker、Docker Compose、Nginx

### 强烈建议完成项（生产环境必需）

- [ ] 配置 Nginx 反向代理 + HTTPS（生产环境必需）
- [ ] 配置数据库备份策略（每日自动备份）
- [ ] 配置 Sentry 错误监控（错误告警）
- [ ] 配置 logrotate 日志轮转（防止日志占满磁盘）
- [ ] 修改 CORS 配置为生产域名（修改 `backend-go/main.go`）
- [ ] 调整数据库连接池参数（MaxOpenConns=25, MaxIdleConns=25）
- [ ] 配置防火墙规则（仅开放必要端口）

---

## 配置参考

### .env 文件模板

### 1. 根目录 `.env` 文件

```bash
# 数据库配置
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your-strong-password-here
POSTGRES_DB=debugai
DB_PORT=5432

# JWT 配置（必须修改！）
JWT_SECRET=your-random-64-character-secret-key-change-in-production

# DeepSeek API（必须）
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 端口映射（可选）
BACKEND_PORT=8080
AI_PORT=8000
FRONTEND_PORT=80
```

**安全提示**：
- `.env` 文件必须加入 `.gitignore`，严禁提交到代码仓库
- `JWT_SECRET` 使用 `openssl rand -base64 64` 生成强密钥
- `POSTGRES_PASSWORD` 使用强密码（≥ 16 位，包含大小写字母、数字、特殊字符）

### 2. 根目录 `docker-compose.yml`

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: debugai-postgres
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-debugai}
    ports:
      - "${DB_PORT:-5432}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: [ "CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres}" ]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    networks:
      - debugai-network

  backend:
    build:
      context: ./backend-go
      dockerfile: Dockerfile
    container_name: debugai-backend
    environment:
      DB_HOST: postgres
      DB_USER: ${POSTGRES_USER:-postgres}
      DB_PASSWORD: ${POSTGRES_PASSWORD}
      DB_NAME: ${POSTGRES_DB:-debugai}
      DB_PORT: 5432
      JWT_SECRET: ${JWT_SECRET}
      AI_SERVICE_URL: http://ai-service:8000
    ports:
      - "${BACKEND_PORT:-8080}:8080"
    depends_on:
      postgres:
        condition: service_healthy
      ai-service:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - debugai-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  ai-service:
    build:
      context: ./ai-python
      dockerfile: Dockerfile
    container_name: debugai-ai-service
    environment:
      DEEPSEEK_API_KEY: ${DEEPSEEK_API_KEY}
    ports:
      - "${AI_PORT:-8000}:8000"
    healthcheck:
      test: [ "CMD-SHELL", "python -c \"import urllib.request; urllib.request.urlopen('http://localhost:8000/health')\"" ]
      interval: 30s
      timeout: 3s
      start_period: 10s
      retries: 3
    restart: unless-stopped
    networks:
      - debugai-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  frontend:
    build:
      context: ./frontend-vue
      dockerfile: Dockerfile
    container_name: debugai-frontend
    ports:
      - "${FRONTEND_PORT:-80}:80"
    depends_on:
      - backend
    restart: unless-stopped
    networks:
      - debugai-network

volumes:
  postgres_data:

networks:
  debugai-network:
    driver: bridge
```

### 3. 前端 `Dockerfile`（frontend-vue/Dockerfile）

```dockerfile
# -------------------- 构建阶段 --------------------
FROM node:18-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# -------------------- 运行阶段 --------------------
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### 4. 前端 `nginx.conf`（frontend-vue/nginx.conf）

```nginx
server {
    listen 80;
    server_name localhost;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

---

## 验证清单

部署后执行以下命令检查服务是否正常：

```bash
# 1. 所有服务状态正常
docker-compose ps
# 应显示：postgres (healthy)、backend (healthy)、ai-service (healthy)、frontend (running)

# 2. 健康检查通过
curl http://localhost:8000/health
# 应返回：{"status":"ok","message":"AI service is running"}

# 3. 数据库连接正常
docker-compose exec postgres psql -U postgres -d debugai -c "\dt"
# 应显示所有数据表

# 4. 日志系统工作正常
docker-compose logs --tail=50 backend | grep "Starting DebugAI"
docker-compose logs --tail=50 ai-service | grep "health"

# 5. 前端可访问
curl http://localhost/
# 应返回 HTML 内容

# 6. API 接口正常
# 注册用户
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"student_id":"test001","username":"test","password":"123456"}'

# 登录获取 token
TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"student_id":"test001","password":"123456"}' | jq -r '.token')

# 调用评价接口
curl -X POST http://localhost:8080/api/v1/ai/evaluate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"code":"print(\"hello\")","problem_description":"输出hello"}'
```

---

## 快速部署

```bash
# 1. 克隆代码并进入目录
git clone <your-repo-url>
cd debugai

# 2. 配置环境变量（必需）
nano .env  # 填写 JWT_SECRET、DEEPSEEK_API_KEY、POSTGRES_PASSWORD

# 3. 一键启动所有服务
docker compose up -d --build

# 4. 验证部署
docker compose ps
curl http://localhost:8000/health
```

**注意**：生产环境必须配置 HTTPS 和反向代理，详见上方"生产环境配置"章节。