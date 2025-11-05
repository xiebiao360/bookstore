# ==============================================================================
# 图书商城 Makefile - 教学导向的Go微服务项目
# ==============================================================================
#
# 教学说明：
#   Makefile 是项目自动化的核心工具，它将常用的命令封装成简短的目标（target）
#   这样可以：
#     1. 统一开发流程（所有人使用相同的命令）
#     2. 避免记忆复杂的命令参数
#     3. 实现复杂的任务编排（如 generate = swag + wire）
#
# 基础语法：
#   target: dependencies ## 帮助信息
#       @command        # @ 表示不打印命令本身，只显示输出
#
# 使用方式：
#   make help          - 查看所有可用命令
#   make docker-up     - 启动开发环境
#   make run           - 运行应用
#
# ==============================================================================

.PHONY: help run build test lint docker-up docker-down clean install-tools swag wire generate dev

# 默认目标：显示帮助信息
.DEFAULT_GOAL := help

help: ## 显示所有可用命令
	@echo "========================================"
	@echo " 图书商城 - 可用命令列表"
	@echo "========================================"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "教学提示："
	@echo "  1. 首次使用先运行: make install-tools"
	@echo "  2. 启动开发环境: make docker-up"
	@echo "  3. 运行应用: make run"
	@echo "  4. 查看API文档: http://localhost:8080/swagger/index.html"
	@echo ""

# ========================================
# 开发环境管理
# ========================================
# 教学说明：
#   本地开发需要MySQL和Redis，使用Docker Compose一键管理
#   优势：环境隔离、版本一致、快速启动
# ========================================

docker-up: ## 启动Docker环境（MySQL + Redis + phpMyAdmin）
	@echo "正在启动Docker环境..."
	@docker compose up -d
	@echo "等待MySQL初始化（约5秒）..."
	@sleep 5
	@echo ""
	@echo "✓ Docker环境已启动"
	@echo "========================================"
	@echo "服务访问信息："
	@echo "  MySQL:       localhost:3306"
	@echo "    用户名:     bookstore"
	@echo "    密码:       bookstore123"
	@echo "    数据库:     bookstore"
	@echo ""
	@echo "  Redis:       localhost:6379"
	@echo "    密码:       redis123"
	@echo ""
	@echo "  phpMyAdmin:  http://localhost:8081"
	@echo "========================================"
	@echo ""
	@echo "下一步：运行 make run 启动应用"

docker-down: ## 停止并删除Docker容器
	@echo "正在停止Docker环境..."
	@docker compose down
	@echo "✓ Docker环境已停止"

docker-restart: ## 重启Docker环境
	@make docker-down
	@make docker-up

docker-logs: ## 查看Docker容器日志（实时）
	@docker compose logs -f

docker-ps: ## 查看Docker容器状态
	@docker compose ps

docker-clean: ## 停止容器并清理数据卷（⚠️  会删除所有数据）
	@echo "⚠️  警告：此操作会删除所有数据库数据！"
	@read -p "确认继续？[y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		docker compose down -v; \
		echo "✓ 数据已清理"; \
	else \
		echo "已取消"; \
	fi

# ========================================
# 应用构建与运行
# ========================================
# 教学说明：
#   run: 开发模式，直接运行源码（热重载需配合air工具）
#   build: 编译成二进制文件，用于生产部署
#   dev: 一键启动完整开发环境
# ========================================

run: ## 运行应用（开发模式）
	@echo "启动应用..."
	@echo "访问地址："
	@echo "  API:     http://localhost:8080"
	@echo "  健康检查: http://localhost:8080/ping"
	@echo "  Swagger: http://localhost:8080/swagger/index.html"
	@echo ""
	@go run cmd/api/main.go

build: ## 编译应用为可执行文件
	@echo "编译应用..."
	@mkdir -p bin
	@go build -ldflags="-s -w" -o bin/bookstore ./cmd/api
	@echo "✓ 编译完成: bin/bookstore"
	@echo ""
	@echo "教学说明："
	@echo "  -ldflags='-s -w': 去除符号表和调试信息，减小二进制文件体积"
	@echo "  -s: 去除符号表（symbol table）"
	@echo "  -w: 去除DWARF调试信息"
	@echo "  注意: 使用 ./cmd/api 而非 cmd/api/main.go，这样会编译整个包"
	@ls -lh bin/bookstore

build-linux: ## 交叉编译Linux版本（用于容器部署）
	@echo "交叉编译Linux版本..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/bookstore-linux ./cmd/api
	@echo "✓ 编译完成: bin/bookstore-linux"
	@ls -lh bin/bookstore-linux

dev: docker-up run ## 一键启动完整开发环境（Docker + 应用）

watch: ## 热重载模式（需要先安装air: go install github.com/cosmtrek/air@latest）
	@which air > /dev/null || (echo "请先安装air: go install github.com/cosmtrek/air@latest" && exit 1)
	@air

