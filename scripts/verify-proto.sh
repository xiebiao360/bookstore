#!/bin/bash
# Day 23: Protobuf 代码生成验证脚本

set -e

echo "======================================"
echo " Day 23: Protobuf 代码生成验证"
echo "======================================"
echo ""

# 1. 检查 protoc 版本
echo "1. 检查 protoc 版本"
protoc --version
echo ""

# 2. 检查 Go 插件
echo "2. 检查 Go 插件"
ls -1 ~/go/bin/ | grep protoc
echo ""

# 3. 统计生成的文件
echo "3. 统计生成的文件"
echo "总文件数: $(find proto -name "*.pb.go" -o -name "*_grpc.pb.go" | wc -l)"
echo ""

# 4. 各服务生成的文件
echo "4. 各服务生成的文件"
for service in user catalog inventory order payment; do
    count=$(find proto/$service -name "*.pb.go" -o -name "*_grpc.pb.go" 2>/dev/null | wc -l)
    echo "  $service-service: $count 个文件"
done
echo ""

# 5. 代码行数统计
echo "5. 代码行数统计"
echo "  .proto 文件: $(find proto -name "*.proto" | xargs wc -l | tail -1 | awk '{print $1}') 行"
echo "  .pb.go 文件: $(find proto -name "*.pb.go" | xargs wc -l | tail -1 | awk '{print $1}') 行"
echo "  _grpc.pb.go 文件: $(find proto -name "*_grpc.pb.go" | xargs wc -l | tail -1 | awk '{print $1}') 行"
echo ""

# 6. 验证关键接口生成
echo "6. 验证关键接口生成"
if grep -q "type UserServiceServer interface" proto/user/v1/user_grpc.pb.go; then
    echo "  ✓ UserServiceServer 接口已生成"
fi
if grep -q "type UserServiceClient interface" proto/user/v1/user_grpc.pb.go; then
    echo "  ✓ UserServiceClient 接口已生成"
fi
if grep -q "type CatalogServiceServer interface" proto/catalog/v1/catalog_grpc.pb.go; then
    echo "  ✓ CatalogServiceServer 接口已生成"
fi
if grep -q "type InventoryServiceServer interface" proto/inventory/v1/inventory_grpc.pb.go; then
    echo "  ✓ InventoryServiceServer 接口已生成"
fi
if grep -q "type OrderServiceServer interface" proto/order/v1/order_grpc.pb.go; then
    echo "  ✓ OrderServiceServer 接口已生成"
fi
if grep -q "type PaymentServiceServer interface" proto/payment/v1/payment_grpc.pb.go; then
    echo "  ✓ PaymentServiceServer 接口已生成"
fi
echo ""

# 7. 测试编译
echo "7. 测试编译生成的代码"
cd proto/user/v1 && go build -o /dev/null . && cd - > /dev/null
echo "  ✓ user-service 代码编译成功"
cd proto/catalog/v1 && go build -o /dev/null . && cd - > /dev/null
echo "  ✓ catalog-service 代码编译成功"
cd proto/inventory/v1 && go build -o /dev/null . && cd - > /dev/null
echo "  ✓ inventory-service 代码编译成功"
cd proto/order/v1 && go build -o /dev/null . && cd - > /dev/null
echo "  ✓ order-service 代码编译成功"
cd proto/payment/v1 && go build -o /dev/null . && cd - > /dev/null
echo "  ✓ payment-service 代码编译成功"
echo ""

echo "======================================"
echo " ✅ Day 23 验证完成！"
echo "======================================"
echo ""
echo "下一步："
echo "  make proto-gen  - 重新生成代码"
echo "  make proto-clean - 清理生成的代码"
echo ""
