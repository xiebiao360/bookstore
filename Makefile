.PHONY: help run test lint docker-up docker-down clean install-tools

# 默认目标
help: ## 显示帮助信息
	@echo "可用命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ========================================
# 开发环境
# ========================================

docker-up: ## 启动Docker环境（MySQL + Redis）
	docker compose up -d
	@echo "等待MySQL启动..."
	@sleep 5
	@echo "Docker环境已启动"
	@echo "MySQL: localhost:3306 (user: bookstore, password: bookstore123)"
	@echo "Redis: localhost:6379 (password: redis123)"
	@echo "phpMyAdmin: http://localhost:8081"

docker-down: ## 停止Docker环境
	docker compose down

docker-logs: ## 查看Docker日志
	docker compose logs -f

# ========================================
# 应用运行
# ========================================

run: ## 运行应用
	go run cmd/api/main.go

build: ## 编译应用
	go build -o bin/bookstore cmd/api/main.go

# ========================================
# 测试
# ========================================

test: ## 运行所有测试
	go test -v -cover -race ./...

test-unit: ## 运行单元测试
	go test -v -cover -race -short ./...

test-integration: ## 运行集成测试
	go test -v -cover ./test/integration/...

test-coverage: ## 生成测试覆盖率报告
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# ========================================
# 代码质量
# ========================================

lint: ## 代码检查
	golangci-lint run --timeout=5m

fmt: ## 格式化代码
	go fmt ./...
	goimports -w .

tidy: ## 整理依赖
	go mod tidy
	go mod verify

# ========================================
# 工具安装
# ========================================

install-tools: ## 安装开发工具
	@echo "安装golangci-lint..."
	@which golangci-lint > /dev/null || (go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && echo "✓ golangci-lint已安装")
	@echo "安装swag..."
	@which swag > /dev/null || (go install github.com/swaggo/swag/cmd/swag@latest && echo "✓ swag已安装")
	@echo "安装wire..."
	@which wire > /dev/null || (go install github.com/google/wire/cmd/wire@latest && echo "✓ wire已安装")
	@echo "所有工具已安装完成"

# ========================================
# 清理
# ========================================

clean: ## 清理构建产物
	rm -rf bin/ coverage.out coverage.html
	go clean -cache -testcache

# ========================================
# 数据库迁移（后续实现）
# ========================================

migrate-up: ## 执行数据库迁移
	@echo "数据库迁移功能待实现"

migrate-down: ## 回滚数据库迁移
	@echo "数据库迁移功能待实现"

# ========================================
# Swagger文档（后续实现）
# ========================================

swag: ## 生成Swagger文档
	swag init -g cmd/api/main.go -o docs/swagger
	@echo "Swagger文档已生成: docs/swagger"

# ========================================
# 依赖注入（后续实现）
# ========================================

wire: ## 生成依赖注入代码
	wire gen ./cmd/api
	@echo "Wire代码已生成"
