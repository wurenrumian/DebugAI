#!/usr/bin/env bash
# DebugAI PostgreSQL 恢复脚本
#
# 把 scripts/backup.sh 生成的 .sql.gz 备份恢复到当前 compose 中的 postgres。
# 备份由 pg_dump --clean --if-exists 生成，恢复时会先删除再重建备份中包含的表/索引等对象，
# 整个恢复在单个事务中执行（--single-transaction），失败会整体回滚。
#
# 用法：
#   ./scripts/restore.sh backups/debugai_debugai_20250101_030000.sql.gz          # 交互确认
#   ./scripts/restore.sh backups/debugai_debugai_20250101_030000.sql.gz --yes    # 跳过确认（脚本化）
#
# 流程：停止 frontend/backend（避免恢复过程中有写入）-> 恢复 -> 重新启动 backend/frontend
# 建议恢复前先执行一次 ./scripts/backup.sh 保留当前状态。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

log() { printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"; }
die() { log "ERROR: $*" >&2; exit 1; }

usage() {
	echo "用法: $0 <backup.sql.gz> [--yes]" >&2
	exit 2
}

[ $# -ge 1 ] || usage
BACKUP_FILE="$1"
ASSUME_YES="${2:-}"
[ -z "$ASSUME_YES" ] || [ "$ASSUME_YES" = "--yes" ] || usage

# 在切换目录前把备份文件路径转成绝对路径
[ -f "$BACKUP_FILE" ] || die "备份文件不存在: $BACKUP_FILE"
BACKUP_FILE="$(cd "$(dirname "$BACKUP_FILE")" && pwd)/$(basename "$BACKUP_FILE")"
cd "$PROJECT_DIR"

if docker compose version >/dev/null 2>&1; then
	COMPOSE=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
	COMPOSE=(docker-compose)
else
	die "未找到 docker compose，请先安装 Docker Compose"
fi

SERVICE="${COMPOSE_SERVICE:-postgres}"

gzip -t "$BACKUP_FILE" || die "备份文件损坏（gzip 校验失败）: $BACKUP_FILE"

if ! "${COMPOSE[@]}" ps --services --filter status=running 2>/dev/null | grep -qx "$SERVICE"; then
	die "服务 $SERVICE 未运行，请先执行: docker compose up -d $SERVICE"
fi

if [ "$ASSUME_YES" != "--yes" ]; then
	echo "即将用以下备份覆盖当前数据库（备份中包含的表会被删除并重建）："
	echo "  $BACKUP_FILE"
	read -r -p "确认恢复？输入 yes 继续: " answer
	[ "$answer" = "yes" ] || die "已取消"
fi

restart_app() {
	log "重新启动 backend / frontend"
	"${COMPOSE[@]}" start backend frontend >/dev/null || log "WARNING: 启动 backend/frontend 失败，请手动执行 docker compose up -d"
}

log "停止 frontend / backend，避免恢复期间写入"
"${COMPOSE[@]}" stop frontend backend >/dev/null
trap restart_app EXIT

log "开始恢复：$BACKUP_FILE"
# shellcheck disable=SC2016  # $POSTGRES_USER 等在容器内展开
gzip -dc "$BACKUP_FILE" | "${COMPOSE[@]}" exec -T "$SERVICE" sh -c \
	'psql -v ON_ERROR_STOP=1 --single-transaction --quiet -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null

log "恢复完成"
