# Phase 2 - Week 6 启动计划

> **时间范围**：Day 29-35  
> **核心目标**：完成所有微服务拆分，建立完整的微服务体系  
> **依赖前提**：Week 5已完成（user-service + api-gateway运行正常）

---

## 📋 Week 6 总览

### 本周目标

本周将完成剩余的4个微服务拆分：

1. **catalog-service**（图书服务）- Day 29-30
2. **inventory-service**（库存服务）- Day 29-30
3. **order-service**（订单服务）- Day 31-32
4. **payment-service**（支付服务）- Day 33-34
5. **服务发现**（Consul集成）- Day 35

完成后，整个微服务架构将包含：
```
api-gateway (8080)
    ↓
├─→ user-service (9001) ✅ 已完成
├─→ catalog-service (9002) ← 本周
├─→ order-service (9003) ← 本周
├─→ inventory-service (9004) ← 本周
└─→ payment-service (9005) ← 本周
```

### 教学重点

根据TEACHING.md的核心原则，本周的教学重点：

1. **渐进式实现**：从简单服务（catalog）到复杂服务（order）
2. **服务间通信**：order-service如何调用多个下游服务
3. **高并发场景**：inventory-service的库存扣减（Redis + Lua）
4. **分布式基础**：为Week 7的Saga事务打好基础
5. **服务发现**：从硬编码地址到动态服务发现

---

## 📅 详细任务拆解

### Day 29-30: catalog-service + inventory-service

#### **1. catalog-service（图书服务）**

**目标**：将Phase 1的图书查询功能拆分为独立微服务

**架构设计**：
```
catalog-service/
├── cmd/main.go                    # gRPC服务器
├── internal/
│   ├── grpc/handler/
│   │   └── catalog_handler.go     # 实现5个RPC方法
│   ├── domain/                    # 复用Phase 1
│   │   └── book/
│   ├── application/               # 复用Phase 1
│   │   └── book/
│   └── infrastructure/
│       ├── persistence/mysql/     # 图书仓储
│       └── cache/redis/           # 列表缓存
├── config/config.yaml
└── go.mod
```

**实现清单**：

- [ ] **RPC方法实现**（参考proto/catalog/v1/catalog.proto）
  - ListBooks：分页查询（支持排序、关键词搜索）
  - GetBook：获取图书详情
  - SearchBooks：全文搜索（Phase 2暂用LIKE，Week 7引入ES）
  - PublishBook：发布图书（需用户认证）
  - UpdateStock：更新库存（供inventory-service调用）

- [ ] **缓存策略**
  ```go
  // 教学要点：
  // 1. 列表查询结果缓存5分钟（热点数据）
  // 2. 图书详情缓存1小时（变化少）
  // 3. 缓存Key设计：catalog:list:{page}:{sort} 或 catalog:book:{id}
  // 4. 缓存失效策略：更新/删除图书时主动清除
  
  // DO（正确做法）：
  func (s *catalogService) ListBooks(ctx context.Context, req *ListBooksRequest) (*ListBooksResponse, error) {
      cacheKey := fmt.Sprintf("catalog:list:%d:%s", req.Page, req.Sort)
      
      // 1. 尝试从缓存获取
      if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
          return unmarshalResponse(cached), nil
      }
      
      // 2. 查询数据库
      books, err := s.repo.List(ctx, req)
      if err != nil {
          return nil, err
      }
      
      // 3. 写入缓存
      s.cache.Set(ctx, cacheKey, books, 5*time.Minute)
      
      return books, nil
  }
  
  // DON'T（错误做法）：
  // ❌ 缓存时间过长（1天），导致数据不一致
  // ❌ 不设置过期时间，内存泄漏
  // ❌ 缓存Key不包含查询参数，导致脏数据
  ```

- [ ] **数据库迁移**
  - 创建catalog_db数据库
  - 从bookstore.books表导入数据
  - 添加索引（title、price、created_at）

- [ ] **测试验证**
  - grpcurl测试5个RPC方法
  - 缓存命中率验证（Redis MONITOR）
  - 性能测试（QPS>1000）

---

#### **2. inventory-service（库存服务）**

**目标**：实现高并发库存扣减，为订单创建做准备

