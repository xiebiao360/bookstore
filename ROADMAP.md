# 图书商城微服务学习蓝图

> **项目定位**：教学导向的Go微服务实战项目  
> **学习目标**：系统掌握Go微服务架构设计、分布式系统核心技术、高并发场景解决方案  
> **时间规划**：Phase 1（2-3周）→ Phase 2（3-4周）→ Phase 3（可选，2-3周）

---

## 📌 需求概述

### 核心功能
1. **会员登录** - JWT鉴权、会话管理
2. **图书展示** - 分页查询、搜索、排序
3. **图书上架** - 会员发布图书商品
4. **图书购买** - 订单创建、库存管理、支付流程

### 技术约束
- **语言**：Go 1.21+
- **架构**：微服务（Phase 2开始拆分）
- **部署**：Kubernetes（Phase 3，可选）
- **接口**：纯API后端（RESTful + gRPC）

### 学习侧重
- 服务边界合理划分
- 分布式事务与数据一致性
- 高并发场景优化（库存扣减、秒杀）
- 服务治理（熔断、降级、限流）
- 可观测性（链路追踪、监控告警）

---

## 🏗️ 架构演进路线

```
┌─────────────────────────────────────────────────────────────┐
│  Phase 1: 单体分层架构（2-3周）                              │
│  目标：打好工程化基础、领域建模、测试体系                    │
│  技术：Gin + GORM + MySQL + Redis + Docker Compose         │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  Phase 2: 微服务拆分 + 分布式协调（3-4周）                  │
│  目标：服务边界、跨服务通信、最终一致性、熔断降级            │
│  技术：gRPC + RabbitMQ + Saga + Consul + OpenTelemetry     │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│  Phase 3: Kubernetes生产级部署（2-3周，可选）               │
│  目标：容器编排、高可用、自动扩缩容、混沌工程                │
│  技术：Helm + Prometheus + Grafana + Istio + Chaos Mesh    │
└─────────────────────────────────────────────────────────────┘
```

---

## 🎯 Phase 1: 单体分层架构（详细实施计划）

### 1.1 技术栈选型

| 分层 | 技术选型 | 选择理由 |
|------|---------|---------|
| **Web框架** | Gin | 性能优秀（HttpRouter）、中间件生态丰富、社区活跃 |
| **ORM** | GORM v2 | MySQL适配好、支持Hook、事务管理简单、迁移工具完善 |
| **数据库** | MySQL 8.0 | InnoDB事务引擎、主从复制成熟、国内运维经验丰富 |
| **缓存** | Redis 7.x | 会话存储、热点数据缓存、分布式锁（Phase 2） |
| **依赖注入** | Wire | Google官方、编译期生成、零运行时反射开销 |
| **配置管理** | Viper | 支持YAML/ENV、热重载、环境变量覆盖 |
| **日志** | zap | Uber出品、结构化日志、高性能（零分配） |
| **参数验证** | validator/v10 | Tag驱动、支持自定义规则、Gin默认集成 |
| **API文档** | swaggo/swag | 注释生成Swagger、交互式测试界面 |
| **测试** | testify + sqlmock | 断言库 + 数据库Mock、表驱动测试 |
| **本地环境** | Docker Compose | 一键启动MySQL+Redis+phpMyAdmin |

---

### 1.2 目录结构（DDD分层 + Clean Architecture）

