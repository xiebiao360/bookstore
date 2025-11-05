# 图书商城 - Go微服务学习项目

> 本项目是一个教学导向的微服务系统，旨在系统性掌握Go微服务架构、分布式系统、高并发优化等核心能力。

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 项目概述

**核心功能**：
- 用户注册/登录（JWT鉴权）
- 图书商品展示与搜索
- 会员发布（上架）图书
- 图书购买与订单管理

**学习目标**：
- ✅ Go工程化最佳实践
- ✅ 领域驱动设计（DDD）
- ✅ 微服务架构设计
- ✅ 分布式系统核心技术
- ✅ 高并发场景优化

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

- Go 1.21+
- Docker & Docker Compose
- Make（可选，用于快捷命令）

### 1. 克隆项目

```bash
git clone https://github.com/xiebiao/bookstore.git
cd bookstore
```

### 2. 启动基础设施

```bash
# 启动MySQL + Redis + phpMyAdmin
make docker-up

# 或直接使用docker-compose
docker-compose up -d
```

**验证服务**：
- MySQL: `localhost:3306`（用户：`bookstore`，密码：`bookstore123`）
- Redis: `localhost:6379`（密码：`redis123`）
- phpMyAdmin: http://localhost:8081

### 3. 安装开发工具（可选）

```bash
make install-tools
```

将安装：
- `golangci-lint`：代码检查
- `swag`：Swagger文档生成
- `wire`：依赖注入

### 4. 安装依赖

```bash
go mod tidy
```

### 5. 运行应用

```bash
make run

# 或直接运行
go run cmd/api/main.go
```

应用将启动在 `http://localhost:8080`

### 6. 测试API

```bash
# 健康检查
curl http://localhost:8080/ping

# 用户注册（后续实现）
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123","nickname":"测试用户"}'
```

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

```bash
# 开发
make run                # 运行应用
make build              # 编译应用

# Docker环境
make docker-up          # 启动MySQL + Redis
make docker-down        # 停止Docker环境
make docker-logs        # 查看日志

# 测试
make test               # 运行所有测试
make test-unit          # 运行单元测试
make test-coverage      # 生成覆盖率报告

# 代码质量
make lint               # 代码检查
make fmt                # 格式化代码

# 工具
make install-tools      # 安装开发工具
make clean              # 清理构建产物
```

---

## 📚 技术栈

### Phase 1（当前）

| 分类 | 技术 | 说明 |
|------|------|------|
| Web框架 | Gin | 高性能HTTP框架 |
| ORM | GORM v2 | MySQL对象关系映射 |
| 数据库 | MySQL 8.0 | 主数据库 |
| 缓存 | Redis 7.x | 会话存储、热点数据缓存 |
| 配置管理 | Viper | 支持YAML、环境变量 |
| 日志 | zap | 高性能结构化日志 |
| 依赖注入 | Wire | 编译期依赖注入 |
| API文档 | Swagger | 交互式API文档 |

### Phase 2（计划）

- gRPC（服务间通信）
- RabbitMQ（消息队列）
- Consul（服务发现）
- DTM（分布式事务）
- OpenTelemetry + Jaeger（链路追踪）
- Sentinel（熔断降级）

---

## 📖 学习路径

### ✅ Phase 1: 单体分层架构（2-3周）

**Week 1**: 脚手架 + 用户模块
- [x] 项目初始化
- [x] Docker环境搭建
- [ ] 用户注册/登录
- [ ] JWT鉴权中间件

**Week 2**: 图书模块 + 订单模块
- [ ] 图书上架/列表查询
- [ ] 订单创建（防超卖）
- [ ] 事务处理

**Week 3**: 工程化完善
- [ ] 依赖注入（Wire）
- [ ] Swagger文档
- [ ] 性能分析与优化

详细计划见 [ROADMAP.md](./ROADMAP.md)

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

**Q: 为什么不直接用微服务？**  
A: 单体架构是基础，理解单体分层设计，才能合理拆分微服务。

**Q: 这个项目适合谁？**  
A: 适合有Go基础，想系统学习微服务架构和分布式系统的开发者。

**Q: 需要多长时间学完？**  
A: Phase 1约3周，Phase 2约4周，Phase 3（可选）约3周。

**Q: 遇到问题怎么办？**  
A: 先查看 [TEACHING.md](./TEACHING.md)，再检查代码注释，仍有疑问可提Issue。

---

**开始学习吧！** 🚀

建议先阅读 [ROADMAP.md](./ROADMAP.md) 了解完整学习路径，然后按照 [TEACHING.md](./TEACHING.md) 的指导逐步实现。
