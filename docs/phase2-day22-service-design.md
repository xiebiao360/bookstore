# Day 22: 微服务边界设计与接口定义

> **教学目标**：理解如何基于 DDD 原则合理拆分微服务  
> **核心原则**：单一职责、高内聚低耦合、数据库隔离  
> **教学方法**：对比 Phase 1 → Phase 2 的演进过程

---

## 📋 目录

1. [服务拆分原则](#服务拆分原则)
2. [6 个微服务设计](#6-个微服务设计)
3. [服务依赖关系](#服务依赖关系)
4. [数据库拆分策略](#数据库拆分策略)
5. [接口设计总览](#接口设计总览)

---

## 服务拆分原则

### 1. 基于 DDD 聚合根拆分

**Phase 1 的模块化设计**：

```
Phase 1: 单体分层架构
internal/
├── domain/
│   ├── user/        # 聚合根1：用户
│   ├── book/        # 聚合根2：图书
│   └── order/       # 聚合根3：订单
```

**Phase 2 的服务拆分**：

```
Phase 2: 微服务架构
services/
├── user-service/          # 聚合根1 → 独立服务
├── catalog-service/       # 聚合根2 拆分（只读）
├── inventory-service/     # 聚合根2 拆分（读写）
├── order-service/         # 聚合根3 拆分（订单）
├── payment-service/       # 聚合根3 拆分（支付）
└── api-gateway/           # 统一入口
```

**拆分依据**：

| 原则 | 说明 | 示例 |
|------|------|------|
| **业务能力** | 按业务功能划分 | 用户管理、图书管理、订单管理 |
| **单一职责** | 一个服务只做一件事 | catalog-service 只负责图书信息查询 |
| **数据独立** | 每个服务独立数据库 | user_db、catalog_db、order_db |
| **团队边界** | 便于团队独立开发 | 用户团队、商品团队、订单团队 |

---

### 2. 拆分的 DO & DON'T

#### ✅ DO（应该这样做）

```
1. 按业务能力拆分
   ✅ user-service: 用户注册、登录、认证
   ✅ catalog-service: 图书查询、搜索
   ✅ inventory-service: 库存管理

2. 数据库隔离
   ✅ 每个服务独立数据库
   ✅ 通过 API 访问其他服务的数据
   
3. 接口明确
   ✅ 使用 Protobuf 定义清晰的接口
   ✅ 版本化（v1、v2）

4. 独立部署
   ✅ 每个服务可以独立发布
   ✅ 不影响其他服务
```

#### ❌ DON'T（不应该这样做）

```
1. 过度拆分
   ❌ 一个表一个服务（太细粒度）
   ❌ 一个函数一个服务
   
2. 共享数据库
   ❌ 多个服务操作同一个表
   ❌ 直接跨库查询
   
3. 循环依赖
   ❌ A 调用 B，B 调用 A
   ❌ A → B → C → A
   
4. 按技术层拆分
   ❌ dao-service、service-service
   ❌ controller-service
```

---

## 6 个微服务设计

### 1. user-service（用户服务）

**职责**：
- 用户注册、登录
- JWT Token 生成和验证
- 用户信息管理

**为什么需要独立的 user-service？**

```
理由：
1. 用户认证是核心基础服务
   - 所有其他服务都需要验证用户身份
   - 统一的认证中心
   
2. 安全性要求高
   - 密码加密、Token 管理
   - 独立部署便于安全加固
   
3. 高频调用
   - 每个请求都需要验证 Token
   - 需要独立扩展
```

**技术栈**：
- gRPC 服务
- MySQL（user_db）
- Redis（Session 存储）

**数据模型**：

```sql
-- user_db.users
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,  -- bcrypt 加密
    nickname VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_email (email)
);
```

**对外接口**（Protobuf 定义）：

```protobuf
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
```

**端口分配**：
- gRPC: `9001`
- 健康检查: `9001/health`

**Phase 1 → Phase 2 迁移**：

```
Phase 1                          Phase 2
────────────────────────────────────────────
internal/domain/user/       →   services/user-service/internal/domain/user/
internal/application/user/  →   services/user-service/internal/application/user/
HTTP Handler                →   gRPC Handler
```

---

### 2. catalog-service（图书目录服务）

**职责**：
- 图书信息查询
- 图书列表、分页、排序
- 图书搜索

**为什么从 book 模块拆分为 catalog-service？**

```
理由：
1. 读写分离
   - catalog-service: 只读（查询、搜索）
   - inventory-service: 读写（库存扣减）
   
2. 性能优化
   - 查询服务可以独立扩展
   - 可以引入 ElasticSearch（Phase 3）
   
3. 职责清晰
   - catalog: 图书信息（What）
   - inventory: 库存管理（How many）
```

**技术栈**：
- gRPC 服务
- MySQL（catalog_db，只读）
- Redis（查询结果缓存）

**数据模型**：

```sql
-- catalog_db.books
CREATE TABLE books (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    isbn VARCHAR(20) UNIQUE NOT NULL,
    title VARCHAR(200) NOT NULL,
    author VARCHAR(100),
    publisher VARCHAR(100),
    price BIGINT NOT NULL COMMENT '价格（分）',
    cover_url VARCHAR(500),
    description TEXT,
    publisher_id BIGINT COMMENT '发布者用户ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_isbn (isbn),
    INDEX idx_title (title),
    INDEX idx_publisher_id (publisher_id)
);
```

**对外接口**：

```protobuf
service CatalogService {
  // 获取图书详情
  rpc GetBook(GetBookRequest) returns (GetBookResponse);
  
  // 图书列表（分页、排序）
  rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);
  
  // 搜索图书
  rpc SearchBooks(SearchBooksRequest) returns (SearchBooksResponse);
  
  // 发布图书（内部接口，供 api-gateway 调用）
  rpc PublishBook(PublishBookRequest) returns (PublishBookResponse);
}
```

**端口分配**：
- gRPC: `9002`

---

### 3. inventory-service（库存服务）

**职责**：
- 库存查询
- 库存扣减（下单）
- 库存补充（取消订单、补货）

**为什么需要独立的 inventory-service？**

```
理由：
1. 高并发场景
   - 秒杀、抢购需要高性能库存扣减
   - 使用 Redis + Lua 脚本优化
   
2. 数据一致性
   - 库存扣减是关键操作
   - 需要严格的并发控制
   
3. 业务复杂度
   - 库存预扣、释放、补偿
   - 独立服务便于维护
```

**技术栈**：
- gRPC 服务
- MySQL（inventory_db）
- Redis（库存缓存 + Lua 脚本）

**数据模型**：

```sql
-- inventory_db.inventory
CREATE TABLE inventory (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    book_id BIGINT UNIQUE NOT NULL COMMENT '图书ID（关联catalog_db.books.id）',
    stock INT NOT NULL DEFAULT 0 COMMENT '库存数量',
    version BIGINT NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_book_id (book_id)
);

-- 库存变更日志（用于审计和对账）
CREATE TABLE inventory_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    book_id BIGINT NOT NULL,
    change_type ENUM('DEDUCT', 'REFUND', 'RESTOCK') NOT NULL,
    quantity INT NOT NULL,
    before_stock INT NOT NULL,
    after_stock INT NOT NULL,
    order_id BIGINT COMMENT '关联订单ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_book_id (book_id),
    INDEX idx_order_id (order_id)
);
```

**对外接口**：

```protobuf
service InventoryService {
  // 查询库存
  rpc GetStock(GetStockRequest) returns (GetStockResponse);
  
  // 批量查询库存
  rpc BatchGetStock(BatchGetStockRequest) returns (BatchGetStockResponse);
  
  // 扣减库存（下单时调用）
  rpc DeductStock(DeductStockRequest) returns (DeductStockResponse);
  
  // 释放库存（订单取消时调用）
  rpc ReleaseStock(ReleaseStockRequest) returns (ReleaseStockResponse);
  
  // 补充库存（补货）
  rpc RestockInventory(RestockInventoryRequest) returns (RestockInventoryResponse);
}
```

**端口分配**：
- gRPC: `9004`

**Redis 库存缓存设计**：

```lua
-- decrStock.lua
-- 原子性扣减库存的 Lua 脚本

local key = KEYS[1]              -- stock:${book_id}
local quantity = tonumber(ARGV[1])

-- 获取当前库存
local stock = tonumber(redis.call('GET', key))

-- 库存不足
if not stock or stock < quantity then
    return 0
end

-- 扣减库存
redis.call('DECRBY', key, quantity)
return 1
```

**教学重点**：
1. catalog-service 和 inventory-service 的职责划分
2. Redis + Lua 脚本实现高性能库存扣减
3. 库存变更日志用于审计

---

### 4. order-service（订单服务）

**职责**：
- 订单创建
- 订单状态管理
- 订单查询

**为什么需要独立的 order-service？**

```
理由：
1. 复杂业务流程
   - 订单创建涉及多个服务调用
   - 需要 Saga 事务协调
   
2. 状态机管理
   - 订单状态流转（PENDING → PAID → SHIPPED）
   - 独立服务便于状态管理
   
3. 历史数据
   - 订单数据量大
   - 独立数据库便于归档
```

**技术栈**：
- gRPC 服务
- MySQL（order_db）

**数据模型**：

```sql
-- order_db.orders
CREATE TABLE orders (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_no VARCHAR(32) UNIQUE NOT NULL COMMENT '订单号',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    total BIGINT NOT NULL COMMENT '总金额（分）',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1待支付 2已支付 3已发货 4已完成 5已取消',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_order_no (order_no),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);

-- order_db.order_items
CREATE TABLE order_items (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL COMMENT '订单ID',
    book_id BIGINT NOT NULL COMMENT '图书ID',
    quantity INT NOT NULL COMMENT '数量',
    price BIGINT NOT NULL COMMENT '下单时的单价（分）',
    INDEX idx_order_id (order_id),
    INDEX idx_book_id (book_id)
);

-- 订单状态变更日志（用于审计）
CREATE TABLE order_status_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id BIGINT NOT NULL,
    from_status TINYINT NOT NULL,
    to_status TINYINT NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_order_id (order_id)
);
```

**对外接口**：

```protobuf
service OrderService {
  // 创建订单（Saga 编排入口）
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  
  // 更新订单状态
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (UpdateOrderStatusResponse);
  
  // 查询订单详情
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  
  // 查询用户订单列表
  rpc ListUserOrders(ListUserOrdersRequest) returns (ListUserOrdersResponse);
  
  // 取消订单
  rpc CancelOrder(CancelOrderRequest) returns (CancelOrderResponse);
}
```

**端口分配**：
- gRPC: `9003`

**订单创建流程（Saga）**：

```
1. order-service.CreateOrder()
   ├─→ 创建订单（状态=PENDING）
   │
2. ├─→ inventory-service.DeductStock()
   │   ├─ 成功：继续
   │   └─ 失败：返回错误（订单创建失败）
   │
3. ├─→ payment-service.Pay()
   │   ├─ 成功：更新订单状态为 PAID
   │   └─ 失败：
   │       ├─→ inventory-service.ReleaseStock()（补偿）
   │       └─→ 更新订单状态为 CANCELLED
   │
4. └─→ 返回订单结果
```

**教学重点**：
1. 订单状态机设计
2. Saga 事务编排（Week 7 详细讲解）
3. 订单创建流程的幂等性

---

### 5. payment-service（支付服务）

**职责**：
- 支付处理（Mock）
- 支付状态查询
- 退款处理

**为什么需要独立的 payment-service？**

```
理由：
1. 第三方集成
   - 对接支付宝、微信支付
   - 独立服务便于切换支付渠道
   
2. 安全性
   - 支付是敏感操作
   - 独立部署便于安全加固
   
3. 幂等性
   - 支付需要严格的幂等控制
   - 独立服务便于实现
```

**技术栈**：
- gRPC 服务
- MySQL（payment_db）

**数据模型**：

```sql
-- payment_db.payments
CREATE TABLE payments (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    payment_no VARCHAR(32) UNIQUE NOT NULL COMMENT '支付流水号',
    order_id BIGINT NOT NULL COMMENT '订单ID',
    amount BIGINT NOT NULL COMMENT '支付金额（分）',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1待支付 2已支付 3已退款 4失败',
    payment_method VARCHAR(20) COMMENT '支付方式：alipay、wechat',
    third_party_no VARCHAR(100) COMMENT '第三方支付流水号',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_payment_no (payment_no),
    INDEX idx_order_id (order_id)
);
```

**对外接口**：

```protobuf
service PaymentService {
  // 创建支付（Mock：随机成功/失败）
  rpc Pay(PayRequest) returns (PayResponse);
  
  // 查询支付状态
  rpc GetPaymentStatus(GetPaymentStatusRequest) returns (GetPaymentStatusResponse);
  
  // 退款（订单取消时调用）
  rpc Refund(RefundRequest) returns (RefundResponse);
}
```

**端口分配**：
- gRPC: `9005`

**Mock 实现**：

```go
// Phase 2 暂时 Mock 实现
// Phase 3 可以对接真实支付接口

func (s *PaymentService) Pay(ctx context.Context, req *pb.PayRequest) (*pb.PayResponse, error) {
    // Mock：70% 成功率
    success := rand.Intn(100) < 70
    
    if success {
        return &pb.PayResponse{
            Success:    true,
            PaymentNo:  generatePaymentNo(),
            Message:    "支付成功",
        }, nil
    }
    
    return &pb.PayResponse{
        Success: false,
        Message: "支付失败（余额不足）",
    }, nil
}
```

**教学重点**：
1. 支付幂等性设计
2. Mock 实现用于测试
3. 为后续对接真实支付接口预留扩展

---

### 6. api-gateway（API 网关）

**职责**：
- 统一入口（HTTP → gRPC）
- 路由转发
- 统一鉴权
- 协议转换
- 服务聚合

**为什么需要 api-gateway？**

```
理由：
1. 统一入口
   - 前端只需要知道一个地址
   - 隐藏内部服务复杂性
   
2. 协议转换
   - 外部：HTTP/JSON（前端友好）
   - 内部：gRPC/Protobuf（高性能）
   
3. 统一鉴权
   - 所有请求在网关验证 Token
   - 减少各服务重复鉴权代码
   
4. 服务聚合
   - 一次请求调用多个服务
   - 减少前端请求次数
```

**技术栈**：
- Gin（HTTP 服务）
- gRPC 客户端（调用后端服务）
- Consul（服务发现）

**架构设计**：

```
          前端（HTTP/JSON）
                 ↓
        ┌────────────────┐
        │  api-gateway   │ (8080)
        │  ┌──────────┐  │
        │  │ HTTP     │  │
        │  │ Handler  │  │
        │  └────┬─────┘  │
        │       ↓        │
        │  ┌──────────┐  │
        │  │ gRPC     │  │
        │  │ Client   │  │
        │  └────┬─────┘  │
        └───────┼────────┘
                ↓
    ┌───────────┴───────────┐
    │  gRPC（后端服务）      │
    ├───────────────────────┤
    ├─ user-service (9001)  │
    ├─ catalog-service (9002)│
    ├─ order-service (9003) │
    ├─ inventory-service (9004)│
    └─ payment-service (9005)│
```

**路由映射**：

| HTTP 端点 | gRPC 服务 | 方法 |
|----------|-----------|------|
| POST /api/v1/users/register | user-service | Register |
| POST /api/v1/users/login | user-service | Login |
| GET /api/v1/books | catalog-service | ListBooks |
| POST /api/v1/books | catalog-service | PublishBook |
| POST /api/v1/orders | order-service | CreateOrder |
| GET /api/v1/orders/:id | order-service | GetOrder |

**端口分配**：
- HTTP: `8080`（对外）

**教学重点**：
1. HTTP 和 gRPC 的协议转换
2. 统一鉴权中间件设计
3. gRPC 客户端管理（连接池）

---

## 服务依赖关系

### 依赖关系图

```
                    ┌─────────────┐
                    │ api-gateway │ (8080)
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ↓                  ↓                  ↓
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ user-service │  │catalog-service│  │order-service │
│    (9001)    │  │    (9002)     │  │    (9003)    │
└──────────────┘  └──────────────┘  └───────┬──────┘
                                             │
                        ┌────────────────────┼────────────┐
                        ↓                    ↓            ↓
                ┌───────────────┐  ┌─────────────┐ ┌──────────────┐
                │inventory-service│  │payment-service│ │user-service  │
                │     (9004)      │  │    (9005)     │ │(验证Token)   │
                └───────────────┘  └─────────────┘ └──────────────┘
```

### 依赖关系说明

| 服务 | 依赖的服务 | 依赖原因 |
|------|-----------|---------|
| **api-gateway** | 所有服务 | 路由转发、协议转换 |
| **order-service** | inventory-service | 扣减/释放库存 |
|  | payment-service | 支付处理 |
|  | user-service | 验证用户身份 |
|  | catalog-service | 获取图书信息（价格） |
| **其他服务** | 无 | 独立运行 |

### 依赖原则

```
✅ DO：
1. 单向依赖
   - order-service → inventory-service ✅
   - order-service → payment-service ✅

2. 通过 API 调用
   - 使用 gRPC 接口 ✅
   - 不直接访问数据库 ✅

❌ DON'T：
1. 循环依赖
   - A → B → A ❌
   
2. 跨库查询
   - order-service 直接查询 inventory_db ❌
   
3. 共享数据库
   - 多个服务操作同一个表 ❌
```

---

## 数据库拆分策略

### Phase 1 → Phase 2 数据库演进

**Phase 1: 单库**

```sql
bookstore (单库)
├── users           # 用户表
├── books           # 图书表（包含库存字段）
├── orders          # 订单表
└── order_items     # 订单明细表
```

**Phase 2: 库表拆分**

```sql
user_db
└── users           # 用户表

catalog_db
└── books           # 图书表（不含库存字段）

inventory_db
├── inventory       # 库存表（新增）
└── inventory_logs  # 库存变更日志（新增）

order_db
├── orders          # 订单表
├── order_items     # 订单明细表
└── order_status_logs # 订单状态日志（新增）

payment_db
└── payments        # 支付表（新增）
```

### 数据拆分原则

**1. 按服务边界拆分**

```
服务边界 = 数据库边界

user-service       → user_db
catalog-service    → catalog_db
inventory-service  → inventory_db
order-service      → order_db
payment-service    → payment_db
```

**2. 外键处理**

```
❌ Phase 1: 使用数据库外键
ALTER TABLE order_items 
ADD CONSTRAINT fk_order 
FOREIGN KEY (order_id) REFERENCES orders(id);

✅ Phase 2: 应用层外键
// 在应用代码中维护引用关系
// 不使用数据库外键（因为跨库）
```

**3. 数据一致性**

```
Phase 1: 本地事务
BEGIN TRANSACTION;
  -- 扣减库存
  UPDATE books SET stock = stock - 1 WHERE id = 1;
  -- 创建订单
  INSERT INTO orders (...) VALUES (...);
COMMIT;

Phase 2: 分布式事务（Saga）
Step 1: inventory-service.DeductStock()
Step 2: order-service.CreateOrder()
Step 3: payment-service.Pay()
  └─ 失败则执行补偿：
     - inventory-service.ReleaseStock()
     - order-service.CancelOrder()
```

---

## 接口设计总览

### gRPC 接口命名规范

```
服务名：{Domain}Service
方法名：{Action}{Resource}

示例：
service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

服务名：CatalogService （不是 BookService）
方法名：ListBooks （不是 List）
```

### 请求/响应消息规范

```protobuf
// ✅ 正确：明确的消息类型
message RegisterRequest {
  string email = 1;
  string password = 2;
  string nickname = 3;
}

message RegisterResponse {
  uint32 user_id = 1;
  string token = 2;
}

// ❌ 错误：通用消息类型
message Request {
  string type = 1;
  string data = 2;
}
```

### 接口版本化

```
proto/
├── user/
│   ├── v1/
│   │   └── user.proto        # 版本1
│   └── v2/
│       └── user.proto        # 版本2（兼容性升级）
```

---

## 总结

### Day 22 完成清单

- [x] 设计 6 个微服务的边界和职责
- [x] 定义服务依赖关系
- [x] 设计数据库拆分策略
- [x] 制定接口设计规范

### 核心设计原则

1. **单一职责**：每个服务只做一件事
2. **数据库隔离**：每个服务独立数据库
3. **接口明确**：使用 Protobuf 定义清晰接口
4. **单向依赖**：避免循环依赖

### 下一步

**Day 23: 创建 Protobuf 接口定义**

将为 6 个服务创建完整的 Protobuf 定义文件，并生成 Go 代码。

---

**教学要点**：

1. **为什么拆分？**：理解微服务的价值和代价
2. **如何拆分？**：基于 DDD 聚合根，遵循单一职责
3. **拆分后的挑战？**：数据一致性、服务调用、分布式事务
4. **Phase 1 vs Phase 2**：对比单体和微服务的差异

**记住**：微服务不是银弹，合理拆分才能发挥价值！
