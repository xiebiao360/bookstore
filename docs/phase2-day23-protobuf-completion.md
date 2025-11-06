# Day 23: Protobuf 接口定义完成报告

> **教学目标**：掌握 Protobuf IDL 设计和 gRPC 代码生成  
> **完成时间**：2025-11-06  
> **核心成果**：5个微服务的完整 Protobuf 定义 + Go代码生成

---

## 📋 完成清单

- [x] 创建 Protobuf 目录结构
- [x] 定义 user-service 接口（5个RPC方法）
- [x] 定义 catalog-service 接口（5个RPC方法）
- [x] 定义 inventory-service 接口（6个RPC方法）
- [x] 定义 order-service 接口（5个RPC方法）
- [x] 定义 payment-service 接口（3个RPC方法）
- [x] 安装 protoc 编译器（v3.21.12）
- [x] 安装 Go 插件（protoc-gen-go + protoc-gen-go-grpc）
- [x] 生成所有服务的 Go 代码（10个文件）
- [x] 集成到 Makefile（3个新命令）

---

## 🎯 教学重点

### 1. 为什么使用 Protobuf？

**对比 JSON（Phase 1 使用的格式）**：

| 特性 | JSON | Protobuf |
|------|------|----------|
| **序列化格式** | 文本（可读） | 二进制（紧凑） |
| **性能** | 慢（需要解析字符串） | 快（二进制序列化） |
| **大小** | 大（冗余字段名） | 小（只传输值） |
| **类型安全** | 弱（运行时检查） | 强（编译期检查） |
| **版本兼容** | 手动维护 | 自动（字段编号） |
| **跨语言** | 需要手动定义 | 一份proto生成多语言 |

**性能对比**：

```
序列化速度：Protobuf 比 JSON 快 5-10 倍
反序列化：Protobuf 比 JSON 快 5-10 倍
消息大小：Protobuf 比 JSON 小 3-5 倍
```

**示例对比**：

```json
// JSON（112字节）
{
  "user_id": 12345,
  "email": "user@example.com",
  "nickname": "Alice"
}
```

```protobuf
// Protobuf（约30字节，二进制格式）
message User {
  uint64 user_id = 1;
  string email = 2;
  string nickname = 3;
}
```

---

### 2. Protobuf 核心概念

#### 2.1 字段编号（Field Number）

```protobuf
message User {
  uint64 id = 1;        // 字段编号：1
  string email = 2;     // 字段编号：2
  string nickname = 3;  // 字段编号：3
}
```

**为什么需要字段编号？**

1. **版本兼容**：字段编号不能改变，保证前后兼容
2. **二进制序列化**：字段编号用于识别字段，不传输字段名
3. **性能优化**：1-15编号只占1字节，16-2047占2字节

**版本演进示例**：

```protobuf
// v1版本
message User {
  uint64 id = 1;
  string email = 2;
}

// v2版本（向下兼容）
message User {
  uint64 id = 1;
  string email = 2;
  string nickname = 3;     // 新增字段，旧客户端会忽略
  string avatar_url = 4;   // 新增字段
}
```

**❌ 错误示例**：

```protobuf
// v1
message User {
  uint64 id = 1;
  string email = 2;
}

// v2（错误：修改了字段编号）
message User {
  uint64 id = 2;     // ❌ 不能修改已有字段的编号
  string email = 1;  // ❌ 会导致数据错乱
}
```

---

#### 2.2 数据类型映射

**Protobuf → Go 类型映射**：

| Protobuf 类型 | Go 类型 | 说明 |
|--------------|---------|------|
| `int32` | `int32` | 32位整数 |
| `int64` | `int64` | 64位整数 |
| `uint32` | `uint32` | 无符号32位 |
| `uint64` | `uint64` | 无符号64位 |
| `string` | `string` | UTF-8字符串 |
| `bool` | `bool` | 布尔值 |
| `bytes` | `[]byte` | 二进制数据 |
| `repeated` | `[]T` | 数组/切片 |

**教学示例**：

