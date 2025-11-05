# Week 2 Day 12-14: 订单模块实现完成报告

## 📋 任务概述

本阶段实现了电商系统的核心功能：**订单模块**，重点解决了分布式系统中的经典问题：**并发场景下的库存超卖问题**。

## ✅ 完成内容

### 1. 领域层实现

#### 1.1 订单实体 (`internal/domain/order/entity.go`)
```go
// 核心设计点
- Order聚合根：管理订单状态和订单项
- OrderItem子实体：一对多关系
- OrderStatus状态机：防止非法状态转换
- 价格快照机制：OrderItem存储下单时价格，防止商家改价影响历史订单
```

**关键代码**:
```go
// 状态机验证
func (o *Order) CanTransitionTo(target OrderStatus) bool {
    transitions := map[OrderStatus][]OrderStatus{
        OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
        OrderStatusPaid:      {OrderStatusShipped, OrderStatusCancelled},
        OrderStatusShipped:   {OrderStatusCompleted},
        OrderStatusCompleted: {},
        OrderStatusCancelled: {},
    }
    allowedStates, exists := transitions[o.Status]
    if !exists {
        return false
    }
    for _, allowed := range allowedStates {
        if allowed == target {
            return true
        }
    }
    return false
}
```

**教学价值**:
- 状态机模式：防止订单从"已完成"变成"待支付"等非法操作
- 聚合根设计：Order管理所有OrderItem，保证数据一致性
- 价格存储：使用int64存储分（cent），避免float64精度问题

#### 1.2 订单号生成器 (`internal/domain/order/order_no.go`)
```go
// 生成规则: ORD + 时间戳(秒) + 6位随机数
// 示例: ORD1699248000123456
```

**教学说明**: 生产环境应使用Snowflake算法，保证分布式系统下ID全局唯一且有序。

---

### 2. 基础设施层实现

#### 2.1 订单仓储 (`internal/infrastructure/persistence/mysql/order_repo.go`)
**核心功能**:
- 支持一对多关系存储（Order → OrderItems）
- 使用GORM的Preload避免N+1查询问题
- 上下文感知：支持事务传播

**关键实现**:
```go
func (r *orderRepository) FindByID(ctx context.Context, id uint) (*order.Order, error) {
    db := r.getDB(ctx) // 从context获取事务DB
    err := db.Preload("Items").First(&model, id).Error // Preload避免N+1查询
    // ...
}
```

**教学价值**:
- Preload预加载：一次查询加载关联数据，性能优化重要手段
- 事务传播：通过context.Value传递事务对象

#### 2.2 事务管理器 (`internal/infrastructure/persistence/mysql/tx_manager.go`)
**设计目标**: 封装事务逻辑，统一管理事务生命周期

**核心代码**:
```go
func (m *TxManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
    return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 将事务DB注入context
        txCtx := context.WithValue(ctx, "tx", tx)
        return fn(txCtx)
    })
}
```

**教学价值**:
- 事务封装：简化业务代码，避免手动Begin/Commit/Rollback
- Context传播：所有Repository方法通过getDB(ctx)感知事务
- ACID保证：失败自动回滚，成功自动提交

#### 2.3 修复图书仓储 (`internal/infrastructure/persistence/mysql/book_repo.go:121, 133`)
**问题**: LockByID和UpdateStock方法直接使用`r.db`，无法参与事务

**修复**: 改用`r.getDB(ctx)`，支持事务传播
```go
// 修复前
func (r *bookRepository) LockByID(ctx context.Context, id uint) (*book.Book, error) {
    err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, id).Error
}

// 修复后
func (r *bookRepository) LockByID(ctx context.Context, id uint) (*book.Book, error) {
    db := r.getDB(ctx) // 支持事务传播
    err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, id).Error
}
```

---

### 3. 应用层实现

#### 3.1 下单用例 (`internal/application/order/create_order.go`)
**最核心文件**：实现防超卖逻辑

**完整流程**:
```
1. 开启数据库事务
2. 对每个商品执行 SELECT FOR UPDATE 锁定库存行（悲观锁）
3. 校验库存是否充足
4. 使用锁定时的价格计算订单总额（防止改价攻击）
5. 创建订单和订单明细
6. 扣减库存
7. 提交事务
```

