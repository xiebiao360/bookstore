#!/bin/bash

# 图书商城微服务重启脚本
# 功能：重启所有微服务
# 使用方法：./scripts/restart-all.sh

set -e

# 颜色定义
BLUE='\033[0;34m'
NC='\033[0m'

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║             图书商城微服务 - 重启脚本               ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# 停止所有服务
./scripts/stop-all.sh

echo ""
echo "等待3秒..."
sleep 3

# 启动所有服务
./scripts/start-all.sh