```
bookstore/
├── cmd/
│   └── api/                          # 主程序入口
│       └── main.go                   # 启动HTTP服务、依赖注入
│
├── internal/                         # 私有代码（不可被外部import）
│   ├── domain/                       # 领域层（核心业务逻辑，不依赖外部框架）
│   │   ├── user/                     
│   │   │   ├── entity.go             # 用户实体（ID、Email、Password、CreatedAt）
│   │   │   ├── repository.go         # 仓储接口定义（依赖倒置原则）
│   │   │   └── service.go            # 领域服务（密码加密、业务规则校验）
│   │   ├── book/
│   │   │   ├── entity.go             # 图书实体（ISBN、Title、Price、Stock）
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── order/
│   │       ├── entity.go             # 订单实体（OrderNo、Items、Total、Status）
│   │       ├── repository.go
│   │       ├── service.go
│   │       └── status.go             # 订单状态机（防止非法状态跳转）
│   │
│   ├── infrastructure/               # 基础设施层（外部依赖实现）
│   │   ├── persistence/              
│   │   │   ├── mysql/
│   │   │   │   ├── db.go             # GORM连接初始化、连接池配置
│   │   │   │   ├── user_repo.go      # 实现domain/user/repository接口
│   │   │   │   ├── book_repo.go
│   │   │   │   ├── order_repo.go
│   │   │   │   └── tx_manager.go     # 事务管理器（支持嵌套事务）
│   │   │   └── redis/
│   │   │       └── session_store.go  # 会话存储（JWT黑名单、登录状态）
│   │   └── config/
│   │       └── config.go             # Viper加载配置、环境变量覆盖
│   │
│   ├── application/                  # 应用层（用例编排，协调多个领域服务）
│   │   ├── user/
│   │   │   ├── register.go           # 注册用例（校验→创建用户→发送欢迎邮件）
│   │   │   ├── login.go              # 登录用例（验证→生成JWT→记录会话）
│   │   │   └── dto.go                # 应用层DTO（与HTTP层解耦）
│   │   ├── book/
│   │   │   ├── list_books.go         # 列表查询用例（分页、排序、搜索）
│   │   │   ├── publish_book.go       # 上架用例（权限检查→创建图书）
│   │   │   └── dto.go
│   │   └── order/
│   │       ├── create_order.go       # 下单用例（锁库存→创建订单→扣库存）
│   │       └── dto.go
│   │
│   └── interface/                    # 接口层（外部交互）
│       ├── http/
│       │   ├── handler/              # HTTP处理器（解析请求→调用应用层→返回响应）
│       │   │   ├── user.go           # POST /api/v1/users/register
│       │   │   ├── book.go           # GET /api/v1/books
│       │   │   └── order.go          # POST /api/v1/orders
│       │   ├── middleware/           
│       │   │   ├── auth.go           # JWT解析、用户身份注入Context
│       │   │   ├── logger.go         # 请求日志（trace_id、耗时、状态码）
│       │   │   ├── recovery.go       # Panic恢复、错误上报
│       │   │   └── cors.go           # 跨域配置（Phase 1可选）
│       │   └── router.go             # 路由注册、中间件挂载
│       └── dto/                      # HTTP层DTO（请求/响应结构体）
│           ├── user.go               # RegisterRequest、LoginResponse
│           ├── book.go               # ListBooksRequest、BookResponse
│           └── order.go              # CreateOrderRequest、OrderResponse
│
├── pkg/                              # 可导出的公共库（可被其他项目复用）
│   ├── errors/                       
│   │   └── errors.go                 # 自定义错误类型（AppError、错误码定义）
│   ├── jwt/
│   │   └── jwt.go                    # JWT生成、解析、刷新Token
│   ├── logger/
│   │   └── logger.go                 # zap封装（糖化函数、上下文日志）
│   └── validator/
│       └── validator.go              # 自定义验证规则（ISBN、手机号）
│
├── test/                             # 测试目录
│   ├── integration/                  # 集成测试（真实数据库）
│   │   ├── user_test.go              # 注册登录完整流程测试
│   │   └── order_test.go             # 下单流程测试
│   └── fixtures/                     # 测试数据
│       └── data.sql
│
├── config/                           # 配置文件
│   ├── config.yaml                   # 默认配置
│   ├── config.dev.yaml               # 开发环境配置
│   └── config.prod.yaml              # 生产环境配置
│
├── docker-compose.yml                # 本地开发环境（MySQL + Redis + phpMyAdmin）
├── Dockerfile                        # 多阶段构建镜像
├── Makefile                          # 常用命令（run、test、lint、docker-up）
├── go.mod
├── go.sum
├── .golangci.yml                     # 代码检查配置
└── README.md                         # 项目说明、启动步骤

```

**设计亮点**：
1. **依赖倒置**：`domain`层定义接口，`infrastructure`层实现，便于Mock测试和替换实现
2. **清晰边界**：`user`/`book`/`order`三个聚合根边界清晰，为Phase 2拆分做准备
3. **分层隔离**：HTTP层不直接调用Repository，通过Application层协调
4. **可测试性**：每层都可独立测试，使用接口Mock外部依赖

---

### 1.3 核心功能实现清单

#### **Week 1: 脚手架 + 用户模块**

**Day 1-2: 项目初始化**
- [ ] 初始化Go模块（`go mod init github.com/xiebiao/bookstore`）
- [ ] 创建完整目录结构
- [ ] 编写`docker-compose.yml`（MySQL 8.0 + Redis 7.x + phpMyAdmin）
- [ ] 配置管理实现
  - Viper加载YAML
  - 环境变量覆盖（`BOOKSTORE_DB_PASSWORD`）
  - 配置热重载（fsnotify监听文件变化）
- [ ] 数据库连接
  - GORM初始化
  - 连接池配置（MaxOpenConns、MaxIdleConns、ConnMaxLifetime）
  - 慢查询日志
- [ ] Redis连接
  - go-redis客户端
  - 连接健康检查
- [ ] 日志系统
  - zap配置（开发模式console、生产模式JSON）
  - 全局Logger初始化

**Day 3-4: 用户注册**
- [ ] 用户实体定义（`domain/user/entity.go`）
  ```go
  type User struct {
      ID        uint      `gorm:"primaryKey"`
      Email     string    `gorm:"uniqueIndex;size:100"`
      Password  string    `gorm:"size:255"` // bcrypt哈希
      Nickname  string    `gorm:"size:50"`
      CreatedAt time.Time
      UpdatedAt time.Time
  }
  ```