```protobuf
message CreateOrderRequest {
  uint64 user_id = 1;              // → uint64
  repeated OrderItem items = 2;    // → []OrderItem
}

message OrderItem {
  uint64 book_id = 1;              // → uint64
  int32 quantity = 2;              // → int32
}
```

**生成的 Go 代码**：

```go
type CreateOrderRequest struct {
    UserId uint64       `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
    Items  []*OrderItem `protobuf:"bytes,2,rep,name=items,proto3" json:"items,omitempty"`
}

type OrderItem struct {
    BookId   uint64 `protobuf:"varint,1,opt,name=book_id,json=bookId,proto3" json:"book_id,omitempty"`
    Quantity int32  `protobuf:"varint,2,opt,name=quantity,proto3" json:"quantity,omitempty"`
}
```

---

#### 2.3 服务定义（Service）

```protobuf
service UserService {
  // RPC方法定义
  rpc Register(RegisterRequest) returns (RegisterResponse);
  
  // 流式RPC（本项目暂不使用）
  // rpc StreamUsers(stream UserRequest) returns (stream UserResponse);
}
```

**生成的 Go 接口**：

```go
// 服务端需要实现的接口
type UserServiceServer interface {
    Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
    mustEmbedUnimplementedUserServiceServer()
}

// 客户端调用的接口
type UserServiceClient interface {
    Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
}
```

---

### 3. 接口设计规范

#### 3.1 命名规范

```protobuf
// ✅ 正确：服务名使用领域名 + Service
service UserService { }        // ✅
service CatalogService { }     // ✅

// ❌ 错误：
service BookService { }        // ❌ 应该用 CatalogService（更明确）
service UserAPI { }            // ❌ 不要用 API 后缀

// ✅ 正确：方法名使用动词 + 名词
rpc Register(RegisterRequest) returns (RegisterResponse);
rpc GetUser(GetUserRequest) returns (GetUserResponse);
rpc ListBooks(ListBooksRequest) returns (ListBooksResponse);

// ❌ 错误：
rpc User(UserRequest) returns (UserResponse);  // ❌ 缺少动词
rpc List(Request) returns (Response);          // ❌ 太模糊
```

---

#### 3.2 请求/响应消息设计

**统一响应格式**：

```protobuf
message RegisterResponse {
  uint32 code = 1;        // 状态码：0成功，非0失败
  string message = 2;     // 提示信息
  uint64 user_id = 3;     // 业务数据
  string token = 4;
}
```

**为什么这样设计？**

1. **code + message**：兼容 Phase 1 的 HTTP API 格式，方便迁移
2. **业务数据**：user_id、token 等放在同一层级
3. **扩展性**：可以添加更多字段而不破坏兼容性

**对比 gRPC 原生错误处理**：

```go
// gRPC 原生方式（仅适合简单错误）
return nil, status.Errorf(codes.NotFound, "user not found")

// 本项目方式（更灵活）
return &pb.RegisterResponse{
    Code:    1001,
    Message: "邮箱已被注册",
}, nil  // gRPC层面返回nil，业务错误放在响应体
```

---

#### 3.3 分页查询设计

```protobuf
// 请求
message ListBooksRequest {
  uint32 page = 1;        // 页码（从1开始）
  uint32 page_size = 2;   // 每页数量（默认10，最大100）
  string sort_by = 3;     // 排序字段：created_at, price
  string order = 4;       // 排序方向：desc, asc
}

// 响应
message ListBooksResponse {
  uint32 code = 1;
  string message = 2;
  repeated Book books = 3;   // 数据列表
  uint32 total = 4;          // 总数
  uint32 page = 5;           // 当前页
  uint32 page_size = 6;      // 每页数量
}
```

**教学重点**：

1. **page 从 1 开始**：符合用户习惯（而不是从0开始）
2. **total 字段**：前端需要计算总页数
3. **repeated**：Protobuf 的数组类型

---

## 📂 项目结构

### Protobuf 目录结构

```
proto/
├── user/v1/
│   ├── user.proto           # 接口定义（手写）
│   ├── user.pb.go           # 消息代码（自动生成）
│   └── user_grpc.pb.go      # gRPC代码（自动生成）
├── catalog/v1/
│   ├── catalog.proto
│   ├── catalog.pb.go
│   └── catalog_grpc.pb.go
├── inventory/v1/
│   ├── inventory.proto
│   ├── inventory.pb.go
│   └── inventory_grpc.pb.go
├── order/v1/
│   ├── order.proto
│   ├── order.pb.go
│   └── order_grpc.pb.go
└── payment/v1/
    ├── payment.proto
    ├── payment.pb.go
    └── payment_grpc.pb.go
