#!/bin/bash

# 图书商城微服务一键启动脚本
# 功能：启动所有基础设施 + 6个微服务
# 使用方法：./scripts/start-all.sh

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "$1 未安装，请先安装"
        exit 1
    fi
}

# 检查端口是否被占用
check_port() {
    local port=$1
    local service=$2
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1 ; then
        log_warning "端口 $port 已被占用（$service），将尝试继续..."
        return 1
    fi
    return 0
}

# 等待服务就绪
wait_for_service() {
    local host=$1
    local port=$2
    local service=$3
    local max_wait=30
    local count=0

    log_info "等待 $service 就绪..."

    while ! nc -z $host $port 2>/dev/null; do
        sleep 1
        count=$((count + 1))
        if [ $count -ge $max_wait ]; then
            log_error "$service 启动超时"
            return 1
        fi
    done

    log_success "$service 已就绪"
    return 0
}

# 打印横幅
print_banner() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                                                      ║${NC}"
    echo -e "${BLUE}║          图书商城微服务 - 一键启动脚本              ║${NC}"
    echo -e "${BLUE}║                                                      ║${NC}"
    echo -e "${BLUE}║  Phase 2: 微服务架构 + 分布式协调                   ║${NC}"
    echo -e "${BLUE}║                                                      ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 步骤1：检查依赖
check_dependencies() {
    log_info "步骤1: 检查依赖..."

    check_command "docker"
    check_command "go"
    check_command "nc"

    log_success "依赖检查完成"
}

# 步骤2：启动基础设施
start_infrastructure() {
    log_info "步骤2: 启动基础设施（MySQL、Redis、RabbitMQ、Jaeger）..."

    # 启动Docker Compose
    docker compose up -d

    # 等待服务就绪
    wait_for_service localhost 3306 "MySQL"
    wait_for_service localhost 6379 "Redis"
    wait_for_service localhost 5672 "RabbitMQ"
    wait_for_service localhost 16686 "Jaeger"

    log_success "基础设施启动完成"
}

# 步骤3：编译所有微服务
build_services() {
    log_info "步骤3: 编译所有微服务..."

    local services=(
        "user-service"
        "catalog-service"
        "inventory-service"
        "payment-service"
        "order-service"
        "api-gateway"
    )

    for service in "${services[@]}"; do
        log_info "编译 $service..."
        (cd services/$service && go build -o bin/$service cmd/main.go)
        log_success "$service 编译完成"
    done

    log_success "所有微服务编译完成"
}

# 步骤4：启动所有微服务（后台运行）
start_services() {
    log_info "步骤4: 启动所有微服务..."

    # 创建日志目录
    mkdir -p logs

    # 启动user-service（端口50051）
    log_info "启动 user-service (gRPC:50051)..."
    nohup ./services/user-service/bin/user-service > logs/user-service.log 2>&1 &
    echo $! > logs/user-service.pid
    sleep 2

    # 启动catalog-service（端口50052）
    log_info "启动 catalog-service (gRPC:50052)..."
    nohup ./services/catalog-service/bin/catalog-service > logs/catalog-service.log 2>&1 &
    echo $! > logs/catalog-service.pid
    sleep 2

    # 启动inventory-service（端口50053）
    log_info "启动 inventory-service (gRPC:50053)..."
    nohup ./services/inventory-service/bin/inventory-service > logs/inventory-service.log 2>&1 &
    echo $! > logs/inventory-service.pid
    sleep 2

    # 启动payment-service（端口50054）
    log_info "启动 payment-service (gRPC:50054)..."
    nohup ./services/payment-service/bin/payment-service > logs/payment-service.log 2>&1 &
    echo $! > logs/payment-service.pid
    sleep 2

    # 启动order-service（端口50055）
    log_info "启动 order-service (gRPC:50055)..."
    nohup ./services/order-service/bin/order-service > logs/order-service.log 2>&1 &
    echo $! > logs/order-service.pid
    sleep 2

    # 启动api-gateway（端口8080）
    log_info "启动 api-gateway (HTTP:8080)..."
    nohup ./services/api-gateway/bin/api-gateway > logs/api-gateway.log 2>&1 &
    echo $! > logs/api-gateway.pid
    sleep 2

    log_success "所有微服务启动完成"
}

# 步骤5：健康检查
health_check() {
    log_info "步骤5: 健康检查..."

    local all_healthy=true

    # 检查API Gateway（HTTP健康检查）
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        log_success "✓ API Gateway (8080) - 健康"
    else
        log_warning "✗ API Gateway (8080) - 不健康"
        all_healthy=false
    fi

    # 检查gRPC服务（端口检查）
    local grpc_services=(
        "50051:user-service"
        "50052:catalog-service"
        "50053:inventory-service"
        "50054:payment-service"
        "50055:order-service"
    )

    for service in "${grpc_services[@]}"; do
        IFS=':' read -r port name <<< "$service"
        if nc -z localhost $port 2>/dev/null; then
            log_success "✓ $name ($port) - 健康"
        else
            log_warning "✗ $name ($port) - 不健康"
            all_healthy=false
        fi
    done

    if [ "$all_healthy" = true ]; then
        log_success "所有服务健康检查通过"
    else
        log_warning "部分服务健康检查失败，请查看日志"
    fi
}

# 步骤6：显示访问信息
show_access_info() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    启动成功！                        ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}📊 基础设施:${NC}"
    echo -e "  • MySQL:          http://localhost:3306"
    echo -e "  • phpMyAdmin:     http://localhost:8081"
    echo -e "  • Redis:          redis://localhost:6379"
    echo -e "  • RabbitMQ管理:   http://localhost:15672 (admin/admin123)"
    echo -e "  • Jaeger UI:      http://localhost:16686"
    echo ""
    echo -e "${BLUE}🚀 微服务:${NC}"
    echo -e "  • API Gateway:    http://localhost:8080"
    echo -e "  • User Service:   grpc://localhost:50051"
    echo -e "  • Catalog Svc:    grpc://localhost:50052"
    echo -e "  • Inventory Svc:  grpc://localhost:50053"
    echo -e "  • Payment Svc:    grpc://localhost:50054"
    echo -e "  • Order Service:  grpc://localhost:50055"
    echo ""
    echo -e "${BLUE}📁 日志文件:${NC}"
    echo -e "  • logs/user-service.log"
    echo -e "  • logs/catalog-service.log"
    echo -e "  • logs/inventory-service.log"
    echo -e "  • logs/payment-service.log"
    echo -e "  • logs/order-service.log"
    echo -e "  • logs/api-gateway.log"
    echo ""
    echo -e "${BLUE}🔧 常用命令:${NC}"
    echo -e "  • 查看日志:       tail -f logs/*.log"
    echo -e "  • 停止所有服务:   ./scripts/stop-all.sh"
    echo -e "  • 重启所有服务:   ./scripts/restart-all.sh"
    echo ""
}

# 主函数
main() {
    print_banner

    # 检查是否在项目根目录
    if [ ! -f "go.mod" ]; then
        log_error "请在项目根目录执行此脚本"
        exit 1
    fi

    check_dependencies
    start_infrastructure
    build_services
    start_services
    sleep 3  # 等待服务完全启动
    health_check
    show_access_info

    log_success "所有服务启动完成！"
}

# 执行主函数
main