- [ ] 仓储接口定义（`domain/user/repository.go`）
- [ ] MySQL仓储实现（`infrastructure/persistence/mysql/user_repo.go`）
- [ ] 用户领域服务（`domain/user/service.go`）
  - 密码加密（bcrypt.GenerateFromPassword，cost=12）
  - 邮箱格式校验
- [ ] 注册用例（`application/user/register.go`）
  - 邮箱唯一性检查
  - 调用领域服务创建用户
- [ ] HTTP处理器（`interface/http/handler/user.go`）
  - 参数验证（validator tag）
  - 调用应用层
  - 统一响应格式
- [ ] 路由注册（`interface/http/router.go`）
- [ ] 单元测试
  - Mock Repository测试Service层
  - 表驱动测试覆盖边界条件

**Day 5-6: 用户登录 + JWT鉴权**
- [ ] JWT工具封装（`pkg/jwt/jwt.go`）
  - 生成Access Token（有效期2小时）
  - 生成Refresh Token（有效期7天）
  - Token解析与验证
- [ ] Redis会话存储（`infrastructure/persistence/redis/session_store.go`）
  - 存储用户登录状态（Key: `session:{user_id}`）
  - JWT黑名单（用于登出）
- [ ] 登录用例（`application/user/login.go`）
  - 验证邮箱密码
  - 生成JWT
  - 记录会话到Redis
- [ ] 认证中间件（`interface/http/middleware/auth.go`）
  - 从Header提取Token（`Authorization: Bearer <token>`）
  - JWT解析
  - 检查黑名单
  - 用户信息注入Context
- [ ] 受保护路由测试（需要登录才能访问）

**Day 7: 错误处理 + 统一响应**
- [ ] 自定义错误类型（`pkg/errors/errors.go`）
  ```go
  type AppError struct {
      Code    int    `json:"code"`
      Message string `json:"message"`
      Err     error  `json:"-"` // 内部错误，不暴露
  }
  
  // 预定义错误
  var (
      ErrUserNotFound      = &AppError{Code: 40401, Message: "用户不存在"}
      ErrInvalidPassword   = &AppError{Code: 40101, Message: "密码错误"}
      ErrEmailDuplicate    = &AppError{Code: 40901, Message: "邮箱已被注册"}
  )
  ```
- [ ] 统一响应封装（`pkg/response/response.go`）
  ```go
  type Response struct {
      Code    int         `json:"code"`
      Message string      `json:"message"`
      Data    interface{} `json:"data,omitempty"`
  }
  ```
- [ ] 全局错误处理中间件
  - 捕获panic
  - AppError转换为HTTP响应
  - 未知错误日志记录

---

#### **Week 2: 图书模块 + 订单模块**

**Day 8-9: 图书上架**
- [ ] 图书实体定义（`domain/book/entity.go`）
  ```go
  type Book struct {
      ID          uint      `gorm:"primaryKey"`
      ISBN        string    `gorm:"uniqueIndex;size:20"`
      Title       string    `gorm:"size:200"`
      Author      string    `gorm:"size:100"`
      Publisher   string    `gorm:"size:100"`
      Price       int64     `gorm:"comment:价格（分）"`
      Stock       int       `gorm:"default:0"`
      CoverURL    string    `gorm:"size:500"`
      Description string    `gorm:"type:text"`
      PublisherID uint      `gorm:"index;comment:发布者用户ID"`
      CreatedAt   time.Time
      UpdatedAt   time.Time
  }
  ```
- [ ] 仓储接口与实现
- [ ] 上架用例（`application/book/publish_book.go`）
  - 权限检查（只有登录用户可发布）
  - ISBN格式验证（自定义validator）
  - 价格范围校验（1-999999分）
- [ ] HTTP接口实现
  - `POST /api/v1/books`
  - 认证中间件保护

**Day 10-11: 图书列表与搜索**
- [ ] 列表查询用例（`application/book/list_books.go`）
  - 分页参数（page、page_size，默认1/20）
  - 排序（price_asc、price_desc、created_at_desc）
  - 关键词搜索（LIKE查询，Phase 2会迁移到ES）
- [ ] 性能优化
  - 添加索引（title、price、created_at）
  - EXPLAIN分析慢查询
  - 查询结果缓存（Redis，TTL=5分钟）
- [ ] HTTP接口
  - `GET /api/v1/books?page=1&keyword=golang&sort=price_asc`
- [ ] 响应数据优化
  - 只返回必要字段（不返回Description）
  - 分页元信息（total、current_page、total_pages）

