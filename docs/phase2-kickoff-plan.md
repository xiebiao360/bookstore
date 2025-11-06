# Phase 2: 微服务拆分与分布式协调 - 启动计划

> **教学阶段**：Phase 2（预计 3-4 周）  
> **核心使命**：遵循 TEACHING.md 原则，将 Phase 1 单体应用拆分为微服务架构  
> **教学重点**：理解微服务边界、掌握分布式系统核心技术

---

## 🎯 Phase 2 核心目标

### 1. 技术目标

**从单体到微服务的演进**：

```
Phase 1: 单体分层架构
┌─────────────────────────────────┐
│  bookstore-api (8080)           │
│  ├── user module                │
│  ├── book module                │
│  └── order module               │
│  ↓                               │
│  MySQL (单库)                    │
└─────────────────────────────────┘

                ↓ 拆分

Phase 2: 微服务架构
┌─────────────────────────────────────────────────┐
│  api-gateway (8080)                             │
│    ↓                                             │
│  ├─→ user-service (9001) → user_db              │
│  ├─→ catalog-service (9002) → catalog_db        │
│  ├─→ order-service (9003) → order_db            │
│  ├─→ inventory-service (9004) → inventory_db    │
│  └─→ payment-service (9005) → payment_db        │
│                                                  │
│  支撑服务：                                      │
│  ├── Consul (8500) - 服务发现                  │
│  ├── RabbitMQ (5672) - 消息队列                │
│  └── Jaeger (16686) - 链路追踪                 │
└─────────────────────────────────────────────────┘
```

### 2. 学习目标

**掌握的核心技能**：

| 技术领域 | 学习内容 | 应用场景 |
|---------|---------|---------|
| **服务拆分** | DDD 边界设计 | 合理划分微服务 |
| **服务通信** | gRPC + Protobuf | 高性能跨服务调用 |
| **分布式事务** | Saga 模式 | 订单流程一致性保证 |
| **服务发现** | Consul | 动态服务注册与发现 |
| **服务治理** | 熔断、降级、限流 | 提高系统稳定性 |
| **消息队列** | RabbitMQ | 异步解耦、削峰填谷 |
| **链路追踪** | OpenTelemetry + Jaeger | 分布式问题排查 |
| **监控告警** | Prometheus + Grafana | 系统可观测性 |

### 3. 教学原则（TEACHING.md 要求）

✅ **渐进式拆分**：
```
Week 5: 第一个微服务（user-service）
  → 掌握 gRPC 基础
  
Week 6: 完成服务拆分（6个服务）
  → 理解服务边界
  
Week 7: 分布式事务（Saga）
  → 解决数据一致性
  
Week 8: 服务治理（熔断、限流）
  → 提高系统稳定性
```

✅ **可运行性**：
- 每个服务都可以独立启动和测试
- 提供完整的 docker-compose 配置
- 每个阶段都有集成测试验证

✅ **教学注释丰富**：
- Protobuf 接口定义带详细注释
- Saga 补偿逻辑有完整说明
- 熔断降级策略有清晰解释

---

## 📅 Phase 2 学习路径（3-4 周）

### Week 5: 服务拆分 + gRPC 通信（Day 22-28）

#### **Day 22-23: 服务拆分设计**

**任务清单**：
- [ ] 设计 6 个微服务的边界和职责
- [ ] 设计服务间接口（Protobuf）
- [ ] 设计数据库拆分策略
- [ ] 绘制服务依赖关系图

**教学重点**：

1. **如何划分微服务边界？**

```
依据 DDD 的聚合根：

Phase 1 模块           Phase 2 微服务
─────────────────────────────────────────
user module       →   user-service
                      (用户认证、会员管理)

book module       →   catalog-service
                      (图书信息、搜索)

order module      →   order-service
                      (订单管理)
                  
                  →   inventory-service
                      (库存管理，从book模块拆分)
                  
                  →   payment-service
                      (支付，从order模块拆分)
                  
HTTP 路由         →   api-gateway
                      (统一入口、鉴权、路由)
```

2. **服务拆分的原则**：

```
✅ DO（应该这样做）：
- 按业务能力划分（DDD 聚合根）
- 单一职责原则（每个服务只做一件事）
- 数据库隔离（每个服务独立数据库）
- 接口清晰（明确的输入输出）

❌ DON'T（不应该这样做）：
- 按技术层划分（所有 DAO 一个服务）
- 过度拆分（一个表一个服务）
- 共享数据库（多个服务操作同一个库）
- 循环依赖（A 调用 B，B 调用 A）
```