# ========================================
# 测试
# ========================================
# 教学说明：
#   -v: 显示详细测试输出
#   -cover: 显示测试覆盖率
#   -race: 检测数据竞争（并发问题）
#   -short: 跳过集成测试（单元测试用）
# ========================================

test: ## 运行所有测试（单元测试 + 集成测试）
	@echo "运行所有测试..."
	@go test -v -cover -race ./...
	@echo ""
	@echo "教学说明："
	@echo "  -race: 检测并发数据竞争（Go的杀手级特性）"
	@echo "  示例: 两个goroutine同时修改同一个变量会被检测到"

test-unit: ## 仅运行单元测试（快速，不依赖外部服务）
	@echo "运行单元测试..."
	@go test -v -cover -race -short ./...
	@echo ""
	@echo "教学说明："
	@echo "  -short: 跳过标记为integration的测试"
	@echo "  单元测试使用Mock，速度快，适合TDD开发"

test-integration: ## 仅运行集成测试（需要真实数据库）
	@echo "运行集成测试（需要Docker环境）..."
	@docker compose ps | grep -q mysql || (echo "请先启动Docker: make docker-up" && exit 1)
	@go test -v -cover ./test/integration/...

test-coverage: ## 生成测试覆盖率HTML报告
	@echo "生成覆盖率报告..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total | awk '{print "总覆盖率: " $$3}'
	@echo "✓ 覆盖率报告已生成: coverage.html"
	@echo "  在浏览器中打开查看详细覆盖率"

test-bench: ## 运行性能基准测试
	@echo "运行性能测试..."
	@go test -bench=. -benchmem ./...
	@echo ""
	@echo "教学说明："
	@echo "  -bench=.: 运行所有Benchmark函数"
	@echo "  -benchmem: 显示内存分配统计"

# ========================================
# 代码质量检查
# ========================================
# 教学说明：
#   golangci-lint: 集成了50+种linter的工具，是Go生态的事实标准
#   常见检查项：
#     - errcheck: 检查是否忽略了错误
#     - staticcheck: 静态分析，检测潜在bug
#     - unused: 检测未使用的变量/函数
#     - gosimple: 简化代码建议
# ========================================

lint: ## 运行代码检查（golangci-lint）
	@echo "运行代码检查..."
	@which golangci-lint > /dev/null || (echo "请先安装golangci-lint: make install-tools" && exit 1)
	@golangci-lint run --timeout=5m
	@echo "✓ 代码检查通过"
	@echo ""
	@echo "教学说明："
	@echo "  golangci-lint 是多种linter的集合，包括："
	@echo "    - errcheck: 检查未处理的错误"
	@echo "    - staticcheck: 静态分析工具"
	@echo "    - gosimple: 代码简化建议"
	@echo "    - ineffassign: 检测无效赋值"

lint-fix: ## 自动修复可修复的问题
	@echo "自动修复代码问题..."
	@golangci-lint run --fix
	@echo "✓ 自动修复完成"

fmt: ## 格式化所有Go代码
	@echo "格式化代码..."
	@go fmt ./...
	@echo "✓ 代码格式化完成"
	@echo ""
	@echo "教学说明："
	@echo "  go fmt 使用gofmt工具统一代码风格"
	@echo "  Go社区有统一的代码格式，避免格式争论"

vet: ## 运行go vet检查
	@echo "运行go vet..."
	@go vet ./...
	@echo "✓ go vet检查通过"
	@echo ""
	@echo "教学说明："
	@echo "  go vet 是Go官方的静态分析工具，检测可疑代码"
	@echo "  例如：fmt.Printf格式字符串错误、atomic使用错误等"

tidy: ## 整理依赖包（添加缺失、移除未使用）
	@echo "整理Go模块依赖..."
	@go mod tidy
	@go mod verify
	@echo "✓ 依赖整理完成"
	@echo ""
	@echo "教学说明："
	@echo "  go mod tidy: 添加缺失的依赖，移除未使用的依赖"
	@echo "  go mod verify: 验证依赖包的完整性（检测篡改）"

check: fmt vet lint test ## 运行所有检查（格式化 + 静态分析 + 测试）
	@echo ""
	@echo "✅ 所有检查通过！代码质量良好"

# ========================================
# 开发工具安装
# ========================================
# 教学说明：
#   首次使用项目时运行此命令，安装所有必需的开发工具
#   这些工具不包含在go.mod中，需要单独安装
# ========================================