**Day 12-14: 订单模块（核心难点）**
- [ ] 订单实体设计
  ```go
  type Order struct {
      ID        uint        `gorm:"primaryKey"`
      OrderNo   string      `gorm:"uniqueIndex;size:32"`
      UserID    uint        `gorm:"index"`
      Total     int64       `gorm:"comment:总金额（分）"`
      Status    OrderStatus `gorm:"type:tinyint;default:1"`
      CreatedAt time.Time
      UpdatedAt time.Time
  }
  
  type OrderItem struct {
      ID       uint  `gorm:"primaryKey"`
      OrderID  uint  `gorm:"index"`
      BookID   uint  `gorm:"index"`
      Quantity int   `gorm:"default:1"`
      Price    int64 `gorm:"comment:下单时的单价"`
  }
  
  type OrderStatus int
  const (
      OrderStatusPending   OrderStatus = 1 // 待支付
      OrderStatusPaid      OrderStatus = 2 // 已支付
      OrderStatusShipped   OrderStatus = 3 // 已发货
      OrderStatusCompleted OrderStatus = 4 // 已完成
      OrderStatusCancelled OrderStatus = 5 // 已取消
  )
  ```
- [ ] 订单状态机（`domain/order/status.go`）
  ```go
  // 定义合法的状态流转
  var transitions = map[OrderStatus][]OrderStatus{
      OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
      OrderStatusPaid:      {OrderStatusShipped},
      OrderStatusShipped:   {OrderStatusCompleted},
  }
  
  func (o *Order) CanTransitionTo(target OrderStatus) bool {
      allowed, exists := transitions[o.Status]
      if !exists {
          return false
      }
      for _, s := range allowed {
          if s == target {
              return true
          }
      }
      return false
  }
  ```
- [ ] 下单用例（`application/order/create_order.go`）
  - **核心逻辑：防止超卖**
  ```go
  func (s *orderService) CreateOrder(ctx context.Context, userID uint, items []OrderItem) (*Order, error) {
      return s.txManager.Transaction(ctx, func(ctx context.Context) (*Order, error) {
          // 1. 锁定库存（悲观锁）
          for _, item := range items {
              book, err := s.bookRepo.LockByID(ctx, item.BookID) // SELECT FOR UPDATE
              if err != nil {
                  return nil, err
              }
              if book.Stock < item.Quantity {
                  return nil, errors.ErrInsufficientStock
              }
          }
          
          // 2. 计算订单金额（使用下单时的价格，防止改价攻击）
          var total int64
          for i := range items {
              book, _ := s.bookRepo.FindByID(ctx, items[i].BookID)
              items[i].Price = book.Price
              total += book.Price * int64(items[i].Quantity)
          }
          
          // 3. 创建订单
          order := &Order{
              OrderNo: generateOrderNo(), // 雪花算法或UUID
              UserID:  userID,
              Total:   total,
              Status:  OrderStatusPending,
          }
          if err := s.orderRepo.Create(ctx, order); err != nil {
              return nil, err
          }
          
          // 4. 创建订单明细
          for i := range items {
              items[i].OrderID = order.ID
          }
          if err := s.orderRepo.CreateItems(ctx, items); err != nil {
              return nil, err
          }
          
          // 5. 扣减库存
          for _, item := range items {
              if err := s.bookRepo.DecrStock(ctx, item.BookID, item.Quantity); err != nil {
                  return nil, err
              }
          }
          
          return order, nil
      })
  }
  ```
- [ ] 事务管理器（`infrastructure/persistence/mysql/tx_manager.go`）
  - 使用GORM的Transaction方法
  - 支持嵌套事务（Savepoint）
- [ ] HTTP接口
  - `POST /api/v1/orders`
  - 请求体：`{"items": [{"book_id": 1, "quantity": 2}]}`
- [ ] 单元测试
  - Mock场景：库存不足
  - Mock场景：图书不存在
  - 并发测试：100个goroutine同时下单

---

#### **Week 3: 工程化完善**

**Day 15-16: 依赖注入（Wire）**
- [ ] 安装Wire（`go install github.com/google/wire/cmd/wire@latest`）
- [ ] 编写Provider（`cmd/api/wire.go`）
  ```go
  //go:build wireinject
  // +build wireinject
  
  func InitializeApp() (*App, error) {
      wire.Build(
          // 配置
          config.Load,
          
          // 基础设施
          mysql.NewDB,
          redis.NewClient,
          
          // 仓储
          mysql.NewUserRepository,
          mysql.NewBookRepository,
          mysql.NewOrderRepository,
          
          // 服务
          user.NewService,
          book.NewService,
          order.NewService,
          
          // 应用层
          userapp.NewRegisterUseCase,
          userapp.NewLoginUseCase,
          bookapp.NewListBooksUseCase,
          
          // HTTP
          handler.NewUserHandler,
          handler.NewBookHandler,
          handler.NewOrderHandler,
          router.NewRouter,
          
          // 应用
          NewApp,
      )
      return nil, nil
  }
  ```
- [ ] 生成代码（`wire gen ./cmd/api`）
- [ ] 重构`main.go`使用Wire

**Day 17: Swagger文档**
- [ ] 安装swag（`go install github.com/swaggo/swag/cmd/swag@latest`）
- [ ] 编写API注释
  ```go
  // Register godoc
  // @Summary      用户注册
  // @Description  创建新用户账号
  // @Tags         用户
  // @Accept       json
  // @Produce      json
  // @Param        request body dto.RegisterRequest true "注册信息"
  // @Success      200 {object} response.Response{data=dto.UserResponse}
  // @Failure      400 {object} response.Response
  // @Router       /api/v1/users/register [post]
  func (h *UserHandler) Register(c *gin.Context) {
      // ...
  }
  ```