**交付物**：
- `docs/phase2-service-design.md`（服务设计文档）
- `docs/phase2-api-design.md`（接口设计文档）
- 服务依赖关系图

---

#### **Day 24-25: user-service 实现**

**任务清单**：
- [ ] 创建 Protobuf 定义（proto/user/v1/user.proto）
- [ ] 生成 gRPC 代码
- [ ] 实现 user-service gRPC 服务端
- [ ] 迁移用户认证逻辑
- [ ] 编写 gRPC 客户端测试

**目录结构**：

```
services/
├── user-service/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # gRPC 服务启动
│   ├── internal/
│   │   ├── domain/                  # 从 Phase 1 迁移
│   │   │   └── user/
│   │   ├── application/
│   │   │   └── user/
│   │   ├── infrastructure/
│   │   │   └── persistence/
│   │   └── grpc/
│   │       ├── handler/
│   │       │   └── user_handler.go  # gRPC 处理器
│   │       └── server.go            # gRPC 服务器配置
│   ├── proto/
│   │   └── user/
│   │       └── v1/
│   │           └── user.proto       # Protobuf 定义
│   ├── Dockerfile
│   ├── Makefile
│   └── go.mod
```

**Protobuf 定义示例**：

```protobuf
// proto/user/v1/user.proto
syntax = "proto3";

package user.v1;
option go_package = "github.com/xiebiao/bookstore/services/user-service/proto/user/v1";

// 教学说明：用户服务接口定义
//
// 设计原则：
// 1. 只暴露必要的接口（单一职责）
// 2. 使用明确的请求/响应消息（类型安全）
// 3. 遵循 Protobuf 命名规范（CamelCase）
//
// 对比 Phase 1 HTTP API：
// - Phase 1: HTTP JSON（灵活但无类型安全）
// - Phase 2: gRPC Protobuf（强类型、高性能）

service UserService {
  // 用户注册
  rpc Register(RegisterRequest) returns (RegisterResponse);
  
  // 用户登录
  rpc Login(LoginRequest) returns (LoginResponse);
  
  // 验证 Token（供其他服务调用）
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
  
  // 获取用户信息（供其他服务调用）
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message RegisterRequest {
  string email = 1;
  string password = 2;
  string nickname = 3;
}

message RegisterResponse {
  uint32 user_id = 1;
  string token = 2;
}

message LoginRequest {
  string email = 1;
  string password = 2;
}

message LoginResponse {
  uint32 user_id = 1;
  string token = 2;
  string nickname = 3;
}

message ValidateTokenRequest {
  string token = 1;
}

message ValidateTokenResponse {
  bool valid = 1;
  uint32 user_id = 2;
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

**gRPC 服务端实现示例**：

```go
// internal/grpc/handler/user_handler.go
package handler

import (
    "context"
    
    pb "github.com/xiebiao/bookstore/services/user-service/proto/user/v1"
    "github.com/xiebiao/bookstore/services/user-service/internal/application/user"
)

// UserHandler gRPC 处理器
//
// 教学说明：
// gRPC Handler 的职责类似于 Phase 1 的 HTTP Handler
// - 接收 gRPC 请求
// - 调用应用层用例
// - 返回 gRPC 响应
//
// 对比 Phase 1：
// - Phase 1: gin.Context → HTTP 处理
// - Phase 2: context.Context + Protobuf 消息
type UserHandler struct {
    pb.UnimplementedUserServiceServer
    registerUC *user.RegisterUseCase
    loginUC    *user.LoginUseCase
}

func NewUserHandler(registerUC *user.RegisterUseCase, loginUC *user.LoginUseCase) *UserHandler {
    return &UserHandler{
        registerUC: registerUC,
        loginUC:    loginUC,
    }
}

// Register 用户注册
func (h *UserHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // 调用应用层用例（与 Phase 1 相同的业务逻辑）
    user, token, err := h.registerUC.Execute(ctx, req.Email, req.Password, req.Nickname)
    if err != nil {
        return nil, err  // gRPC 会自动转换为 gRPC 错误
    }
    
    return &pb.RegisterResponse{
        UserId: uint32(user.ID),
        Token:  token,
    }, nil
}