**关键代码**:
```go
err := uc.txManager.Transaction(ctx, func(txCtx context.Context) error {
    for _, item := range req.Items {
        // 步骤1: 悲观锁锁定库存
        b, err := uc.bookRepo.LockByID(txCtx, item.BookID)
        // SELECT * FROM books WHERE id = ? FOR UPDATE
        // FOR UPDATE会对行加排他锁，其他事务必须等待
        
        // 步骤2: 校验库存
        if b.Stock < item.Quantity {
            return order.ErrInsufficientStock
        }
        
        // 步骤3: 使用锁定时的价格
        totalCents += b.Price * int64(item.Quantity)
        
        // 步骤4: 创建订单项（价格快照）
        orderItems = append(orderItems, &order.OrderItem{
            BookID:   b.ID,
            Quantity: item.Quantity,
            Price:    b.Price, // 存储当前价格
        })
    }
    
    // 步骤5: 创建订单
    newOrder := order.NewOrder(req.UserID, totalCents, orderItems)
    err := uc.orderRepo.Create(txCtx, newOrder)
    
    // 步骤6: 扣库存
    for _, item := range req.Items {
        err := uc.bookRepo.UpdateStock(txCtx, item.BookID, -item.Quantity)
    }
    
    return nil
})
```

**教学要点**:

**为什么需要SELECT FOR UPDATE？**
```
场景：库存剩余1本，用户A和用户B同时下单

没有锁的情况：
┌─────────┬────────────────────────────┬────────────────────────────┐
│ 时间    │ 用户A事务                  │ 用户B事务                  │
├─────────┼────────────────────────────┼────────────────────────────┤
│ T1      │ SELECT stock (得到1)       │                            │
│ T2      │                            │ SELECT stock (得到1)       │
│ T3      │ IF stock >= 1 ✓            │                            │
│ T4      │                            │ IF stock >= 1 ✓            │
│ T5      │ UPDATE stock = 0           │                            │
│ T6      │                            │ UPDATE stock = -1 ❌超卖！ │
│ T7      │ COMMIT                     │                            │
│ T8      │                            │ COMMIT                     │
└─────────┴────────────────────────────┴────────────────────────────┘

使用SELECT FOR UPDATE：
┌─────────┬────────────────────────────┬────────────────────────────┐
│ 时间    │ 用户A事务                  │ 用户B事务                  │
├─────────┼────────────────────────────┼────────────────────────────┤
│ T1      │ SELECT FOR UPDATE (锁定)   │                            │
│ T2      │ stock = 1 ✓                │ SELECT FOR UPDATE (等待)   │
│ T3      │ UPDATE stock = 0           │ (等待中...)                │
│ T4      │ COMMIT (释放锁)            │ (等待中...)                │
│ T5      │                            │ stock = 0 ✗ 库存不足      │
│ T6      │                            │ ROLLBACK                   │
└─────────┴────────────────────────────┴────────────────────────────┘
```

**悲观锁 vs 乐观锁对比**:

| 特性         | 悲观锁 (SELECT FOR UPDATE)     | 乐观锁 (Version字段)          |
|--------------|--------------------------------|-------------------------------|
| 并发性能     | 较低（串行化）                 | 较高（允许并发读）            |
| 实现复杂度   | 简单                           | 中等（需要重试逻辑）          |
| 适用场景     | 库存扣减、抢购                 | 更新用户资料                  |
| 冲突频率     | 高冲突场景                     | 低冲突场景                    |
| 是否超卖     | 不会                           | 不会（需正确处理version）     |

本项目选择悲观锁的原因：
1. 秒杀场景冲突频率高，乐观锁会导致大量重试
2. 教学目的明确，便于理解锁机制
3. 逻辑简单，不需要复杂的重试逻辑

---

### 4. 接口层实现

#### 4.1 HTTP处理器 (`internal/interface/http/handler/order.go`)
```go
func (h *OrderHandler) CreateOrder(c *gin.Context) {
    // 1. 获取当前登录用户ID
    userID := middleware.GetUserID(c)
    
    // 2. 绑定请求参数
    var req dto.CreateOrderRequest
    c.ShouldBindJSON(&req)
    
    // 3. 调用用例
    result := h.createOrderUC.Execute(c.Request.Context(), &order.CreateOrderRequest{
        UserID: userID,
        Items:  items,
    })
    
    // 4. 返回响应
    response.Success(c, dto.CreateOrderResponse{...})
}
```