**架构设计**：
```
inventory-service/
├── cmd/main.go
├── internal/
│   ├── grpc/handler/
│   │   └── inventory_handler.go   # 6个RPC方法
│   ├── domain/
│   │   └── inventory/
│   │       ├── entity.go          # 库存实体、库存日志
│   │       ├── repository.go
│   │       └── service.go         # 核心业务逻辑
│   └── infrastructure/
│       ├── persistence/
│       │   ├── mysql/             # 库存数据持久化
│       │   └── redis/             # 库存缓存 + Lua脚本
│       └── scripts/
│           └── decrStock.lua      # 原子扣减脚本
└── config/config.yaml
```

**实现清单**：

- [ ] **RPC方法实现**
  - GetStock：查询库存
  - LockStock：锁定库存（订单创建时调用）
  - ReleaseStock：释放库存（订单取消时调用）
  - DecrStock：扣减库存（支付成功时调用）
  - IncrStock：增加库存（退货时调用）
  - GetStockLog：查询库存日志

- [ ] **核心难点：高并发库存扣减（防超卖）**

  ```go
  // 教学要点：高并发场景下的库存扣减方案对比
  
  // ❌ 方案1：无锁扣减（错误！会超卖）
  func DecrStockWrong(bookID uint, quantity int) error {
      stock := db.Query("SELECT stock FROM inventory WHERE book_id = ?", bookID)
      if stock < quantity {
          return ErrInsufficientStock
      }
      db.Exec("UPDATE inventory SET stock = stock - ? WHERE book_id = ?", quantity, bookID)
      // 问题：两个请求同时读到stock=10，都判断充足，导致超卖
  }
  
  // ✅ 方案2：数据库悲观锁（可行，但性能差）
  func DecrStockWithDBLock(ctx context.Context, bookID uint, quantity int) error {
      return db.Transaction(func(tx *gorm.DB) error {
          var inv Inventory
          // SELECT FOR UPDATE：锁定该行，其他事务等待
          if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
              First(&inv, "book_id = ?", bookID).Error; err != nil {
              return err
          }
          
          if inv.Stock < quantity {
              return ErrInsufficientStock
          }
          
          inv.Stock -= quantity
          return tx.Save(&inv).Error
      })
      // 优点：强一致性
      // 缺点：DB锁竞争激烈，TPS低（~500）
  }
  
  // ✅ 方案3：Redis + Lua脚本（推荐！高性能）
  const decrStockLua = `
  local key = KEYS[1]
  local quantity = tonumber(ARGV[1])
  
  local stock = tonumber(redis.call('GET', key))
  if not stock or stock < quantity then
      return 0  -- 库存不足
  end
  
  redis.call('DECRBY', key, quantity)
  return 1  -- 扣减成功
  `
  
  func (s *inventoryService) DecrStock(ctx context.Context, bookID uint, quantity int) error {
      key := fmt.Sprintf("stock:%d", bookID)
      
      // 步骤1：Redis原子扣减（Lua保证原子性）
      script := redis.NewScript(decrStockLua)
      result, err := script.Run(ctx, s.redis, []string{key}, quantity).Int()
      if err != nil {
          return err
      }
      
      if result == 0 {
          return ErrInsufficientStock
      }
      
      // 步骤2：记录库存日志（用于审计）
      log := &InventoryLog{
          BookID:    bookID,
          Quantity:  -quantity,
          Type:      LogTypeDecr,
          CreatedAt: time.Now(),
      }
      if err := s.logRepo.Create(ctx, log); err != nil {
          // 日志写入失败不影响主流程，只记录错误
          logger.Error("failed to create inventory log", zap.Error(err))
      }
      
      // 步骤3：异步同步到MySQL（最终一致性）
      s.producer.Send("inventory.changed", &InventoryEvent{
          BookID:   bookID,
          Quantity: -quantity,
      })
      
      return nil
      // 优点：TPS高（>10000），Redis单线程天然防并发
      // 缺点：Redis与MySQL最终一致（可接受，库存允许短暂不一致）
  }
  ```