- [ ] 生成文档（`swag init -g cmd/api/main.go`）
- [ ] 挂载Swagger UI（`GET /swagger/*`）

**Day 18: Makefile + README**
- [ ] 编写Makefile
  ```makefile
  .PHONY: help run test lint docker-up docker-down migrate-up migrate-down swag
  
  help: ## 显示帮助信息
      @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
  
  run: ## 运行应用
      go run cmd/api/main.go
  
  test: ## 运行测试
      go test -v -cover -race ./...
  
  lint: ## 代码检查
      golangci-lint run --timeout=5m
  
  docker-up: ## 启动Docker环境
      docker-compose up -d
  
  docker-down: ## 停止Docker环境
      docker-compose down
  
  migrate-up: ## 执行数据库迁移
      go run cmd/migrate/main.go up
  
  swag: ## 生成Swagger文档
      swag init -g cmd/api/main.go
  
  wire: ## 生成依赖注入代码
      wire gen ./cmd/api
  ```
- [ ] 完善README
  - 项目介绍
  - 技术栈
  - 快速开始（docker-compose up → go run）
  - API文档地址
  - 目录结构说明

**Day 19-21: 性能分析与优化**
- [ ] 集成pprof
  ```go
  import _ "net/http/pprof"
  
  go func() {
      log.Println(http.ListenAndServe(":6060", nil))
  }()
  ```
- [ ] 压测工具
  - 使用wrk或vegeta压测注册/登录接口
  - 目标：单机QPS > 1000
- [ ] 性能分析
  - CPU Profile（`go tool pprof http://localhost:6060/debug/pprof/profile`）
  - 内存分配（`go tool pprof http://localhost:6060/debug/pprof/heap`）
  - goroutine泄漏检查
- [ ] 优化点
  - 数据库连接池调优
  - 减少不必要的JSON序列化
  - 缓存热点数据（图书列表）
- [ ] 慢查询分析
  - 开启MySQL慢查询日志（`slow_query_log=1`）
  - 使用EXPLAIN分析执行计划
  - 添加必要索引

---

### 1.4 Phase 1核心学习要点

#### **1. 仓储模式（Repository Pattern）**

**为什么需要仓储模式？**
- 领域层不应依赖具体的数据库实现（GORM、sqlx、MongoDB）
- 便于单元测试（Mock接口而非真实数据库）
- 未来切换数据库只需实现新的Repository

**示例代码**：
```go
// domain/user/repository.go（接口定义）
package user

import "context"

type Repository interface {
    Create(ctx context.Context, user *User) error
    FindByEmail(ctx context.Context, email string) (*User, error)
    FindByID(ctx context.Context, id uint) (*User, error)
}

// infrastructure/persistence/mysql/user_repo.go（实现）
package mysql

import (
    "context"
    "bookstore/internal/domain/user"
    "gorm.io/gorm"
)

type userRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) user.Repository {
    return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
    return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
    var u user.User
    err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

**测试时的Mock**：
```go
// user/service_test.go
type mockUserRepository struct {
    mock.Mock
}

func (m *mockUserRepository) Create(ctx context.Context, user *user.User) error {
    args := m.Called(ctx, user)
    return args.Error(0)
}

func TestUserService_Register(t *testing.T) {
    repo := new(mockUserRepository)
    repo.On("Create", mock.Anything, mock.Anything).Return(nil)
    
    svc := user.NewService(repo)
    err := svc.Register(context.Background(), "test@example.com", "password")
    
    assert.NoError(t, err)
    repo.AssertExpectations(t)
}
```

---

#### **2. 优雅的错误处理**

**核心原则**：
- 业务错误（用户不存在、密码错误）→ 返回明确的错误码和提示
- 系统错误（数据库宕机、网络超时）→ 记录详细日志，返回通用错误
- 避免敏感信息泄露（不要直接返回SQL错误给前端）

**实现**：
```go
// pkg/errors/errors.go
package errors

import "fmt"

type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"` // 内部错误，不序列化
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *AppError) Unwrap() error {
    return e.Err
}

// 预定义业务错误
var (
    ErrUserNotFound      = &AppError{Code: 40401, Message: "用户不存在"}
    ErrInvalidPassword   = &AppError{Code: 40101, Message: "密码错误"}
    ErrInsufficientStock = &AppError{Code: 40001, Message: "库存不足"}
)

