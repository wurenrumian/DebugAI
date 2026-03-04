#!/bin/bash

# 清理脚本 - 用于清理注册限流记录和测试数据
# 或者，直接运行docker exec -it debugai-redis redis-cli flushdb
set -e

echo "=== DebugAI 清理脚本 ==="
echo ""

# 检查 docker-compose.yml 是否存在
if [ ! -f "../docker-compose.yml" ]; then
    echo "错误：未找到 docker-compose.yml，请确保在 backend-go/scripts 目录下运行"
    exit 1
fi

cd ..

# 获取容器名称
REDIS_CONTAINER=$(docker-compose ps redis 2>/dev/null | grep -v NAME | awk '{print $1}' | head -n1)
POSTGRES_CONTAINER=$(docker-compose ps postgres 2>/dev/null | grep -v NAME | awk '{print $1}' | head -n1)

# 清理 Redis 限流记录
if [ -n "$REDIS_CONTAINER" ]; then
    echo "1. 清理 Redis 注册限流记录..."
    docker exec $REDIS_CONTAINER redis-cli --scan --pattern "rate_limit:register:*" | xargs -r docker exec $REDIS_CONTAINER redis-cli del
    echo "   ✅ 已清理所有注册限流记录"
else
    echo "   ⚠️  Redis 容器未运行，跳过"
fi

# 清理数据库测试用户（可选）
if [ -n "$POSTGRES_CONTAINER" ]; then
    echo ""
    read -p "2. 是否清理数据库中的所有用户记录？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "   清理 users 表..."
        docker exec $POSTGRES_CONTAINER psql -U postgres -d debugai -c "DELETE FROM users;" 2>/dev/null || echo "   ⚠️  删除失败，请检查数据库连接"
        echo "   ✅ 已清理所有用户记录"
    else
        echo "   ⏭️  跳过用户清理"
    fi
else
    echo "   ⚠️  PostgreSQL 容器未运行，跳过"
fi

echo ""
echo "=== 清理完成 ==="