- [ ] **库存锁定机制（订单创建流程）**

  ```go
  // 教学要点：库存锁定 vs 直接扣减
  //
  // 场景：用户下单后需要15分钟内完成支付
  // 问题：如果直接扣减库存，用户不支付会占用库存
  // 解决：锁定机制
  //   1. 下单时：锁定库存（stock → locked_stock）
  //   2. 支付成功：扣减locked_stock
  //   3. 支付超时/取消：释放locked_stock → stock
  
  type Inventory struct {
      BookID       uint  `gorm:"primaryKey"`
      Stock        int   `gorm:"comment:可用库存"`
      LockedStock  int   `gorm:"comment:锁定库存（待支付订单）"`
      TotalStock   int   `gorm:"comment:总库存=Stock+LockedStock"`
  }
  
  // LockStock 锁定库存（订单创建时调用）
  func (s *inventoryService) LockStock(ctx context.Context, bookID uint, quantity int) error {
      // Redis Lua脚本：
      // if stock >= quantity then
      //     stock = stock - quantity
      //     locked_stock = locked_stock + quantity
      //     return 1
      // else
      //     return 0
      // end
      
      const lockStockLua = `
      local stockKey = KEYS[1]
      local lockedKey = KEYS[2]
      local quantity = tonumber(ARGV[1])
      
      local stock = tonumber(redis.call('GET', stockKey) or 0)
      if stock < quantity then
          return 0
      end
      
      redis.call('DECRBY', stockKey, quantity)
      redis.call('INCRBY', lockedKey, quantity)
      return 1
      `
      
      stockKey := fmt.Sprintf("stock:%d", bookID)
      lockedKey := fmt.Sprintf("stock:locked:%d", bookID)
      
      script := redis.NewScript(lockStockLua)
      result, err := script.Run(ctx, s.redis, []string{stockKey, lockedKey}, quantity).Int()
      if err != nil {
          return err
      }
      
      if result == 0 {
          return ErrInsufficientStock
      }
      
      // 异步记录日志
      s.logLockEvent(ctx, bookID, quantity)
      
      return nil
  }
  
  // ReleaseStock 释放库存（订单取消时调用）
  func (s *inventoryService) ReleaseStock(ctx context.Context, bookID uint, quantity int) error {
      // locked_stock -= quantity
      // stock += quantity
      
      const releaseStockLua = `
      local stockKey = KEYS[1]
      local lockedKey = KEYS[2]
      local quantity = tonumber(ARGV[1])
      
      redis.call('DECRBY', lockedKey, quantity)
      redis.call('INCRBY', stockKey, quantity)
      return 1
      `
      
      // ... 实现类似LockStock
  }
  ```

- [ ] **数据库设计**
  ```sql
  CREATE DATABASE inventory_db;
  USE inventory_db;
  
  -- 库存表
  CREATE TABLE inventory (
      book_id BIGINT UNSIGNED PRIMARY KEY COMMENT '图书ID',
      stock INT NOT NULL DEFAULT 0 COMMENT '可用库存',
      locked_stock INT NOT NULL DEFAULT 0 COMMENT '锁定库存',
      total_stock INT NOT NULL DEFAULT 0 COMMENT '总库存',
      created_at DATETIME(3) NOT NULL,
      updated_at DATETIME(3) NOT NULL,
      INDEX idx_stock (stock)
  ) ENGINE=InnoDB COMMENT='库存表';
  
  -- 库存日志表（审计用）
  CREATE TABLE inventory_logs (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      book_id BIGINT UNSIGNED NOT NULL,
      quantity INT NOT NULL COMMENT '变化数量（正数=增加，负数=减少）',
      type TINYINT NOT NULL COMMENT '类型：1-锁定 2-释放 3-扣减 4-增加',
      order_id BIGINT UNSIGNED COMMENT '关联订单ID',
      remark VARCHAR(255) COMMENT '备注',
      created_at DATETIME(3) NOT NULL,
      INDEX idx_book_id (book_id),
      INDEX idx_created_at (created_at)
  ) ENGINE=InnoDB COMMENT='库存变更日志';
  ```

- [ ] **Redis数据初始化**
  ```bash
  # 将MySQL库存数据同步到Redis
  # Key: stock:{book_id}
  # Value: 可用库存数量
  
  # 示例：
  SET stock:1 100
  SET stock:2 200
  SET stock:locked:1 0
  SET stock:locked:2 0
  ```

- [ ] **测试验证**
  - 并发扣减测试（1000个goroutine同时扣减）
  - 验证无超卖（最终库存>=0）
  - 锁定/释放流程测试
  - 性能基准测试（TPS>10000）

---

### Day 31-32: order-service（订单服务）

**目标**：实现订单微服务，协调多个下游服务

