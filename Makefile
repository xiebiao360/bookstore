.PHONY: help start stop restart logs test build clean docker-up docker-down

# 默认目标：显示帮助
help: ## 显示帮助信息
	@echo ""
	@echo "图书商城微服务 - 可用命令："
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

# 一键启动所有服务
start: ## 启动所有微服务和基础设施
	@./scripts/start-all.sh

# 停止所有服务
stop: ## 停止所有微服务和基础设施
	@./scripts/stop-all.sh

# 重启所有服务
restart: ## 重启所有微服务
	@./scripts/restart-all.sh

# 查看日志
logs: ## 查看所有服务日志
	@tail -f logs/*.log

# 运行所有测试
test: ## 运行所有单元测试
	@echo "运行所有测试..."
	@go test -v ./pkg/...
	@go test -v ./services/*/internal/...

# 编译所有服务
build: ## 编译所有微服务
	@echo "编译所有微服务..."
	@cd services/user-service && go build -o bin/user-service cmd/main.go
	@cd services/catalog-service && go build -o bin/catalog-service cmd/main.go
	@cd services/inventory-service && go build -o bin/inventory-service cmd/main.go
	@cd services/payment-service && go build -o bin/payment-service cmd/main.go
	@cd services/order-service && go build -o bin/order-service cmd/main.go
	@cd services/api-gateway && go build -o bin/api-gateway cmd/main.go
	@echo "✓ 所有微服务编译完成"

# 清理编译产物和日志
clean: ## 清理编译产物和日志文件
	@echo "清理编译产物..."
	@rm -rf services/*/bin
	@rm -rf logs/*.log
	@rm -rf logs/*.pid
	@echo "✓ 清理完成"

# 仅启动Docker基础设施
docker-up: ## 仅启动Docker基础设施（MySQL、Redis、RabbitMQ、Jaeger）
	@echo "启动Docker基础设施..."
	@docker compose up -d
	@echo ""
	@echo "基础设施访问地址："
	@echo "  • MySQL:         localhost:3306"
	@echo "  • phpMyAdmin:    http://localhost:8081"
	@echo "  • Redis:         localhost:6379"
	@echo "  • RabbitMQ管理:  http://localhost:15672 (admin/admin123)"
	@echo "  • Jaeger UI:     http://localhost:16686"

# 停止Docker基础设施
docker-down: ## 停止Docker基础设施
	@docker compose down

# 代码检查
lint: ## 运行代码检查
	@echo "运行golangci-lint..."
	@golangci-lint run --timeout=5m || echo "提示：如未安装golangci-lint，请运行: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# 运行Saga测试
test-saga: ## 运行Saga框架测试
	@go test -v ./pkg/saga/...

# 运行熔断器测试
test-circuit-breaker: ## 运行熔断器测试
	@go test -v ./pkg/circuitbreaker/...

# 运行消息队列测试
test-mq: ## 运行消息队列测试
	@go test -v ./pkg/mq/...

# 运行追踪测试
test-tracing: ## 运行追踪框架测试
	@go test -v ./pkg/tracing/...

# 运行监控测试
test-metrics: ## 运行监控框架测试
	@go test -v ./pkg/metrics/...

# 格式化代码
fmt: ## 格式化Go代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "✓ 代码格式化完成"

# 更新依赖
deps: ## 更新Go依赖
	@echo "更新依赖..."
	@go mod tidy
	@go mod download
	@echo "✓ 依赖更新完成"

# 快速启动（仅基础设施 + 编译）
quick-start: docker-up build ## 快速启动（先启动基础设施，再编译服务）
	@echo ""
	@echo "✓ 快速启动完成"
	@echo "提示：运行 'make start' 启动所有微服务"
