# 图书商城 - Go微服务学习项目

> 本项目是一个教学导向的微服务系统，旨在系统性掌握Go微服务架构、分布式系统、高并发优化等核心能力。

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 项目概述

**核心功能**：
- ✅ 用户注册/登录（JWT鉴权 + Redis会话）
- ✅ 图书商品展示与搜索（分页、排序、关键词搜索）
- ✅ 会员发布（上架）图书
- ✅ 图书购买与订单管理（防超卖、悲观锁）

**已实现特性**：
- ✅ DDD分层架构（Domain + Application + Infrastructure + Interface）
- ✅ Wire依赖注入（编译期生成，零运行时开销）
- ✅ Swagger API文档（交互式测试界面）
- ✅ JWT认证 + Redis会话管理
- ✅ 防超卖机制（SELECT FOR UPDATE悲观锁）
- ✅ 统一错误处理与响应格式
- ✅ Docker Compose一键启动开发环境

**学习目标**：
- ✅ Go工程化最佳实践（目录结构、配置管理、日志处理）
- ✅ 领域驱动设计（实体、仓储、领域服务）
- ✅ 依赖注入模式（Wire自动化）
- ✅ API文档自动化（Swagger）
- 🚧 微服务架构设计（Phase 2计划）
- 🚧 分布式系统核心技术（Phase 2计划）
- 🚧 高并发场景优化（Phase 2计划）

**详细学习蓝图**：请查看 [ROADMAP.md](./ROADMAP.md)  
**教学指导原则**：请查看 [TEACHING.md](./TEACHING.md)

---

## 🏗️ 当前架构

**Phase 1: 单体分层架构**（当前阶段）

```
┌─────────────────────────────────────┐
│         HTTP API (Gin)              │
├─────────────────────────────────────┤
│      Application Layer              │
│   (Use Cases Orchestration)         │
├─────────────────────────────────────┤
│        Domain Layer                 │
│  (Business Logic & Entities)        │
├─────────────────────────────────────┤
│     Infrastructure Layer            │
│  (MySQL, Redis, Config, Logger)     │
└─────────────────────────────────────┘
```

**Phase 2**（计划）：微服务拆分 + gRPC + 分布式事务  
**Phase 3**（可选）：Kubernetes部署 + 服务网格

---

## 🚀 快速开始

### 前置要求