**架构设计**：
```
order-service/
├── cmd/main.go
├── internal/
│   ├── grpc/
│   │   ├── handler/order_handler.go
│   │   └── client/                 # gRPC客户端
│   │       ├── user_client.go      # 调用user-service
│   │       ├── catalog_client.go   # 调用catalog-service
│   │       ├── inventory_client.go # 调用inventory-service
│   │       └── payment_client.go   # 调用payment-service
│   ├── domain/
│   │   └── order/
│   │       ├── entity.go           # 订单、订单项
│   │       ├── status.go           # 订单状态机
│   │       └── service.go
│   └── infrastructure/
│       └── persistence/mysql/
└── config/config.yaml
```

**实现清单**：

- [ ] **RPC方法实现**
  - CreateOrder：创建订单（核心流程）
  - GetOrder：获取订单详情
  - ListOrders：查询订单列表
  - CancelOrder：取消订单
  - UpdateOrderStatus：更新订单状态

- [ ] **核心流程：CreateOrder（调用链）**

  ```go
  // 教学要点：微服务编排流程
  //
  // 流程图：
  // 1. user-service.ValidateToken（验证用户身份）
  // 2. catalog-service.GetBook（获取图书信息，计算金额）
  // 3. inventory-service.LockStock（锁定库存）
  // 4. order-service创建订单（写数据库）
  // 5. 返回订单信息
  //
  // 注意：支付流程放在Week 7（Saga事务）实现
  
  func (s *orderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
      // 步骤1：验证用户Token（防止非法请求）
      userResp, err := s.userClient.ValidateToken(ctx, req.Token)
      if err != nil || !userResp.Valid {
          return nil, errors.ErrUnauthorized
      }
      userID := userResp.UserId
      
      // 步骤2：查询图书信息，计算订单金额
      var total int64
      var items []*OrderItem
      
      for _, item := range req.Items {
          // 调用catalog-service
          book, err := s.catalogClient.GetBook(ctx, item.BookId)
          if err != nil {
              return nil, errors.Wrap(err, "图书不存在")
          }
          
          // 使用下单时的价格（防止改价攻击）
          items = append(items, &OrderItem{
              BookID:   item.BookId,
              Quantity: item.Quantity,
              Price:    book.Price,  // 快照价格
          })
          
          total += book.Price * int64(item.Quantity)
      }
      
      // 步骤3：锁定库存（调用inventory-service）
      for _, item := range items {
          if err := s.inventoryClient.LockStock(ctx, item.BookID, int(item.Quantity)); err != nil {
              // 如果锁定失败，需要回滚已锁定的库存
              // Phase 2简化处理：直接返回错误
              // Week 7会用Saga事务解决
              return nil, errors.Wrap(err, "库存不足")
          }
      }
      
      // 步骤4：创建订单
      order := &Order{
          OrderNo: generateOrderNo(),
          UserID:  uint(userID),
          Total:   total,
          Status:  OrderStatusPending,  // 待支付
          Items:   items,
      }
      
      if err := s.repo.Create(ctx, order); err != nil {
          // 订单创建失败，释放库存
          s.rollbackInventory(ctx, items)
          return nil, err
      }
      
      return order, nil
  }
  
  // rollbackInventory 回滚库存锁定
  func (s *orderService) rollbackInventory(ctx context.Context, items []*OrderItem) {
      // 教学要点：补偿操作
      // 问题：如果释放失败怎么办？
      // 解决：Week 7会用消息队列保证最终一致性
      
      for _, item := range items {
          if err := s.inventoryClient.ReleaseStock(ctx, item.BookID, int(item.Quantity)); err != nil {
              // 释放失败，记录日志，后续人工介入或定时任务补偿
              logger.Error("failed to release stock",
                  zap.Uint("book_id", item.BookID),
                  zap.Error(err),
              )
          }
      }
  }
  ```