// 包装系统错误
func Wrap(err error, msg string) *AppError {
    return &AppError{Code: 50000, Message: msg, Err: err}
}
```

**使用示例**：
```go
func (s *userService) Login(ctx context.Context, email, password string) (*User, error) {
    user, err := s.repo.FindByEmail(ctx, email)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, errors.ErrUserNotFound
        }
        return nil, errors.Wrap(err, "查询用户失败")
    }
    
    if !bcrypt.CheckPassword(user.Password, password) {
        return nil, errors.ErrInvalidPassword
    }
    
    return user, nil
}
```

**HTTP层处理**：
```go
func (h *UserHandler) Login(c *gin.Context) {
    user, err := h.useCase.Login(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        var appErr *errors.AppError
        if errors.As(err, &appErr) {
            // 业务错误
            c.JSON(http.StatusOK, response.Error(appErr.Code, appErr.Message))
            return
        }
        // 系统错误
        logger.Error("login failed", zap.Error(err))
        c.JSON(http.StatusInternalServerError, response.Error(50000, "系统错误"))
        return
    }
    c.JSON(http.StatusOK, response.Success(user))
}
```

---

#### **3. 事务处理（订单创建）**

**问题场景**：
下单流程包含：
1. 检查库存
2. 创建订单
3. 扣减库存

如果第3步失败，前两步必须回滚，否则会出现"订单已创建但库存未扣"的脏数据。

**解决方案：数据库事务**
```go
// infrastructure/persistence/mysql/tx_manager.go
package mysql

import (
    "context"
    "gorm.io/gorm"
)

type TxManager struct {
    db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
    return &TxManager{db: db}
}

func (m *TxManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 将事务DB注入到Context中
        txCtx := context.WithValue(ctx, "tx", tx)
        return fn(txCtx)
    })
}

// 仓储层需要支持从Context提取事务
func (r *orderRepository) getDB(ctx context.Context) *gorm.DB {
    if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
        return tx
    }
    return r.db
}
```

**订单服务使用事务**：
```go
func (s *orderService) CreateOrder(ctx context.Context, userID uint, items []OrderItem) (*Order, error) {
    var order *Order
    err := s.txManager.Transaction(ctx, func(ctx context.Context) error {
        // 1. 锁定库存（悲观锁，防止并发超卖）
        for _, item := range items {
            book, err := s.bookRepo.LockByID(ctx, item.BookID) // SELECT FOR UPDATE
            if err != nil {
                return err
            }
            if book.Stock < item.Quantity {
                return errors.ErrInsufficientStock
            }
        }
        
        // 2. 创建订单
        order = &Order{
            OrderNo: generateOrderNo(),
            UserID:  userID,
            Items:   items,
            Total:   calculateTotal(items),
            Status:  OrderStatusPending,
        }
        if err := s.orderRepo.Create(ctx, order); err != nil {
            return err
        }
        
        // 3. 扣减库存
        for _, item := range items {
            if err := s.bookRepo.DecrStock(ctx, item.BookID, item.Quantity); err != nil {
                return err
            }
        }
        
        return nil
    })
    
    return order, err
}
```

**关键点**：
- `SELECT FOR UPDATE`：锁定查询的行，防止其他事务修改
- 事务内所有操作要么全部成功（COMMIT），要么全部失败（ROLLBACK）
- Phase 2会引入Saga模式替代本地事务（因为微服务拆分后无法使用单机事务）

---

### 1.5 Phase 1交付物

完成后你将拥有：
1. ✅ 一个可运行的完整API服务
   - 用户注册/登录（JWT鉴权）
   - 图书上架/列表查询
   - 订单创建（防超卖）
2. ✅ 完整的单元测试覆盖（核心业务逻辑>80%）
3. ✅ Swagger交互式文档（`http://localhost:8080/swagger/`）
4. ✅ Docker Compose一键启动开发环境
5. ✅ 清晰的DDD分层架构（为Phase 2拆分做准备）

**技能掌握清单**：
- [x] Go工程化最佳实践（目录结构、依赖注入、配置管理）
- [x] 领域驱动设计（实体、仓储、领域服务）
- [x] 数据库事务与并发控制（悲观锁、乐观锁）
- [x] JWT鉴权流程
- [x] 错误处理与日志规范
- [x] 性能分析（pprof、慢查询优化）

---

## 🚀 Phase 2: 微服务拆分与分布式协调（预览）

**Phase 1结束后进入此阶段，以下为概要计划。**

### 2.1 服务拆分策略

**按领域边界拆分（DDD聚合根）**：

| 服务名 | 职责 | 数据库 | 核心技术 |
|--------|------|--------|----------|
| **user-service** | 用户认证、会员管理 | user_db | gRPC、JWT |
| **catalog-service** | 图书信息、搜索 | catalog_db + ElasticSearch | gRPC、ES |
| **order-service** | 订单管理 | order_db | gRPC、Saga |
| **inventory-service** | 库存管理 | inventory_db + Redis | gRPC、分布式锁 |
| **payment-service** | 支付（Mock） | payment_db | gRPC、幂等性 |
| **api-gateway** | 统一入口、鉴权、路由 | - | Gin、负载均衡 |

**服务依赖关系**：
```
API Gateway
    ↓
    ├─→ user-service（鉴权）
    ├─→ catalog-service（查询图书）
    └─→ order-service（下单）
            ↓
            ├─→ inventory-service（锁库存）
            └─→ payment-service（支付）
```