// Login 用户登录
func (h *UserHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
    user, token, err := h.loginUC.Execute(ctx, req.Email, req.Password)
    if err != nil {
        return nil, err
    }
    
    return &pb.LoginResponse{
        UserId:   uint32(user.ID),
        Token:    token,
        Nickname: user.Nickname,
    }, nil
}
```

**教学重点**：
1. Protobuf 消息设计（明确的字段类型和编号）
2. gRPC 服务定义（RPC 方法命名规范）
3. 代码生成流程（`protoc` 工具使用）
4. gRPC 错误处理（status code）

**交付物**：
- user-service 完整实现
- Protobuf 定义和生成代码
- gRPC 客户端测试

---

#### **Day 26-27: api-gateway 实现**

**任务清单**：
- [ ] 创建 api-gateway 项目结构
- [ ] 实现 HTTP → gRPC 转换
- [ ] 实现统一鉴权中间件
- [ ] 实现服务路由
- [ ] 负载均衡（客户端负载均衡）

**目录结构**：

```
services/
├── api-gateway/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── handler/
│   │   │   ├── user_handler.go      # HTTP → gRPC (user-service)
│   │   │   ├── book_handler.go      # HTTP → gRPC (catalog-service)
│   │   │   └── order_handler.go     # HTTP → gRPC (order-service)
│   │   ├── middleware/
│   │   │   ├── auth.go              # 调用 user-service 验证 Token
│   │   │   ├── logger.go
│   │   │   └── recovery.go
│   │   ├── grpc/
│   │   │   └── client/
│   │   │       ├── user_client.go   # user-service gRPC 客户端
│   │   │       ├── book_client.go
│   │   │       └── order_client.go
│   │   └── router/
│   │       └── router.go
│   ├── Dockerfile
│   ├── Makefile
│   └── go.mod
```

**HTTP → gRPC 转换示例**：

```go
// internal/handler/user_handler.go
package handler

import (
    "github.com/gin-gonic/gin"
    pb "github.com/xiebiao/bookstore/services/user-service/proto/user/v1"
)

// UserHandler API Gateway 的用户处理器
//
// 教学说明：
// API Gateway 的职责：
// 1. 接收 HTTP 请求
// 2. 转换为 gRPC 请求
// 3. 调用后端微服务
// 4. 转换 gRPC 响应为 HTTP 响应
//
// 为什么需要 API Gateway？
// - 统一入口（前端只需要知道一个地址）
// - 协议转换（HTTP → gRPC）
// - 统一鉴权（减少重复代码）
// - 服务聚合（一次请求调用多个服务）
type UserHandler struct {
    userClient pb.UserServiceClient  // gRPC 客户端
}

func NewUserHandler(userClient pb.UserServiceClient) *UserHandler {
    return &UserHandler{
        userClient: userClient,
    }
}

// Register 用户注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "注册信息"
// @Success      200 {object} RegisterResponse
// @Failure      400 {object} ErrorResponse
// @Router       /api/v1/users/register [post]
func (h *UserHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Message: "参数错误"})
        return
    }
    
    // HTTP 请求 → gRPC 请求
    grpcReq := &pb.RegisterRequest{
        Email:    req.Email,
        Password: req.Password,
        Nickname: req.Nickname,
    }
    
    // 调用 user-service
    grpcResp, err := h.userClient.Register(c.Request.Context(), grpcReq)
    if err != nil {
        // gRPC 错误 → HTTP 错误
        c.JSON(500, ErrorResponse{Message: err.Error()})
        return
    }
    
    // gRPC 响应 → HTTP 响应
    c.JSON(200, RegisterResponse{
        UserID: grpcResp.UserId,
        Token:  grpcResp.Token,
    })
}
```

**教学重点**：
1. HTTP 和 gRPC 的协议转换
2. gRPC 客户端创建和管理
3. API Gateway 的职责和价值
4. 统一错误处理

**交付物**：
- api-gateway 完整实现
- HTTP → gRPC 转换代码
- 统一鉴权中间件

---

#### **Day 28: Week 5 总结与测试**

**任务清单**：
- [ ] 编写 user-service 集成测试
- [ ] 编写 api-gateway 集成测试
- [ ] 编写端到端测试（HTTP → Gateway → user-service）
- [ ] 更新 docker-compose.yml
- [ ] 编写 Week 5 完成报告

**docker-compose 配置示例**：

```yaml
version: '3.8'

services:
  # Phase 1 的服务（保留用于对比）
  mysql:
    image: mysql:8.0
    # ... 配置

  redis:
    image: redis:7
    # ... 配置

  # Phase 2 新增：user-service
  user-service:
    build: ./services/user-service
    ports:
      - "9001:9001"
    environment:
      - DB_HOST=mysql
      - DB_NAME=user_db
      - REDIS_HOST=redis
    depends_on:
      - mysql
      - redis

  # Phase 2 新增：api-gateway
  api-gateway:
    build: ./services/api-gateway
    ports:
      - "8080:8080"
    environment:
      - USER_SERVICE_ADDR=user-service:9001
    depends_on:
      - user-service
