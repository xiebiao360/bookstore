#!/bin/bash

# 订单模块集成测试脚本
# 测试场景：
# 1. 正常下单流程
# 2. 库存不足场景
# 3. 并发下单场景（验证防超卖）

set -e

BASE_URL="http://localhost:8080/api/v1"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "================================================"
echo "订单模块集成测试"
echo "================================================"

# 辅助函数
function print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

function print_error() {
    echo -e "${RED}✗ $1${NC}"
}

function print_info() {
    echo -e "${YELLOW}➜ $1${NC}"
}

# 1. 准备测试数据：注册用户并登录
print_info "步骤1: 注册测试用户"
REGISTER_RESP=$(curl -s -X POST "$BASE_URL/users/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "buyer@test.com",
    "password": "Test1234",
    "nickname": "测试买家"
  }')

echo "注册响应: $REGISTER_RESP"

print_info "步骤2: 用户登录获取Token"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/users/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "buyer@test.com",
    "password": "Test1234"
  }')

echo "登录响应: $LOGIN_RESP"

# 提取access_token
ACCESS_TOKEN=$(echo $LOGIN_RESP | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ACCESS_TOKEN" ]; then
    print_error "登录失败，无法获取token"
    exit 1
fi

print_success "登录成功，Token: ${ACCESS_TOKEN:0:20}..."

# 2. 上架测试图书
print_info "步骤3: 上架测试图书（库存10本）"
BOOK_RESP=$(curl -s -X POST "$BASE_URL/books" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "title": "《Go并发编程实战》",
    "author": "测试作者",
    "isbn": "978-7-111-12345-6",
    "publisher": "测试出版社",
    "price": 8900,
    "stock": 10,
    "description": "测试用图书"
  }')

echo "上架响应: $BOOK_RESP"

# 提取book_id
BOOK_ID=$(echo $BOOK_RESP | grep -o '"book_id":[0-9]*' | cut -d':' -f2)

if [ -z "$BOOK_ID" ]; then
    print_error "图书上架失败"
    exit 1
fi

print_success "图书上架成功，图书ID: $BOOK_ID"

# 测试场景1: 正常下单
echo ""
echo "================================================"
echo "测试场景1: 正常下单（购买3本）"
echo "================================================"

ORDER1_RESP=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{
    \"items\": [
      {
        \"book_id\": $BOOK_ID,
        \"quantity\": 3
      }
    ]
  }")

echo "下单响应: $ORDER1_RESP"

ORDER1_ID=$(echo $ORDER1_RESP | grep -o '"order_id":[0-9]*' | cut -d':' -f2)

if [ -z "$ORDER1_ID" ]; then
    print_error "下单失败"
else
    print_success "下单成功，订单ID: $ORDER1_ID"
    print_info "预期结果：剩余库存 = 10 - 3 = 7本"
fi

# 测试场景2: 库存不足
echo ""
echo "================================================"
echo "测试场景2: 库存不足（购买8本，剩余7本）"
echo "================================================"

ORDER2_RESP=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d "{
    \"items\": [
      {
        \"book_id\": $BOOK_ID,
        \"quantity\": 8
      }
    ]
  }")

echo "下单响应: $ORDER2_RESP"

# 检查是否返回库存不足错误
if echo "$ORDER2_RESP" | grep -q "库存不足\|insufficient"; then
    print_success "正确返回库存不足错误"
else
    print_error "未正确处理库存不足场景"
fi

# 测试场景3: 并发下单（验证防超卖）
echo ""
echo "================================================"
echo "测试场景3: 并发下单（10个用户同时抢购剩余7本）"
echo "================================================"

print_info "创建10个并发请求，每个购买1本"

# 创建临时目录存储并发请求结果
TMP_DIR=$(mktemp -d)
SUCCESS_COUNT=0
FAIL_COUNT=0

# 并发发起10个订单请求
for i in {1..10}; do
    (
        RESP=$(curl -s -X POST "$BASE_URL/orders" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $ACCESS_TOKEN" \
          -d "{
            \"items\": [
              {
                \"book_id\": $BOOK_ID,
                \"quantity\": 1
              }
            ]
          }")

        echo "$RESP" > "$TMP_DIR/order_$i.json"
    ) &
done

# 等待所有并发请求完成
wait

# 统计成功和失败的订单数
for i in {1..10}; do
    if grep -q '"order_id"' "$TMP_DIR/order_$i.json"; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
done

echo ""
print_info "并发测试结果："
echo "  - 成功下单: $SUCCESS_COUNT 个"
echo "  - 失败下单: $FAIL_COUNT 个"

# 验证结果
# 期望：剩余7本库存，所以应该有7个成功，3个失败
if [ "$SUCCESS_COUNT" -eq 7 ] && [ "$FAIL_COUNT" -eq 3 ]; then
    print_success "防超卖机制测试通过！成功订单数 = 剩余库存数"
    print_success "悲观锁(SELECT FOR UPDATE)有效防止了超卖"
elif [ "$SUCCESS_COUNT" -gt 7 ]; then
    print_error "防超卖机制失败！出现超卖情况"
    print_error "成功订单数($SUCCESS_COUNT) > 剩余库存(7)"
else
    print_error "并发测试结果异常"
    print_error "预期成功7个，实际成功$SUCCESS_COUNT个"
fi

# 清理临时文件
rm -rf "$TMP_DIR"

echo ""
echo "================================================"
echo "集成测试完成"
echo "================================================"
echo ""
echo "教学要点总结："
echo "1. 正常下单流程：锁库存 → 创建订单 → 扣库存"
echo "2. 库存不足校验：在事务内进行，保证一致性"
echo "3. 并发防超卖：使用SELECT FOR UPDATE悲观锁"
echo "   - 多个事务同时执行时，只有一个能获得锁"
echo "   - 其他事务等待，直到前一个事务提交或回滚"
echo "   - 保证库存扣减的原子性和一致性"
echo ""