- [ ] **订单状态机（复用Phase 1）**
  ```go
  type OrderStatus int
  
  const (
      OrderStatusPending   OrderStatus = 1 // 待支付
      OrderStatusPaid      OrderStatus = 2 // 已支付
      OrderStatusShipped   OrderStatus = 3 // 已发货
      OrderStatusCompleted OrderStatus = 4 // 已完成
      OrderStatusCancelled OrderStatus = 5 // 已取消
  )
  
  // 合法状态流转
  var transitions = map[OrderStatus][]OrderStatus{
      OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},
      OrderStatusPaid:      {OrderStatusShipped, OrderStatusCancelled},
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

- [ ] **数据库设计**
  ```sql
  CREATE DATABASE order_db;
  USE order_db;
  
  -- 订单表
  CREATE TABLE orders (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      order_no VARCHAR(32) UNIQUE NOT NULL COMMENT '订单号',
      user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
      total BIGINT NOT NULL COMMENT '订单总金额（分）',
      status TINYINT NOT NULL DEFAULT 1 COMMENT '订单状态',
      created_at DATETIME(3) NOT NULL,
      updated_at DATETIME(3) NOT NULL,
      INDEX idx_user_id (user_id),
      INDEX idx_status (status),
      INDEX idx_created_at (created_at)
  ) ENGINE=InnoDB COMMENT='订单表';
  
  -- 订单明细表
  CREATE TABLE order_items (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      order_id BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
      book_id BIGINT UNSIGNED NOT NULL COMMENT '图书ID',
      quantity INT NOT NULL DEFAULT 1 COMMENT '购买数量',
      price BIGINT NOT NULL COMMENT '下单时的单价（分）',
      INDEX idx_order_id (order_id)
  ) ENGINE=InnoDB COMMENT='订单明细表';
  ```

- [ ] **测试验证**
  - 完整下单流程测试
  - 库存不足场景测试
  - 订单状态流转测试
  - 并发下单测试

---

### Day 33-34: payment-service（支付服务）

**目标**：实现支付接口（Mock实现，真实支付Week 8对接）

**实现清单**：

- [ ] **RPC方法实现**
  - CreatePayment：创建支付单
  - QueryPayment：查询支付状态
  - MockCallback：模拟支付回调（测试用）

- [ ] **支付流程（Mock实现）**
  ```go
  // 教学要点：支付接口设计
  //
  // 真实场景：对接支付宝/微信支付
  // Phase 2：Mock实现，直接返回成功
  // Week 8：对接真实支付网关
  
  func (s *paymentService) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error) {
      payment := &Payment{
          PaymentNo: generatePaymentNo(),
          OrderID:   req.OrderId,
          Amount:    req.Amount,
          Status:    PaymentStatusPending,
      }
      
      if err := s.repo.Create(ctx, payment); err != nil {
          return nil, err
      }
      
      // Mock：3秒后自动回调成功（模拟真实支付）
      go func() {
          time.Sleep(3 * time.Second)
          s.mockCallback(context.Background(), payment.ID)
      }()
      
      return payment, nil
  }
  
  func (s *paymentService) mockCallback(ctx context.Context, paymentID uint) {
      // 更新支付状态
      if err := s.repo.UpdateStatus(ctx, paymentID, PaymentStatusSuccess); err != nil {
          logger.Error("mock callback failed", zap.Error(err))
          return
      }
      
      // 发送消息通知order-service（Week 7引入消息队列）
      logger.Info("payment success", zap.Uint("payment_id", paymentID))
  }
  ```

- [ ] **数据库设计**
  ```sql
  CREATE DATABASE payment_db;
  USE payment_db;
  
  CREATE TABLE payments (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      payment_no VARCHAR(32) UNIQUE NOT NULL COMMENT '支付单号',
      order_id BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
      amount BIGINT NOT NULL COMMENT '支付金额（分）',
      status TINYINT NOT NULL DEFAULT 1 COMMENT '支付状态：1-待支付 2-成功 3-失败',
      created_at DATETIME(3) NOT NULL,
      updated_at DATETIME(3) NOT NULL,
      INDEX idx_order_id (order_id)
  ) ENGINE=InnoDB COMMENT='支付表';
  ```

- [ ] **测试验证**
  - 支付创建测试
  - Mock回调测试
  - 支付状态查询测试

---

### Day 35: 服务发现（Consul集成）

**目标**：从硬编码服务地址迁移到动态服务发现

**实现清单**：

- [ ] **Consul部署**
  ```yaml
  # docker-compose.yml 新增Consul服务
  consul:
    image: consul:1.16
    container_name: bookstore-consul
    ports:
      - "8500:8500"
      - "8600:8600/udp"
    command: agent -server -ui -bootstrap-expect=1 -client=0.0.0.0
  ```

- [ ] **服务注册**
  ```go
  // 教学要点：服务注册与健康检查
  //
  // 每个服务启动时：
  // 1. 注册到Consul（服务名、地址、端口）
  // 2. 定义健康检查（HTTP /health 或 gRPC health check）
  // 3. 心跳上报（默认10秒）
  
  func registerService(consulAddr, serviceName, serviceAddr string, servicePort int) error {
      client, err := consulapi.NewClient(&consulapi.Config{
          Address: consulAddr,
      })
      if err != nil {
          return err
      }
      
      registration := &consulapi.AgentServiceRegistration{
          ID:      fmt.Sprintf("%s-%s", serviceName, uuid.New().String()),
          Name:    serviceName,
          Address: serviceAddr,
          Port:    servicePort,
          Check: &consulapi.AgentServiceCheck{
              GRPC:                           fmt.Sprintf("%s:%d", serviceAddr, servicePort),
              Interval:                       "10s",
              Timeout:                        "3s",
              DeregisterCriticalServiceAfter: "30s",
          },
      }
      
      return client.Agent().ServiceRegister(registration)
  }
  ```

- [ ] **服务发现（客户端）**
  ```go
  // 教学要点：gRPC Resolver集成Consul
  //
  // 从硬编码：
  //   conn, _ := grpc.Dial("localhost:9001", grpc.WithInsecure())
  //
  // 到服务发现：
  //   conn, _ := grpc.Dial("consul://user-service", grpc.WithInsecure())
  
  import _ "github.com/mbobakov/grpc-consul-resolver"
  
  func newUserServiceClient(consulAddr string) (userv1.UserServiceClient, error) {
      // 注册Consul Resolver
      target := fmt.Sprintf("consul://%s/user-service", consulAddr)
      
      conn, err := grpc.Dial(
          target,
          grpc.WithInsecure(),
          grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
      )
      if err != nil {
          return nil, err
      }
      
      return userv1.NewUserServiceClient(conn), nil
  }
  ```

- [ ] **负载均衡测试**
  - 启动2个user-service实例（9001、9002端口）
  - 客户端调用观察负载均衡效果
  - 停止一个实例，验证自动剔除

---

## 📊 Week 6 完成标准

### 代码质量

- [ ] 所有服务编译通过
- [ ] gRPC方法全部实现
- [ ] 单元测试覆盖核心逻辑
- [ ] 代码注释占比>40%（TEACHING.md要求）
- [ ] 通过golangci-lint检查

### 功能验证

- [ ] catalog-service：5个RPC方法测试通过
- [ ] inventory-service：6个RPC方法测试通过，并发扣减无超卖
- [ ] order-service：完整下单流程成功
- [ ] payment-service：Mock支付流程正常
- [ ] Consul：所有服务注册成功，健康检查通过
- [ ] api-gateway：集成新服务，HTTP接口正常

### 教学文档

- [ ] Day 29-30文档（catalog + inventory实现）
- [ ] Day 31-32文档（order实现）
- [ ] Day 33-34文档（payment实现）
- [ ] Day 35文档（Consul集成）
- [ ] Week 6总结文档

### 性能指标

- [ ] catalog-service列表查询QPS>1000
- [ ] inventory-service扣减TPS>10000
- [ ] order-service下单TPS>500
- [ ] 缓存命中率>80%

---

## 🎓 Week 6 学习要点总结

### 1. 服务拆分原则（DDD）

- 每个服务独立数据库
- 服务间只通过gRPC通信
- 避免循环依赖

### 2. 高并发优化

- Redis Lua脚本保证原子性
- 库存锁定机制
- 缓存策略设计

### 3. 分布式基础

- 服务编排（order-service调用链）
- 补偿操作（库存回滚）
- 服务发现（Consul）

### 4. DO/DON'T对比（教学重点）

每个关键模块都包含：
- ✅ 正确做法及原理
- ❌ 错误做法及后果
- 性能对比数据
- 常见陷阱说明

---

## 🚀 准备开始Week 6！

完成Week 6后，整个微服务体系将搭建完成，为Week 7的分布式事务（Saga）和Week 8的服务治理打下坚实基础。

**记住TEACHING.md的核心原则**：
- 渐进式实现（从简单到复杂）
- 丰富的教学注释（>40%）
- DO/DON'T对比
- 每个模块可运行、可测试

加油！💪