```

**为什么使用 v1 目录？**

1. **版本管理**：未来可以添加 v2、v3
2. **兼容性**：旧客户端继续使用 v1，新客户端使用 v2
3. **平滑迁移**：两个版本可以并存

---

## 🔨 生成的代码统计

### 代码行数

```bash
$ find proto -name "*.pb.go" -o -name "*_grpc.pb.go" | xargs wc -l
   12263  proto/user/v1/user_grpc.pb.go
   24837  proto/user/v1/user.pb.go
   # ... 其他文件
```

### 生成的文件

| 服务 | .proto文件 | .pb.go（消息） | _grpc.pb.go（RPC） |
|------|-----------|---------------|-------------------|
| user-service | user.proto (106行) | user.pb.go (~25KB) | user_grpc.pb.go (~12KB) |
| catalog-service | catalog.proto (124行) | catalog.pb.go (~28KB) | catalog_grpc.pb.go (~14KB) |
| inventory-service | inventory.proto (132行) | inventory.pb.go (~30KB) | inventory_grpc.pb.go (~16KB) |
| order-service | order.proto (118行) | order.pb.go (~26KB) | order_grpc.pb.go (~13KB) |
| payment-service | payment.proto (78行) | payment.pb.go (~18KB) | payment_grpc.pb.go (~9KB) |

**教学重点**：

1. **不要手动修改 .pb.go 文件**：每次生成都会覆盖
2. **只修改 .proto 文件**：然后重新生成
3. **提交 .pb.go 到 Git**：避免团队成员 protoc 版本不一致

---

## 🛠️ 工具链

### 1. protoc 编译器

**安装**：

```bash
# Debian/Ubuntu
sudo apt-get install protobuf-compiler

# macOS
brew install protobuf

# 验证
protoc --version
# libprotoc 3.21.12
```

---

### 2. Go 插件

**安装**：

```bash
# protoc-gen-go（生成消息代码）
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# protoc-gen-go-grpc（生成gRPC代码）
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 验证
ls ~/go/bin/ | grep protoc
# protoc-gen-go
# protoc-gen-go-grpc
```

**为什么需要两个插件？**

1. **protoc-gen-go**：生成 Protobuf 消息的序列化/反序列化代码
2. **protoc-gen-go-grpc**：生成 gRPC 服务端/客户端接口

---

### 3. Makefile 命令

**新增命令**：

```bash
# 生成所有 Protobuf 代码
make proto-gen

# 清理生成的代码
make proto-clean

# 检查 Protobuf 定义（提示安装 buf）
make proto-lint
```

**生成命令详解**：

```bash
protoc \
  --go_out=. \                      # 生成 .pb.go 文件到当前目录
  --go_opt=paths=source_relative \  # 使用相对路径
  --go-grpc_out=. \                 # 生成 _grpc.pb.go 文件
  --go-grpc_opt=paths=source_relative \
  proto/user/v1/user.proto