- **Go 1.21+** - [安装指南](https://golang.org/doc/install)
- **Docker & Docker Compose** - [安装指南](https://docs.docker.com/get-docker/)
- **Make**（可选，推荐）- Linux/macOS自带，Windows可用Git Bash

### 🎯 Phase 2 微服务一键启动（推荐）

**适用于**: Week 6-10 微服务架构学习

```bash
# 1. 克隆项目
git clone https://github.com/xiebiao/bookstore.git
cd bookstore

# 2. 一键启动所有服务（基础设施 + 6个微服务）
make start
```

启动后可访问：

**📊 基础设施**:
- MySQL: `localhost:3306` (bookstore/bookstore123)
- phpMyAdmin: http://localhost:8081
- Redis: `localhost:6379` (密码: redis123)
- RabbitMQ管理: http://localhost:15672 (admin/admin123)
- Jaeger UI: http://localhost:16686

**🚀 微服务**:
- API Gateway: http://localhost:8080
- User Service: `grpc://localhost:50051`
- Catalog Service: `grpc://localhost:50052`
- Inventory Service: `grpc://localhost:50053`
- Payment Service: `grpc://localhost:50054`
- Order Service: `grpc://localhost:50055`

**常用命令**:
```bash
make stop      # 停止所有服务
make restart   # 重启所有服务
make logs      # 查看所有日志
make help      # 查看所有可用命令
```

### 5分钟快速启动（Phase 1 单体版）

**适用于**: Week 1-5 单体架构学习

```bash
# 1. 克隆项目
git clone https://github.com/xiebiao/bookstore.git
cd bookstore

# 2. 启动Docker环境（MySQL + Redis）
make docker-up

# 3. 安装Go依赖
go mod tidy

# 4. 运行应用
make run
```

应用启动后访问：
- **API服务**: http://localhost:8080
- **Swagger文档**: http://localhost:8080/swagger/index.html
- **健康检查**: http://localhost:8080/ping

### Docker服务信息

启动 `make docker-up` 后，以下服务将可用：

| 服务 | 地址 | 凭证 |
|------|------|------|
| **MySQL** | `localhost:3306` | 用户: `bookstore`<br>密码: `bookstore123`<br>数据库: `bookstore` |
| **Redis** | `localhost:6379` | 密码: `redis123` |
| **phpMyAdmin** | http://localhost:8081 | 同MySQL凭证 |

### API使用示例

#### 1. 用户注册

```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "nickname": "测试用户"
  }'
```

#### 2. 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

响应中会返回 `access_token`，后续请求需要携带。

#### 3. 发布图书（需要登录）

```bash
curl -X POST http://localhost:8080/api/v1/books \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "isbn": "978-7-111-54742-6",
    "title": "Go语言实战",
    "author": "威廉·肯尼迪",
    "publisher": "机械工业出版社",
    "price": 5900,
    "stock": 100,
    "description": "Go语言经典教材"
  }'
```

#### 4. 查询图书列表

```bash
# 基础查询
curl http://localhost:8080/api/v1/books

# 带参数查询（分页、搜索、排序）
curl "http://localhost:8080/api/v1/books?page=1&page_size=10&keyword=Go&sort_by=price_asc"
```

#### 5. 创建订单（需要登录）

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "items": [
      {
        "book_id": 1,
        "quantity": 2
      }
    ]
  }'
```

**更多示例请查看 Swagger 文档**: http://localhost:8080/swagger/index.html

---

## 📂 项目结构

```
bookstore/
├── cmd/                          # 应用程序入口
│   └── api/                      # HTTP API服务
│       └── main.go
├── internal/                     # 私有代码（不可被外部导入）
│   ├── domain/                   # 领域层（核心业务逻辑）
│   │   ├── user/                 # 用户聚合
│   │   ├── book/                 # 图书聚合
│   │   └── order/                # 订单聚合
│   ├── infrastructure/           # 基础设施层
│   │   ├── persistence/          # 数据持久化（MySQL + Redis）
│   │   └── config/               # 配置管理
│   ├── application/              # 应用层（用例编排）
│   └── interface/                # 接口层（HTTP + DTO）
├── pkg/                          # 可导出的公共库
│   ├── errors/                   # 错误处理
│   ├── response/                 # 统一响应格式
│   ├── logger/                   # 日志封装
│   └── jwt/                      # JWT工具
├── test/                         # 测试
│   ├── integration/              # 集成测试
│   └── fixtures/                 # 测试数据
├── config/                       # 配置文件
│   └── config.yaml
├── docs/                         # 文档
│   └── adr/                      # 架构决策记录
├── docker-compose.yml            # 本地开发环境
├── Makefile                      # 快捷命令
├── ROADMAP.md                    # 学习蓝图
├── TEACHING.md                   # 教学指导原则
└── README.md
```

**设计亮点**：
- ✅ **清晰的分层架构**：Domain → Application → Interface
- ✅ **依赖倒置**：领域层定义接口，基础设施层实现
- ✅ **易于测试**：每层独立测试，使用Mock隔离外部依赖
- ✅ **为微服务拆分做准备**：user/book/order边界清晰

---

## 🛠️ 常用命令

运行 `make help` 查看所有可用命令。

### 开发环境

```bash
make docker-up          # 启动Docker环境（MySQL + Redis + phpMyAdmin）
make docker-down        # 停止Docker环境
make docker-restart     # 重启Docker环境
make docker-logs        # 查看Docker日志（实时）
make docker-ps          # 查看Docker容器状态
make docker-clean       # 清理Docker数据（⚠️ 会删除所有数据）
```

### 应用构建与运行

```bash
make run                # 运行应用（开发模式）
make build              # 编译应用为可执行文件
make build-linux        # 交叉编译Linux版本（用于Docker部署）
make dev                # 一键启动完整开发环境（Docker + 应用）
make watch              # 热重载模式（需要安装air）
```

### 测试

```bash
make test               # 运行所有测试（单元 + 集成）
make test-unit          # 仅运行单元测试（快速）
make test-integration   # 仅运行集成测试（需要Docker）
make test-coverage      # 生成测试覆盖率HTML报告
make test-bench         # 运行性能基准测试
```

### 代码质量

```bash
make lint               # 运行代码检查（golangci-lint）
make lint-fix           # 自动修复可修复的问题
make fmt                # 格式化所有Go代码
make vet                # 运行go vet检查
make tidy               # 整理依赖包
make check              # 运行所有检查（格式化 + 静态分析 + 测试）
```

### 代码生成

```bash
make swag               # 生成Swagger文档
make wire               # 生成Wire依赖注入代码
make generate           # 运行所有代码生成工具（Swagger + Wire）
```

### 工具管理

```bash
make install-tools      # 安装所有开发工具（golangci-lint, swag, wire）
make check-tools        # 检查开发工具是否已安装
make clean              # 清理所有构建产物和缓存
make clean-build        # 仅清理编译产物（保留缓存）
```

**提示**: 所有命令都有详细的教学说明，运行任何 `make` 命令都会显示相关教学内容。

---

## 📚 技术栈

### Phase 1（当前阶段 - 已完成）

| 分类 | 技术 | 版本 | 说明 |
|------|------|------|------|
| **Web框架** | Gin | v1.9+ | 高性能HTTP路由，基于HttpRouter |
| **ORM** | GORM | v2.0+ | MySQL对象关系映射，支持事务、钩子 |
| **数据库** | MySQL | 8.0 | InnoDB事务引擎，支持行级锁 |
| **缓存** | Redis | 7.x | 会话存储、JWT黑名单 |
| **配置管理** | Viper | v1.17+ | 支持YAML、环境变量覆盖 |
| **日志** | zap | v1.26+ | Uber高性能结构化日志（零分配） |
| **依赖注入** | Wire | v0.5+ | Google官方DI工具，编译期生成 |
| **API文档** | Swag | v1.16+ | Swagger文档生成，支持OpenAPI 3.0 |
| **参数验证** | validator | v10+ | Tag驱动的参数校验 |
| **密码加密** | bcrypt | - | 安全的哈希算法（防彩虹表） |
| **JWT** | jwt-go | v5+ | JSON Web Token实现 |

**核心特性**：
- ✅ DDD分层架构（Domain + Application + Infrastructure + Interface）
- ✅ Repository模式（依赖倒置，便于测试）
- ✅ 统一错误处理与响应格式
- ✅ 数据库事务管理（悲观锁防超卖）
- ✅ JWT鉴权 + Redis会话管理
- ✅ Swagger交互式API文档

### Phase 2（计划中）

**服务拆分 + 分布式协调**

| 技术 | 用途 |
|------|------|
| **gRPC** | 服务间高性能通信（替代HTTP JSON） |
| **RabbitMQ** | 消息队列（异步解耦、削峰填谷） |
| **Consul** | 服务发现与配置中心 |
| **DTM** | 分布式事务管理（Saga模式） |
| **OpenTelemetry + Jaeger** | 分布式链路追踪 |
| **Sentinel** | 熔断降级、流量控制 |
| **ElasticSearch** | 全文搜索（替代MySQL LIKE） |

### Phase 3（可选）

**Kubernetes生产级部署**

- Helm（应用打包）
- Prometheus + Grafana（监控告警）
- Istio（服务网格）
- Chaos Mesh（混沌工程）

---

## 📖 学习路径

### ✅ Phase 1: 单体分层架构（当前进度：Week 3 Day 18）

**Week 1**: 脚手架 + 用户模块 ✅
- [x] 项目初始化与目录结构设计
- [x] Docker环境搭建（MySQL + Redis）
- [x] 用户注册功能（bcrypt密码加密）
- [x] 用户登录功能（JWT Token生成）
- [x] JWT认证中间件
- [x] 统一错误处理与响应格式

**Week 2**: 图书模块 + 订单模块 ✅
- [x] Day 8-9: 图书上架功能（ISBN验证、价格范围校验）
- [x] Day 10-11: 图书列表查询（分页、排序、关键词搜索）
- [x] Day 12-14: 订单创建（**核心**：SELECT FOR UPDATE防超卖）
- [x] 数据库事务管理（TxManager）
- [x] 订单状态机（防止非法状态跳转）

**Week 3**: 工程化完善 ✅
- [x] Day 15-16: Wire依赖注入（编译期生成，零运行时开销）
- [x] Day 17: Swagger API文档（5个API接口，交互式测试）
- [x] Day 18: Makefile完善 + README更新（**当前阶段**）

**Week 4**: 性能优化与测试（计划中）
- [ ] Day 19: 集成测试编写（真实数据库测试）
- [ ] Day 20: 性能分析（pprof）与优化
- [ ] Day 21: 压力测试（wrk/vegeta）

### 🚧 Phase 2: 微服务拆分（计划：4-5周）

**服务拆分**:
- [ ] user-service（用户认证）
- [ ] catalog-service（图书目录）
- [ ] order-service（订单管理）
- [ ] inventory-service（库存管理）
- [ ] api-gateway（统一入口）

**核心技术**:
- [ ] gRPC通信
- [ ] Consul服务发现
- [ ] Saga分布式事务
- [ ] 熔断降级（Sentinel）
- [ ] 分布式追踪（Jaeger）

### 🔮 Phase 3: Kubernetes部署（可选：2-3周）

- [ ] Helm Chart打包
- [ ] ConfigMap/Secret配置管理
- [ ] HPA自动扩缩容
- [ ] Istio服务网格
- [ ] Prometheus监控

**详细计划**: 查看 [ROADMAP.md](./ROADMAP.md)  
**教学指导**: 查看 [TEACHING.md](./TEACHING.md)

---

## 🤝 贡献指南

本项目主要用于教学，欢迎提Issue讨论技术方案。

**提交PR前请确保**：
- 通过所有测试（`make test`）
- 代码检查无误（`make lint`）
- 遵循项目的代码规范

---

## 📄 许可证

[MIT License](LICENSE)

---

## 💡 常见问题

### Q: 为什么不直接用微服务？
**A**: 单体架构是基础，必须先理解：
- 如何划分领域边界（DDD聚合根）
- 如何管理数据库事务（本地事务 vs 分布式事务）
- 如何设计可测试的代码（Repository模式）

只有掌握了单体架构的分层设计，才能在Phase 2做出合理的服务拆分。

### Q: 这个项目适合谁？
**A**: 适合以下人群：
- ✅ 有Go基础（了解goroutine、channel、interface）
- ✅ 想系统学习微服务架构和分布式系统
- ✅ 希望通过实战项目掌握技术，而非只看文档
- ✅ 追求代码质量和工程化最佳实践

### Q: 需要多长时间学完？
**A**: 
- **Phase 1**（单体架构）：2-3周
- **Phase 2**（微服务拆分）：3-4周
- **Phase 3**（Kubernetes部署，可选）：2-3周

**建议节奏**: 每天2-3小时，扎实完成每个阶段的任务，不要急于求成。

### Q: 遇到问题怎么办？
**A**: 按以下顺序解决：
1. 查看 [TEACHING.md](./TEACHING.md) 教学指导
2. 阅读代码注释（每个关键模块都有详细注释）
3. 运行测试用例，查看预期行为
4. 查看 `docs/` 目录下的完成报告
5. 仍有疑问可提Issue讨论

### Q: 如何验证环境是否正常？
**A**: 运行以下命令：
```bash
# 检查工具安装状态
make check-tools

# 检查Docker服务
make docker-ps

# 运行健康检查
curl http://localhost:8080/ping
```

### Q: Wire和Swagger的代码需要手动维护吗？
**A**: 
- **Wire**: 只需维护 `cmd/api/wire.go`（定义Provider），运行 `make wire` 自动生成 `wire_gen.go`
- **Swagger**: 只需维护代码注释（`// @Summary` 等），运行 `make swag` 自动生成 `docs/`

**原则**: 永远不要手动修改自动生成的文件！

### Q: 如何查看已实现的功能？
**A**: 查看以下资源：
- **Swagger UI**: http://localhost:8080/swagger/index.html（所有API接口）
- **完成报告**: `docs/week*-completion-report.md`（每个阶段的详细说明）
- **测试用例**: `test/` 目录（通过测试了解功能行为）

### Q: Phase 1完成后我能掌握什么？
**A**: 
- ✅ Go工程化最佳实践（目录结构、依赖管理、配置管理）
- ✅ DDD分层架构（实体、仓储、领域服务、应用服务）
- ✅ 数据库事务与并发控制（悲观锁、乐观锁）
- ✅ JWT鉴权流程（生成、解析、刷新、黑名单）
- ✅ API设计规范（RESTful、统一响应格式、错误处理）
- ✅ 依赖注入模式（Wire自动化）
- ✅ API文档自动化（Swagger）

这些技能足以支撑你开发一个生产级的单体应用！

---

**开始学习吧！** 🚀

建议先阅读 [ROADMAP.md](./ROADMAP.md) 了解完整学习路径，然后按照 [TEACHING.md](./TEACHING.md) 的指导逐步实现。
