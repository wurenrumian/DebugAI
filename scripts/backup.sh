#!/usr/bin/env bash
# DebugAI PostgreSQL 备份脚本
#
# 通过 `docker compose exec -T postgres pg_dump` 导出整个数据库（纯 SQL，含 DROP ... IF EXISTS），
# gzip 压缩后保存为 $BACKUP_DIR/debugai_<库名>_<时间戳>.sql.gz，并删除超过保留天数的旧备份。
#
# 用法：
#   ./scripts/backup.sh
#
# 配置（环境变量优先，其次读取项目根目录 .env，最后使用默认值）：
#   BACKUP_DIR             备份目录（默认 <项目根目录>/backups，相对路径相对于项目根目录）
#   BACKUP_RETENTION_DAYS  保留天数（默认 14；设为 0 表示不自动清理）
#   COMPOSE_SERVICE        数据库服务名（默认 postgres）
#
# 定时任务示例（每天 03:00，见 deploy.md）：
#   0 3 * * * /opt/DebugAI/scripts/backup.sh >> /var/log/debugai-backup.log 2>&1
#
# 恢复：./scripts/restore.sh backups/debugai_xxx.sql.gz
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

log() { printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"; }
die() { log "ERROR: $*" >&2; exit 1; }

# 读取 .env 中的单个变量（不 source 整个文件，避免密码中的特殊字符被 shell 解释）
env_from_file() {
	local key="$1"
	[ -f .env ] || return 0
	grep -E "^[[:space:]]*${key}=" .env | tail -n 1 | cut -d= -f2- | sed -e 's/^["'\'']//' -e 's/["'\'']$//'
}

# 取值顺序：环境变量 > .env > 默认值
setting() {
	local key="$1" default="$2" value
	value="${!key:-}"
	if [ -z "$value" ]; then
		value="$(env_from_file "$key")"
	fi
	printf '%s' "${value:-$default}"
}

# 兼容 docker compose (v2) 与 docker-compose (v1)
if docker compose version >/dev/null 2>&1; then
	COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
	COMPOSE=(docker-compose)
else
	die "未找到 docker compose，请先安装 Docker Compose"
fi

BACKUP_DIR="$(setting BACKUP_DIR "$PROJECT_DIR/backups")"
RETENTION_DAYS="$(setting BACKUP_RETENTION_DAYS 14)"
SERVICE="$(setting COMPOSE_SERVICE postgres)"
DB_NAME="$(setting POSTGRES_DB debugai)"

case "$RETENTION_DAYS" in
	'' | *[!0-9]*) die "BACKUP_RETENTION_DAYS 必须是非负整数，当前值: $RETENTION_DAYS" ;;
esac
case "$DB_NAME" in
	*[!A-Za-z0-9_-]*) die "POSTGRES_DB 含有非法字符: $DB_NAME" ;;
esac

if ! "${COMPOSE[@]}" ps --services --filter status=running 2>/dev/null | grep -qx "$SERVICE"; then
	die "服务 $SERVICE 未运行（docker compose ps 查看状态）"
fi

umask 077
mkdir -p "$BACKUP_DIR"

TIMESTAMP="$(date '+%Y%m%d_%H%M%S')"
TARGET="$BACKUP_DIR/debugai_${DB_NAME}_${TIMESTAMP}.sql.gz"
TMP_FILE="$TARGET.partial"
trap 'rm -f "$TMP_FILE"' EXIT

log "开始备份数据库 -> $TARGET"

# 使用容器内的 POSTGRES_USER / POSTGRES_DB，通过本地 socket 连接，无需密码
# shellcheck disable=SC2016  # $POSTGRES_USER 等在容器内展开
"${COMPOSE[@]}" exec -T "$SERVICE" sh -c \
	'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --no-owner --no-privileges' \
	| gzip -c >"$TMP_FILE"

# 校验压缩包完整、内容非空
gzip -t "$TMP_FILE" || die "gzip 校验失败"
# （子 shell 关闭 pipefail：grep -q 提前退出会让 gzip 收到 SIGPIPE，这不是错误）
if ! (set +o pipefail; gzip -dc "$TMP_FILE" | head -c 4096 | grep -q "PostgreSQL database dump"); then
	die "备份内容异常（未找到 pg_dump 头部）"
fi

mv "$TMP_FILE" "$TARGET"
trap - EXIT
log "备份完成：$TARGET（$(du -h "$TARGET" | cut -f1)）"

if [ "$RETENTION_DAYS" -gt 0 ]; then
	log "清理 ${RETENTION_DAYS} 天前的旧备份"
	find "$BACKUP_DIR" -maxdepth 1 -type f -name 'debugai_*.sql.gz' -mtime +"$RETENTION_DAYS" -print -delete
fi