```

**参数说明**：

- `--go_out=.`：输出目录为当前目录
- `paths=source_relative`：生成的文件和 .proto 文件在同一目录
- `--go-grpc_out=.`：生成 gRPC 代码的输出目录

---

## 📝 5个服务的接口总览

### 1. user-service

**端口**：9001

**RPC方法**：

| 方法 | 说明 | 调用方 |
|------|------|--------|
| `Register` | 用户注册 | api-gateway |
| `Login` | 用户登录 | api-gateway |
| `ValidateToken` | 验证Token | order-service等 |
| `GetUser` | 获取用户信息 | order-service等 |
| `RefreshToken` | 刷新Token | api-gateway |

**教学重点**：

- Token验证是跨服务调用的典型场景
- order-service 需要验证用户身份时调用 `ValidateToken`

---

### 2. catalog-service

**端口**：9002

**RPC方法**：

| 方法 | 说明 | 调用方 |
|------|------|--------|
| `GetBook` | 获取图书详情 | api-gateway |
| `ListBooks` | 图书列表（分页） | api-gateway |
| `SearchBooks` | 搜索图书 | api-gateway |
| `PublishBook` | 发布图书 | api-gateway |
| `BatchGetBooks` | 批量获取图书 | order-service |

**教学重点**：

- `BatchGetBooks` 是内部接口，用于订单创建时获取图书价格
- 读写分离：catalog-service 只负责图书信息，不管理库存

---

### 3. inventory-service

**端口**：9004

**RPC方法**：

| 方法 | 说明 | 调用方 |
|------|------|--------|
| `GetStock` | 查询库存 | api-gateway |
| `BatchGetStock` | 批量查询库存 | api-gateway |
| `DeductStock` | 扣减库存 | order-service |
| `ReleaseStock` | 释放库存 | order-service |
| `RestockInventory` | 补充库存 | api-gateway |
| `GetInventoryLogs` | 库存变更日志 | api-gateway |

**教学重点**：

- `DeductStock` 和 `ReleaseStock` 是 Saga 事务的核心操作
- 库存变更日志用于审计和对账

---

### 4. order-service

**端口**：9003

**RPC方法**：

| 方法 | 说明 | 调用方 |
|------|------|--------|
| `CreateOrder` | 创建订单 | api-gateway |
| `UpdateOrderStatus` | 更新订单状态 | 内部调用 |
| `GetOrder` | 查询订单详情 | api-gateway |
| `ListUserOrders` | 用户订单列表 | api-gateway |
| `CancelOrder` | 取消订单 | api-gateway |

**教学重点**：

- `CreateOrder` 是 Saga 编排的入口，会调用多个服务
- 订单状态机：PENDING → PAID → SHIPPED → COMPLETED

---

### 5. payment-service

**端口**：9005

**RPC方法**：

| 方法 | 说明 | 调用方 |
|------|------|--------|
| `Pay` | 创建支付 | order-service |
| `GetPaymentStatus` | 查询支付状态 | order-service |
| `Refund` | 退款 | order-service |

**教学重点**：

- Phase 2 使用 Mock 实现（70%成功率）
- Phase 3 可以对接真实支付接口（支付宝、微信）

---

## 🎓 教学对比：Phase 1 vs Phase 2

### 接口定义方式

**Phase 1: HTTP + JSON**

```go
// handler/user_handler.go
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6"`
    Nickname string `json:"nickname"`
}

// 手动验证
func (h *UserHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"code": 1, "message": err.Error()})
        return
    }
    // ...
}
```

**Phase 2: Protobuf + gRPC**

```protobuf
// proto/user/v1/user.proto
message RegisterRequest {
  string email = 1;
  string password = 2;
  string nickname = 3;
}

service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
}
```

```go
// 自动生成的代码
func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // req.Email 已经是强类型，不需要手动解析
    // ...
}
```

**对比总结**：

| 特性 | Phase 1 (HTTP/JSON) | Phase 2 (Protobuf/gRPC) |
|------|---------------------|------------------------|
| 接口定义 | Go结构体 | .proto文件 |
| 验证 | 手动（binding tag） | 自动（强类型） |
| 序列化 | JSON（慢） | Protobuf（快） |
| 跨语言 | 需要手动定义 | 自动生成 |
| 版本兼容 | 手动维护 | 字段编号自动兼容 |

---

### 服务调用方式

**Phase 1: HTTP 调用**

```go
// 在monolith中直接调用
userService := application.NewUserService(userRepo, jwtManager)
user, err := userService.Register(ctx, email, password, nickname)
```

**Phase 2: gRPC 调用**

```go
// order-service 调用 user-service
conn, _ := grpc.Dial("user-service:9001", grpc.WithInsecure())
client := userv1.NewUserServiceClient(conn)

