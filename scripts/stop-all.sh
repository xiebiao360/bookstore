#!/bin/bash

# 图书商城微服务停止脚本
# 功能：停止所有微服务 + 基础设施
# 使用方法：./scripts/stop-all.sh

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 停止微服务
stop_services() {
    log_info "停止所有微服务..."

    local services=(
        "user-service"
        "catalog-service"
        "inventory-service"
        "payment-service"
        "order-service"
        "api-gateway"
    )

    for service in "${services[@]}"; do
        if [ -f "logs/$service.pid" ]; then
            local pid=$(cat logs/$service.pid)
            if kill -0 $pid 2>/dev/null; then
                log_info "停止 $service (PID: $pid)..."
                kill $pid
                rm logs/$service.pid
                log_success "$service 已停止"
            else
                log_warning "$service 进程不存在"
                rm logs/$service.pid
            fi
        else
            log_warning "$service PID文件不存在"
        fi
    done

    log_success "所有微服务已停止"
}

# 停止基础设施
stop_infrastructure() {
    log_info "停止基础设施..."

    docker compose down

    log_success "基础设施已停止"
}

# 主函数
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║             图书商城微服务 - 停止脚本               ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""

    # 检查是否在项目根目录
    if [ ! -f "go.mod" ]; then
        log_error "请在项目根目录执行此脚本"
        exit 1
    fi

    stop_services
    stop_infrastructure

    log_success "所有服务已停止！"
}

main
