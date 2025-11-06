#!/bin/bash
# 图书商城性能测试脚本
# 用途：在没有 wrk 的情况下进行简单的性能测试

set -e

echo "========================================"
echo " 图书商城性能基线测试"
echo "========================================"
echo ""

# 配置
BASE_URL="http://localhost:8080"
CONCURRENCY=50
DURATION=10

echo "测试配置："
echo "  目标地址: $BASE_URL"
echo "  并发数: $CONCURRENCY"
echo "  持续时间: ${DURATION}秒"
echo ""

# 测试健康检查接口
echo "1. 测试健康检查接口 (/ping)"
echo "----------------------------------------"
START=$(date +%s%N)
SUCCESS=0
FAILED=0

for i in $(seq 1 $CONCURRENCY); do
    (
        for j in $(seq 1 10); do
            RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $BASE_URL/ping)
            if [ "$RESPONSE" = "200" ]; then
                SUCCESS=$((SUCCESS + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        done
    ) &
done

wait

END=$(date +%s%N)
ELAPSED=$(echo "scale=2; ($END - $START) / 1000000000" | bc)
TOTAL_REQUESTS=$((CONCURRENCY * 10))
QPS=$(echo "scale=2; $TOTAL_REQUESTS / $ELAPSED" | bc)

echo "✓ 测试完成"
echo "  总请求数: $TOTAL_REQUESTS"
echo "  耗时: ${ELAPSED}秒"
echo "  QPS: $QPS"
echo ""

# 测试图书列表接口
echo "2. 测试图书列表接口 (/api/v1/books)"
echo "----------------------------------------"
START=$(date +%s%N)

for i in $(seq 1 $((CONCURRENCY / 5))); do
    (
        for j in $(seq 1 5); do
            curl -s -o /dev/null $BASE_URL/api/v1/books
        done
    ) &
done

wait

END=$(date +%s%N)
ELAPSED=$(echo "scale=2; ($END - $START) / 1000000000" | bc)
TOTAL_REQUESTS=$((CONCURRENCY / 5 * 5))
QPS=$(echo "scale=2; $TOTAL_REQUESTS / $ELAPSED" | bc)

echo "✓ 测试完成"
echo "  总请求数: $TOTAL_REQUESTS"
echo "  耗时: ${ELAPSED}秒"
echo "  QPS: $QPS"
echo ""

echo "========================================"
echo " 性能基线测试完成"
echo "========================================"
echo ""
echo "教学说明："
echo "  这是一个简单的压测脚本，真实场景建议使用："
echo "  - wrk: 高性能HTTP压测工具"
echo "  - ab (Apache Bench): Apache自带的压测工具"
echo "  - vegeta: Go编写的HTTP负载测试工具"
echo ""
echo "下一步："
echo "  1. 使用 pprof 分析性能瓶颈"
echo "  2. 针对性优化（数据库连接池、缓存、索引）"
echo "  3. 再次压测验证效果"