---

### 2.2 Phase 2核心技能点

#### **2.2.1 服务间通信**

**gRPC实现**：
```protobuf
// proto/user/v1/user.proto
syntax = "proto3";

package user.v1;
option go_package = "github.com/xiebiao/bookstore/proto/user/v1";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
}

message GetUserRequest {
  uint32 user_id = 1;
}

message GetUserResponse {
  uint32 id = 1;
  string email = 2;
  string nickname = 3;
}
```

**客户端调用（带熔断）**：
```go
// 使用sentinel-golang实现熔断
import sentinel "github.com/alibaba/sentinel-golang/api"

func (c *OrderClient) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
    entry, err := sentinel.Entry("order-service.CreateOrder", sentinel.WithTrafficType(base.Outbound))
    if err != nil {
        return nil, errors.New("服务熔断")
    }
    defer entry.Exit()
    
    resp, err := c.client.CreateOrder(ctx, req)
    if err != nil {
        sentinel.TraceError(entry, err)
        return nil, err
    }
    return resp, nil
}
```

---

#### **2.2.2 分布式事务（Saga模式）**

**问题**：Phase 1使用本地事务，微服务拆分后无法跨服务使用事务。

**解决方案：Saga编排模式**
```
下单流程：
1. order-service创建订单（状态=PENDING）
2. inventory-service锁定库存
3. payment-service扣款
4. order-service更新订单状态（状态=PAID）

如果第3步失败，需要执行补偿操作：
- inventory-service释放库存
- order-service取消订单
```

**手写Saga状态机**：
```go
type SagaStep struct {
    Name        string
    Action      func(ctx context.Context) error // 正向操作
    Compensate  func(ctx context.Context) error // 补偿操作
}

type Saga struct {
    steps []SagaStep
}

func (s *Saga) Execute(ctx context.Context) error {
    executed := []SagaStep{}
    
    for _, step := range s.steps {
        if err := step.Action(ctx); err != nil {
            // 回滚已执行的步骤
            for i := len(executed) - 1; i >= 0; i-- {
                _ = executed[i].Compensate(ctx)
            }
            return err
        }
        executed = append(executed, step)
    }
    return nil
}
```

**使用DTM框架（生产推荐）**：
```go
import "github.com/dtm-labs/dtm/client/dtmcli"

func CreateOrderSaga(orderID string) error {
    saga := dtmcli.NewSaga(dtmServer, gid).
        Add(inventoryURL+"/lock", inventoryURL+"/unlock", &LockInventoryReq{OrderID: orderID}).
        Add(paymentURL+"/pay", paymentURL+"/refund", &PayReq{OrderID: orderID})
    
    return saga.Submit()
}
```

---

#### **2.2.3 高并发库存扣减**

**问题**：秒杀场景下，大量并发扣库存会导致数据库锁竞争。

**解决方案：Redis + Lua脚本**
```lua
-- decrStock.lua
local key = KEYS[1]
local quantity = tonumber(ARGV[1])

local stock = tonumber(redis.call('GET', key))
if not stock or stock < quantity then
    return 0 -- 库存不足
end

redis.call('DECRBY', key, quantity)
return 1 -- 扣减成功
```

```go
func (s *inventoryService) DecrStock(ctx context.Context, bookID uint, quantity int) error {
    key := fmt.Sprintf("stock:%d", bookID)
    
    script := redis.NewScript(decrStockLua)
    result, err := script.Run(ctx, s.redis, []string{key}, quantity).Int()
    if err != nil {
        return err
    }
    
    if result == 0 {
        return errors.ErrInsufficientStock
    }
    
    // 异步同步到MySQL（消息队列）
    s.producer.Send("stock.decreased", &StockEvent{
        BookID:   bookID,
        Quantity: quantity,
    })
    
    return nil
}
```

---

#### **2.2.4 分布式追踪**