#### 4.2 DTO定义 (`internal/interface/http/dto/book.go`)
```go
type CreateOrderRequest struct {
    Items []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

type CreateOrderItemRequest struct {
    BookID   uint `json:"book_id" binding:"required"`
    Quantity int  `json:"quantity" binding:"required,min=1,max=999"`
}
```

---

### 5. 主程序集成 (`cmd/api/main.go`)

**依赖注入链**:
```
基础设施层 → 领域层 → 应用层 → 接口层

orderRepo := mysql.NewOrderRepository(db)
txManager := mysql.NewTxManager(db)
createOrderUseCase := apporder.NewCreateOrderUseCase(orderRepo, bookRepo, txManager)
orderHandler := handler.NewOrderHandler(createOrderUseCase)
```

**路由注册**:
```go
orders := v1.Group("/orders")
orders.Use(authMiddleware.RequireAuth()) // 需要登录
{
    orders.POST("", orderHandler.CreateOrder)
}
```

---

## 🧪 集成测试

测试文件: `test/integration/order_integration.go`

### 测试场景1: 正常下单流程
```
✓ 用户注册登录成功
✓ 上架图书（库存10本，单价89元）
✓ 购买3本
  - 订单创建成功
  - 订单金额: 267.00元 (3 × 89 = 267)
  - 剩余库存: 7本
```

### 测试场景2: 库存不足场景
```
✓ 尝试购买8本（剩余7本）
✓ 系统正确返回"库存不足"错误
✓ 订单未创建，库存未扣减
```

### 测试场景3: 并发防超卖测试（核心）
```
场景: 10个用户同时抢购剩余7本

测试结果:
  - 成功下单: 7个
  - 失败下单: 3个（库存不足）

✓ 防超卖机制测试通过！
✓ 成功订单数 = 剩余库存数
✓ SELECT FOR UPDATE悲观锁有效防止了超卖
```

**并发测试代码**:
```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        // 每个goroutine购买1本
        postJSON(baseURL+"/orders", orderReq, token)
    }()
}
wg.Wait()
// 统计结果: 成功7个，失败3个
```

---

## 📊 数据库表设计

### orders表
```sql
CREATE TABLE `orders` (
  `id` bigint unsigned AUTO_INCREMENT,
  `order_no` varchar(32) NOT NULL COMMENT '订单号',
  `user_id` bigint unsigned NOT NULL COMMENT '买家用户ID',
  `total` bigint NOT NULL COMMENT '订单总金额(分)',
  `status` tinyint DEFAULT 1 COMMENT '订单状态',
  `created_at` datetime(3) NULL,
  `updated_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_orders_order_no` (`order_no`),
  INDEX `idx_orders_user_id` (`user_id`),
  INDEX `idx_orders_status` (`status`)
)
```

### order_items表
```sql
CREATE TABLE `order_items` (
  `id` bigint unsigned AUTO_INCREMENT,
  `order_id` bigint unsigned NOT NULL COMMENT '订单ID',
  `book_id` bigint unsigned NOT NULL COMMENT '图书ID',
  `quantity` bigint NOT NULL COMMENT '购买数量',
  `price` bigint NOT NULL COMMENT '下单时单价(分)',
  PRIMARY KEY (`id`),
  INDEX `idx_order_items_order_id` (`order_id`),
  INDEX `idx_order_items_book_id` (`book_id`),
  CONSTRAINT `fk_orders_items` FOREIGN KEY (`order_id`) REFERENCES `orders`(`id`)
)
```

**索引设计说明**:
- `order_no`唯一索引：快速查询订单
- `user_id`索引：查询用户订单列表
- `status`索引：查询待支付/已完成订单
- `order_id`外键索引：关联查询

---

## 🎓 核心教学要点总结

### 1. 分布式系统核心问题：防超卖
**问题本质**: 并发事务的竞态条件（Race Condition）

**解决方案对比**:
| 方案               | 优点                     | 缺点                     | 适用场景           |
|--------------------|--------------------------|--------------------------|-------------------|
| 悲观锁(FOR UPDATE) | 简单、绝对不会超卖       | 性能较低、串行化         | 秒杀、库存扣减    |
| 乐观锁(Version)    | 并发高、无锁等待         | 需要重试逻辑             | 低冲突场景        |
| Redis分布式锁      | 跨服务、高性能           | 实现复杂、需Redis        | 分布式系统        |
| 消息队列           | 削峰填谷、异步处理       | 延迟高、复杂度高         | 大促场景          |