install-tools: ## 安装所有开发工具（golangci-lint, swag, wire）
	@echo "========================================"
	@echo " 安装开发工具"
	@echo "========================================"
	@echo ""
	@echo "[1/3] 检查golangci-lint..."
	@which golangci-lint > /dev/null && echo "  ✓ golangci-lint 已安装" || \
		(echo "  → 正在安装golangci-lint..." && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
		echo "  ✓ golangci-lint 安装完成")
	@echo ""
	@echo "[2/3] 检查swag..."
	@which swag > /dev/null && echo "  ✓ swag 已安装" || \
		(echo "  → 正在安装swag..." && \
		go install github.com/swaggo/swag/cmd/swag@latest && \
		echo "  ✓ swag 安装完成")
	@echo ""
	@echo "[3/3] 检查wire..."
	@which wire > /dev/null && echo "  ✓ wire 已安装" || \
		(echo "  → 正在安装wire..." && \
		go install github.com/google/wire/cmd/wire@latest && \
		echo "  ✓ wire 安装完成")
	@echo ""
	@echo "========================================"
	@echo "✅ 所有工具安装完成！"
	@echo "========================================"
	@echo ""
	@echo "已安装工具："
	@echo "  • golangci-lint: 代码检查工具"
	@echo "  • swag:          Swagger文档生成"
	@echo "  • wire:          依赖注入代码生成"
	@echo ""
	@echo "教学说明："
	@echo "  这些工具安装在 $$GOPATH/bin 目录"
	@echo "  请确保 $$GOPATH/bin 在你的 PATH 环境变量中"
	@echo "  验证: echo $$PATH | grep go"

check-tools: ## 检查开发工具是否已安装
	@echo "检查开发工具状态..."
	@echo ""
	@which golangci-lint > /dev/null && echo "✓ golangci-lint: $$(golangci-lint version --format short)" || echo "✗ golangci-lint: 未安装"
	@which swag > /dev/null && echo "✓ swag: $$(swag --version)" || echo "✗ swag: 未安装"
	@which wire > /dev/null && echo "✓ wire: 已安装" || echo "✗ wire: 未安装"
	@which go > /dev/null && echo "✓ go: $$(go version)" || echo "✗ go: 未安装"
	@which docker > /dev/null && echo "✓ docker: $$(docker --version)" || echo "✗ docker: 未安装"
	@echo ""
	@echo "如有工具未安装，运行: make install-tools"

# ========================================
# 清理构建产物
# ========================================

clean: ## 清理所有构建产物和缓存
	@echo "清理构建产物..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache
	@echo "✓ 清理完成"
	@echo ""
	@echo "已清理："
	@echo "  • bin/ 目录（可执行文件）"
	@echo "  • coverage.out, coverage.html（测试覆盖率）"
	@echo "  • Go缓存（build cache, test cache, module cache）"

clean-build: ## 仅清理编译产物（保留缓存）
	@echo "清理编译产物..."
	@rm -rf bin/
	@echo "✓ bin/ 已清理"

# ========================================
# 数据库迁移（后续实现）
# ========================================
# 教学说明：
#   数据库迁移（Migration）用于版本化管理数据库结构变更
#   常用工具：golang-migrate, goose
#   Phase 1暂时使用GORM的AutoMigrate，Phase 2会引入专业迁移工具
# ========================================

migrate-up: ## 执行数据库迁移（升级）
	@echo "⚠️  数据库迁移功能将在后续阶段实现"
	@echo ""
	@echo "当前阶段（Phase 1）："
	@echo "  使用GORM的AutoMigrate自动建表"
	@echo "  代码位置: internal/infrastructure/persistence/mysql/db.go"
	@echo ""
	@echo "Phase 2计划："
	@echo "  引入golang-migrate工具，实现版本化迁移"
	@echo "  每次数据库变更都有对应的up/down SQL文件"

migrate-down: ## 回滚数据库迁移
	@echo "⚠️  数据库迁移功能将在后续阶段实现"

migrate-create: ## 创建新的迁移文件（示例: make migrate-create name=add_users_table）
	@echo "⚠️  数据库迁移功能将在后续阶段实现"

# ========================================
# 代码生成工具
# ========================================

swag: ## 生成Swagger文档
	@echo "生成Swagger文档..."
	@swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	@echo "✓ Swagger文档已生成: docs/"
	@echo "  - docs/docs.go (Go代码)"
	@echo "  - docs/swagger.json (OpenAPI JSON)"
	@echo "  - docs/swagger.yaml (OpenAPI YAML)"
	@echo ""
	@echo "教学说明："
	@echo "  --parseDependency: 解析依赖包中的注释"
	@echo "  --parseInternal: 解析internal包中的注释"
	@echo "  启动应用后访问: http://localhost:8080/swagger/index.html"

wire: ## 生成依赖注入代码
	@echo "生成Wire依赖注入代码..."
	@cd cmd/api && wire
	@echo "✓ Wire代码已生成: cmd/api/wire_gen.go"
	@echo ""
	@echo "教学说明："
	@echo "  wire.go: 定义Provider和Injector（手写）"
	@echo "  wire_gen.go: Wire自动生成的依赖注入代码（不要手动修改）"
	@echo "  优势: 编译期生成，零运行时开销，类型安全"

generate: swag wire ## 运行所有代码生成工具（Swagger + Wire）
	@echo ""
	@echo "✓ 所有代码生成完成"