**使用OpenTelemetry**：
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (s *orderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
    tracer := otel.Tracer("order-service")
    ctx, span := tracer.Start(ctx, "CreateOrder")
    defer span.End()
    
    // 1. 调用inventory-service（自动传递trace context）
    span.AddEvent("Locking inventory")
    if err := s.inventoryClient.LockStock(ctx, req.Items); err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    // 2. 调用payment-service
    span.AddEvent("Processing payment")
    if err := s.paymentClient.Pay(ctx, req.Amount); err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    span.SetAttributes(attribute.String("order.id", order.ID))
    return order, nil
}
```

**效果**：在Jaeger UI可以看到完整的调用链路和耗时分布。

---

### 2.3 Phase 2学习路径

#### **Week 4-5: 服务拆分 + gRPC通信**
- [ ] 拆分user-service、catalog-service、order-service
- [ ] Protobuf定义接口
- [ ] 实现gRPC服务端和客户端
- [ ] 实现API Gateway（HTTP → gRPC转换）

#### **Week 6: 服务发现 + 负载均衡**
- [ ] 部署Consul集群
- [ ] 服务注册与健康检查
- [ ] 客户端负载均衡（gRPC resolver）

#### **Week 7: 分布式事务**
- [ ] 手写Saga状态机（理解原理）
- [ ] 引入DTM框架
- [ ] 实现下单Saga（锁库存→支付→确认订单）

#### **Week 8: 熔断降级 + 限流**
- [ ] sentinel-golang集成
- [ ] 熔断规则配置（错误率、慢调用比例）
- [ ] 降级预案（返回缓存数据、默认值）

#### **Week 9: 消息队列**
- [ ] RabbitMQ部署
- [ ] 订单事件驱动（订单创建→发送邮件/推送通知）
- [ ] 库存异步同步（Redis → MySQL）

#### **Week 10: 可观测性**
- [ ] OpenTelemetry集成
- [ ] Jaeger部署（查看链路追踪）
- [ ] Prometheus + Grafana监控大盘

---

## 🎓 Phase 3: Kubernetes生产级部署（可选）

### 3.1 目标
- 理解云原生运维体系
- 实现真正的高可用（多副本、自动扩缩容）
- 掌握K8s核心资源对象

### 3.2 核心技能点
- Helm Chart打包应用
- ConfigMap/Secret管理配置
- HPA（Horizontal Pod Autoscaler）
- Ingress + Cert-Manager（HTTPS）
- Istio服务网格（流量管理、金丝雀发布）
- Prometheus + Grafana监控
- ELK/Loki日志聚合
- Chaos Mesh混沌工程

### 3.3 学习路径（简要）
- Week 11: K8s基础 + 本地集群搭建
- Week 12: 应用部署 + 配置管理
- Week 13: 监控告警 + 日志聚合
- Week 14: 服务网格 + 灰度发布

---

## 📚 学习资源推荐

### Go语言进阶
- 《Go语言设计与实现》 - 深入理解底层原理
- 《Concurrency in Go》 - 并发模式
- Dave Cheney的博客 - 最佳实践

### 微服务架构
- 《微服务架构设计模式》（Chris Richardson）
- 《分布式系统模式》
- Martin Fowler的微服务博文

### 分布式系统
- MIT 6.824（分布式系统课程）
- 《数据密集型应用系统设计》（DDIA）
- 《凤凰架构》（周志明）

### Kubernetes
- 《Kubernetes in Action》
- 官方文档（kubernetes.io）
- CNCF Landscape（了解云原生生态）

---

## ✅ 学习检查点

### Phase 1检查清单
- [ ] 能独立搭建Go Web项目脚手架
- [ ] 理解DDD分层架构的优势
- [ ] 能正确处理数据库事务和并发
- [ ] 能编写可测试的代码（接口Mock）
- [ ] 能进行基本的性能分析和优化

### Phase 2检查清单
- [ ] 能合理划分微服务边界
- [ ] 理解gRPC通信原理
- [ ] 能设计Saga补偿事务
- [ ] 能处理分布式系统常见问题（网络分区、超时、重试）
- [ ] 能搭建完整的可观测性体系

### Phase 3检查清单
- [ ] 能编写Kubernetes资源清单
- [ ] 理解Pod、Service、Deployment的关系
- [ ] 能配置HPA实现自动扩缩容
- [ ] 能使用Prometheus + Grafana监控集群
- [ ] 能进行混沌工程实验

---

## 🚧 常见问题与解决方案

### Q1: Phase 1的单体架构会不会学了没用？
**A**: 恰恰相反！绝大多数系统不需要微服务，单体架构是基础。理解单体架构的分层设计，才能在Phase 2做出合理的服务拆分。

### Q2: 为什么不直接学K8s？
**A**: K8s是部署工具，不能解决分布式系统的本质问题（事务、一致性、容错）。Phase 2学会分布式协调，再上K8s才有意义。

### Q3: 项目太复杂，能不能简化？
**A**: 每个Phase都可以独立运行。如果时间紧张，完成Phase 1就已经掌握了Go工程化的核心能力。

### Q4: 代码量太大，写不完怎么办？
**A**: 学习重点在"理解设计思想"，不在"代码量"。关键模块我会提供示例代码，你只需理解原理并实现核心逻辑。

---

## 📝 附录：技术栈版本推荐

```yaml
# Go生态
go: 1.21+
gin: v1.9+
gorm: v1.25+
wire: v0.5+
viper: v1.17+
zap: v1.26+
validator: v10.15+

# 基础设施
mysql: 8.0+
redis: 7.x
rabbitmq: 3.12+
consul: 1.16+
elasticsearch: 8.x

# 可观测性
opentelemetry: v1.20+
jaeger: 1.50+
prometheus: 2.47+
grafana: 10.x

# Kubernetes（Phase 3）
kubernetes: 1.28+
helm: 3.13+
istio: 1.19+
```

---

**本蓝图将持续更新，随着学习进度调整细节。祝学习顺利！**