```

**端到端测试示例**：

```go
// test/e2e/user_test.go
package e2e

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

// TestUserRegisterE2E 端到端测试：用户注册
//
// 测试流程：
// 1. HTTP 请求 → api-gateway (8080)
// 2. Gateway → user-service (9001) gRPC 调用
// 3. user-service → MySQL 存储
// 4. 响应返回
func TestUserRegisterE2E(t *testing.T) {
    // 准备测试数据
    email := fmt.Sprintf("test_%d@example.com", time.Now().Unix())
    req := map[string]string{
        "email":    email,
        "password": "Test1234",
        "nickname": "测试用户",
    }
    
    // 发送 HTTP 请求到 api-gateway
    resp := PostJSON(t, "http://localhost:8080/api/v1/users/register", req)
    
    // 验证响应
    assert.Equal(t, 200, resp.StatusCode)
    assert.NotEmpty(t, resp.Body.UserID)
    assert.NotEmpty(t, resp.Body.Token)
}
```

**交付物**：
- Week 5 完成报告
- 端到端测试
- 更新的 docker-compose 配置

---

### Week 6: 完成服务拆分（Day 29-35）

#### **Day 29-30: catalog-service 和 inventory-service**

**任务清单**：
- [ ] 实现 catalog-service（图书信息查询）
- [ ] 实现 inventory-service（库存管理）
- [ ] Protobuf 接口定义
- [ ] 数据库拆分（catalog_db、inventory_db）

**教学重点**：
1. 从 Phase 1 的 book 模块拆分为两个服务
2. catalog-service：只读服务（图书信息、搜索）
3. inventory-service：读写服务（库存扣减、补充）

---

#### **Day 31-32: order-service 和 payment-service**

**任务清单**：
- [ ] 实现 order-service（订单管理）
- [ ] 实现 payment-service（支付 Mock）
- [ ] Protobuf 接口定义
- [ ] 数据库拆分（order_db、payment_db）

**教学重点**：
1. 订单创建流程需要调用多个服务
2. payment-service 暂时 Mock（返回成功/失败）
3. 为 Week 7 的 Saga 事务做准备

---

#### **Day 33-34: 服务发现（Consul）**

**任务清单**：
- [ ] 部署 Consul 服务
- [ ] 实现服务注册（每个服务启动时注册）
- [ ] 实现服务发现（Gateway 通过 Consul 发现服务）
- [ ] 健康检查

**教学重点**：
1. 为什么需要服务发现？
2. Consul 的工作原理
3. 服务注册和发现的流程

---

#### **Day 35: Week 6 总结**

**任务清单**：
- [ ] 所有 6 个微服务联调测试
- [ ] 编写完整的端到端测试
- [ ] 更新架构文档
- [ ] 编写 Week 6 完成报告

---

### Week 7: 分布式事务（Saga）（Day 36-42）

**核心任务**：
- [ ] 理解 Saga 模式原理
- [ ] 手写简单的 Saga 编排器
- [ ] 实现订单流程的 Saga（创建订单→锁库存→支付）
- [ ] 实现补偿逻辑（支付失败→释放库存→取消订单）
- [ ] 引入 DTM 框架（可选）

**教学重点**：
1. 为什么微服务不能用本地事务？
2. Saga 的正向操作和补偿操作
3. 幂等性的重要性

---

### Week 8: 服务治理与可观测性（Day 43-49）

**核心任务**：
- [ ] 熔断降级（Sentinel）
- [ ] 限流策略
- [ ] 消息队列（RabbitMQ）
- [ ] 链路追踪（OpenTelemetry + Jaeger）
- [ ] 监控告警（Prometheus + Grafana）

---

## 📚 学习资源

### 推荐阅读

1. **gRPC 官方文档**
   - https://grpc.io/docs/languages/go/

2. **Protobuf 指南**
   - https://developers.google.com/protocol-buffers/docs/proto3

3. **Saga 模式**
   - 《微服务架构设计模式》第 4 章

4. **服务发现**
   - Consul 官方文档

5. **分布式追踪**
   - OpenTelemetry 文档

---

## ✅ Phase 2 Week 5 立即开始

**第一步：创建服务设计文档**

现在让我们开始 Day 22 的任务：设计 6 个微服务的边界和接口！

---

**Phase 2 启动！让我们带着 Phase 1 的扎实基础，进入微服务的世界！** 🚀