### 2. 事务管理
**ACID特性**:
- Atomicity（原子性）：锁库存→创建订单→扣库存，要么全成功要么全失败
- Consistency（一致性）：库存 + 订单数量 = 总量（守恒）
- Isolation（隔离性）：SELECT FOR UPDATE保证事务间隔离
- Durability（持久性）：COMMIT后数据持久化

**事务传播机制**:
```go
// 通过context传递事务对象
txCtx := context.WithValue(ctx, "tx", tx)

// Repository感知事务
func (r *repo) getDB(ctx context.Context) *gorm.DB {
    if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
        return tx // 返回事务DB
    }
    return r.db // 返回普通DB
}
```

### 3. 领域驱动设计(DDD)
**聚合根**: Order管理OrderItem，保证订单一致性
**值对象**: OrderStatus状态机
**仓储模式**: 隔离持久化细节

### 4. 架构分层
```
接口层 (handler)     ← HTTP请求
应用层 (use case)    ← 业务流程编排
领域层 (domain)      ← 核心业务逻辑
基础设施层 (repo)    ← 数据持久化
```

### 5. 性能优化
- **N+1查询问题**: 使用Preload预加载关联数据
- **索引设计**: order_no唯一索引、user_id索引、status索引
- **价格存储**: 使用int64分存储，避免float精度问题

---

## 📁 新增/修改文件清单

### 新增文件（10个）
```
internal/domain/order/
├── entity.go              # 订单实体和状态机
├── errors.go              # 订单领域错误
├── order_no.go            # 订单号生成器
└── repository.go          # 订单仓储接口

internal/infrastructure/persistence/mysql/
├── order_repo.go          # 订单仓储实现
└── tx_manager.go          # 事务管理器

internal/application/order/
└── create_order.go        # 下单用例（核心）

internal/interface/http/handler/
└── order.go               # 订单HTTP处理器

test/integration/
├── order_integration.go   # 集成测试程序
└── order_test.sh          # 测试脚本（备用）
```

### 修改文件（3个）
```
cmd/api/main.go:64,75,87,101,138
  - 新增订单模块依赖注入
  - 注册订单路由

internal/infrastructure/persistence/mysql/db.go:38-77
  - 新增OrderModel和OrderItemModel
  - 更新AutoMigrate

internal/infrastructure/persistence/mysql/book_repo.go:121,133
  - 修复LockByID使用getDB(ctx)支持事务
  - 修复UpdateStock使用getDB(ctx)支持事务

internal/interface/http/dto/book.go:60-82
  - 新增CreateOrderRequest
  - 新增CreateOrderResponse
```

---

## 🚀 下一步计划

根据ROADMAP.md，接下来的任务：

**Week 2 Day 15-17: 支付模块**
- 集成第三方支付（支付宝沙箱）
- 实现支付回调处理
- 订单状态更新流程

**Week 3: 代码优化**
- 引入Wire依赖注入
- 添加单元测试
- 性能优化和监控

---

## 💡 经验总结

### 成功经验
1. **教学优先**: 代码包含详细注释，解释"为什么"而不仅仅是"怎么做"
2. **测试驱动**: 集成测试覆盖正常、异常、并发三大场景
3. **渐进式实现**: 先实现功能，再优化性能，符合教学节奏

### 遇到的问题及解决
1. **问题**: 事务不生效导致并发超卖
   **原因**: book_repo.go的LockByID和UpdateStock直接使用r.db
   **解决**: 改用r.getDB(ctx)支持事务传播

2. **问题**: 泛型方法编译失败
   **原因**: Go 1.21不支持泛型方法
   **解决**: 使用闭包捕获返回值

3. **问题**: ISBN格式验证失败
   **原因**: 生成的ISBN不是13位
   **解决**: 使用fmt.Sprintf("9787115428%03d", timestamp%1000)

---

## 📚 参考资料

- [GORM事务文档](https://gorm.io/docs/transactions.html)
- [MySQL SELECT FOR UPDATE](https://dev.mysql.com/doc/refman/8.0/en/innodb-locking-reads.html)
- [领域驱动设计](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- 项目内部文档: TEACHING.md, ROADMAP.md

---

**报告生成时间**: 2025-11-05  
**实现周期**: Week 2 Day 12-14  
**代码行数**: 约1200行  
**测试覆盖**: 3个核心场景（正常/异常/并发）  
**测试结果**: ✅ 全部通过