resp, err := client.ValidateToken(ctx, &userv1.ValidateTokenRequest{
    Token: token,
})
if err != nil || !resp.Valid {
    return errors.New("invalid token")
}
```

**对比总结**：

- Phase 1：进程内调用（快，但不能独立部署）
- Phase 2：网络调用（慢一点，但可以独立部署和扩展）

---

## 🚀 下一步：Day 24-25

### 任务概览

**目标**：实现第一个微服务 `user-service`

**步骤**：

1. **创建服务目录结构**
   ```
   services/user-service/
   ├── cmd/
   │   └── main.go              # gRPC服务器入口
   ├── internal/
   │   ├── domain/              # 复用Phase 1的domain层
   │   ├── application/         # 复用Phase 1的application层
   │   ├── infrastructure/      # 复用Phase 1的infrastructure层
   │   └── grpc/
   │       └── handler/
   │           └── user_handler.go  # gRPC Handler实现
   └── config/
       └── config.yaml
   ```

2. **实现 gRPC Handler**
   ```go
   type UserServiceServer struct {
       pb.UnimplementedUserServiceServer
       userService *application.UserService
   }
   
   func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
       // 调用 Phase 1 的 application.UserService
       user, token, err := s.userService.Register(ctx, req.Email, req.Password, req.Nickname)
       // ...
   }
   ```

3. **启动 gRPC 服务器**
   ```go
   lis, _ := net.Listen("tcp", ":9001")
   grpcServer := grpc.NewServer()
   pb.RegisterUserServiceServer(grpcServer, &UserServiceServer{})
   grpcServer.Serve(lis)
   ```

4. **测试**
   - 使用 `grpcurl` 测试接口
   - 编写集成测试

---

## 📊 Day 23 成果总结

### 文件清单

| 文件 | 行数 | 说明 |
|------|------|------|
| `proto/user/v1/user.proto` | 106 | 用户服务接口定义 |
| `proto/catalog/v1/catalog.proto` | 124 | 图书目录服务接口定义 |
| `proto/inventory/v1/inventory.proto` | 132 | 库存服务接口定义 |
| `proto/order/v1/order.proto` | 118 | 订单服务接口定义 |
| `proto/payment/v1/payment.proto` | 78 | 支付服务接口定义 |
| **生成的代码** | ~200KB | 10个 .pb.go 文件 |
| `Makefile` | +55 | 新增3个proto命令 |

### 知识点

1. **Protobuf 基础**
   - 字段编号和版本兼容
   - 数据类型映射
   - 服务定义

2. **gRPC 代码生成**
   - protoc 编译器
   - Go 插件
   - 生成的接口和实现

3. **接口设计规范**
   - 命名规范
   - 请求/响应消息设计
   - 分页查询设计

4. **工具链**
   - protoc
   - protoc-gen-go
   - protoc-gen-go-grpc
   - Makefile 自动化

---

## 🎯 学习检查清单

完成 Day 23 后，你应该能够回答：

- [ ] Protobuf 相比 JSON 有哪些优势？
- [ ] 字段编号的作用是什么？为什么不能修改？
- [ ] `repeated` 关键字对应 Go 的什么类型？
- [ ] `.pb.go` 和 `_grpc.pb.go` 有什么区别？
- [ ] 为什么需要两个 protoc 插件？
- [ ] 如何保证 Protobuf 的版本兼容性？
- [ ] gRPC 的 `service` 定义生成了哪些 Go 接口？

---

**教学要点**：

1. **Protobuf 是强类型接口定义**：编译期检查，避免运行时错误
2. **一次定义，多处使用**：服务端、客户端、多种语言都用同一份 proto
3. **性能优势**：二进制序列化比 JSON 快得多
4. **版本兼容**：字段编号保证前后兼容，便于系统演进

**下一步**：开始实现第一个微服务 `user-service`！
