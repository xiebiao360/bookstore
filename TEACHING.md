# 图书商城项目教学指导原则

> **本文档说明**：这是整个学习项目的指导性文件，所有代码实现都必须遵循此原则。

---

## 📖 核心教学目标

本项目不是为了"快速完成一个商城系统"，而是为了**系统性掌握Go微服务架构的核心能力**：

1. **工程化能力**：规范的项目结构、依赖管理、配置管理、日志处理
2. **架构设计能力**：领域驱动设计、分层架构、服务拆分、接口设计
3. **分布式系统能力**：分布式事务、数据一致性、服务治理、可观测性
4. **高并发优化**：并发控制、缓存策略、性能调优

---

## 🎯 教学模式运作规则

### 1. 蓝图驱动开发

**强制要求**：
- 所有开发任务必须参考 `ROADMAP.md` 中的当前阶段计划
- 技术选型必须符合蓝图规定（Phase 1不允许引入gRPC、消息队列等Phase 2技术）
- 架构设计必须为后续阶段留好扩展点（如Repository接口、Service分层）

**检查清单**（每次编码前确认）：
- [ ] 当前任务属于ROADMAP.md的哪个Phase？
- [ ] 使用的技术栈是否符合该Phase的规定？
- [ ] 实现方式是否便于Phase迁移？（如Phase 1的本地事务→Phase 2的Saga）

---

### 2. 代码必须具备教学价值

**注释规范**：

```go
// ❌ 错误示例：无意义注释
// Create user
func (s *userService) Create(user *User) error {
    return s.repo.Create(user)
}

// ✅ 正确示例：解释设计思想和最佳实践
// Register 用户注册流程
// 设计要点：
// 1. 使用bcrypt加密密码（cost=12，平衡安全性与性能）
// 2. 邮箱唯一性校验在数据库层通过UNIQUE索引保证（防止并发注册）
// 3. 密码验证规则：8-20位，必须包含字母+数字
func (s *userService) Register(ctx context.Context, email, password string) (*User, error) {
    // 业务规则验证
    if err := validatePassword(password); err != nil {
        return nil, errors.ErrWeakPassword
    }
    
    // 密码加密（使用bcrypt而非MD5/SHA1，防止彩虹表攻击）
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        return nil, errors.Wrap(err, "密码加密失败")
    }
    
    // 持久化（仓储层会处理邮箱重复异常）
    user := &User{Email: email, Password: string(hashedPassword)}
    return user, s.repo.Create(ctx, user)
}
```

**关键模块必须解释的内容**：
- **为什么这样设计**（如为何使用Repository模式而非直接调用GORM）
- **有哪些替代方案**（如JWT vs Session，各自优缺点）
- **常见陷阱**（如bcrypt cost太低不安全、太高影响性能）

---

### 3. 渐进式实现（禁止跳跃）

**阶段依赖关系**：
```
Phase 1: 单体分层架构
  ↓ 必须完全掌握：分层设计、事务处理、错误处理
  
Phase 2: 微服务拆分
  ↓ 必须完全掌握：服务拆分、分布式事务、熔断降级
  
Phase 3: Kubernetes部署
```

**禁止行为**：
- ❌ Phase 1直接引入服务网格、消息队列
- ❌ 未理解本地事务就跳到分布式事务
- ❌ 为了"看起来高大上"堆砌技术

**推荐节奏**：
- 每完成一个Week的任务，停下来回顾：学到了什么？为什么这样设计？
- 每个Phase结束后，能独立回答ROADMAP.md中的"技能掌握清单"

---

### 4. 实战为王

**可运行性要求**：
- 任何功能模块完成后，必须能通过 `make run` 启动并测试
- 提供完整的本地开发环境（docker-compose一键启动MySQL+Redis）
- 包含Swagger文档或curl命令，方便手动测试

**测试要求**：
```go
// 核心业务逻辑必须有单元测试（使用Mock）
func TestUserService_Register_EmailDuplicate(t *testing.T) {
    repo := new(mockUserRepository)
    repo.On("Create", mock.Anything, mock.Anything).Return(errors.ErrEmailDuplicate)
    
    svc := user.NewService(repo)
    _, err := svc.Register(context.Background(), "test@example.com", "password123")
    
    assert.ErrorIs(t, err, errors.ErrEmailDuplicate)
}

// 关键流程必须有集成测试（真实数据库）
func TestCreateOrder_Integration(t *testing.T) {
    db := setupTestDB(t) // 使用Docker启动测试MySQL
    defer teardownTestDB(t, db)
    
    // 准备测试数据
    user := createTestUser(t, db)
    book := createTestBook(t, db, stock: 10)
    
    // 执行下单
    order, err := orderService.CreateOrder(ctx, user.ID, []OrderItem{
        {BookID: book.ID, Quantity: 2},
    })
    
    // 验证结果
    assert.NoError(t, err)
    assert.Equal(t, OrderStatusPending, order.Status)
    
    // 验证库存已扣减
    updatedBook, _ := bookRepo.FindByID(ctx, book.ID)
    assert.Equal(t, 8, updatedBook.Stock)
}
```

---

### 5. 文档同步更新

**必须维护的文档**：
1. **README.md**：项目介绍、快速开始、目录结构说明
2. **ROADMAP.md**：学习蓝图（本文档）
3. **CHANGELOG.md**：每个阶段完成后记录重要变更
4. **ADR（架构决策记录）**：重要技术选型的理由

**ADR示例**（保存在 `docs/adr/001-use-repository-pattern.md`）：
```markdown
# ADR 001: 使用Repository模式隔离数据访问层

## 状态
已采纳

## 背景
领域层（domain）需要持久化数据，但不应依赖具体的数据库实现（GORM、sqlx）。

## 决策
采用Repository模式：
- domain层定义Repository接口
- infrastructure层实现具体的MySQL Repository
- 应用层通过接口调用，不感知底层实现

## 后果
### 优点
- 便于单元测试（Mock接口）
- 未来可无缝切换数据库（PostgreSQL、MongoDB）
- 符合依赖倒置原则

### 缺点
- 增加代码量（需要定义接口）
- 学习曲线略高

## 替代方案
直接在Service层调用GORM（被拒绝，因为耦合度高）
```

---

## 🔍 代码审查标准

每个模块完成后，按以下标准自我检查：

### 1. 可维护性（Maintainability）
- [ ] 函数单一职责（长度<50行）
- [ ] 嵌套层级≤3
- [ ] 变量命名清晰（userID vs uid）
- [ ] 魔法数字已提取为常量

### 2. 可测试性（Testability）
- [ ] 核心业务逻辑有单元测试
- [ ] 使用接口而非具体类型（便于Mock）
- [ ] 避免全局变量（使用依赖注入）

### 3. 性能（Performance）
- [ ] 数据库查询有适当索引
- [ ] 避免N+1查询（使用Preload）
- [ ] 热点数据有缓存
- [ ] 使用pprof分析过性能瓶颈

### 4. 安全性（Security）
- [ ] 密码使用bcrypt加密
- [ ] SQL注入防护（使用参数化查询）
- [ ] JWT签名验证
- [ ] 敏感信息不记录到日志

### 5. 规范性（Code Style）
- [ ] 通过golangci-lint检查
- [ ] 导出函数有godoc注释
- [ ] 错误处理完整（不吞噬error）
- [ ] 使用context传递超时控制

### 6. 文档完整性（Documentation）
- [ ] 关键设计有注释说明
- [ ] API有Swagger文档
- [ ] README包含启动步骤
- [ ] 复杂业务有流程图

---

## 📦 Phase转换检查点

### Phase 1 → Phase 2迁移前确认

**知识掌握**：
- [ ] 理解DDD的实体、值对象、聚合根概念
- [ ] 能独立实现CRUD+事务
- [ ] 理解依赖注入的优势
- [ ] 能进行基本的性能分析

**代码质量**：
- [ ] 核心业务逻辑测试覆盖率>80%
- [ ] 无明显性能瓶颈（单机QPS>1000）
- [ ] 错误处理规范统一
- [ ] API文档完整

**架构准备**：
- [ ] user/book/order模块边界清晰
- [ ] Repository接口已定义
- [ ] 服务层不依赖HTTP框架
- [ ] 配置可通过环境变量覆盖

### Phase 2 → Phase 3迁移前确认

**知识掌握**：
- [ ] 理解CAP理论与最终一致性
- [ ] 能设计Saga补偿事务
- [ ] 理解熔断降级原理
- [ ] 能排查分布式链路问题

**代码质量**：
- [ ] 服务间调用有超时控制
- [ ] 关键操作有幂等性保证
- [ ] 有完整的链路追踪
- [ ] 监控指标已暴露

**架构准备**：
- [ ] 服务可独立部署
- [ ] 配置已中心化（Consul）
- [ ] 日志已结构化
- [ ] 健康检查端点已实现

---

## 💡 常见陷阱与最佳实践

### 1. 过度设计（Overengineering）
❌ **错误示例**：Phase 1就引入事件溯源、CQRS
✅ **正确做法**：先用简单的CRUD，等遇到性能瓶颈再优化

### 2. 忽略错误处理
❌ **错误示例**：
```go
user, _ := userRepo.FindByID(ctx, id) // 吞噬错误
```
✅ **正确做法**：
```go
user, err := userRepo.FindByID(ctx, id)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.ErrUserNotFound
    }
    return nil, errors.Wrap(err, "查询用户失败")
}
```

### 3. 裸奔的并发（无锁保护）
❌ **错误示例**：
```go
// 并发扣库存，会超卖！
book.Stock -= quantity
db.Save(&book)
```
✅ **正确做法**：
```go
// 使用悲观锁
db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id)
if book.Stock < quantity {
    return errors.ErrInsufficientStock
}
book.Stock -= quantity
db.Save(&book)
```

### 4. 日志打印敏感信息
❌ **错误示例**：
```go
logger.Info("user login", zap.String("password", password))
```
✅ **正确做法**：
```go
logger.Info("user login", zap.String("email", email))
```

---

## 📊 进度追踪

**当前阶段**：Phase 1 - Week 1
**完成标志**：
- [ ] 项目脚手架搭建完成
- [ ] Docker环境一键启动
- [ ] 用户注册/登录功能可用
- [ ] 通过Swagger测试API

**下一里程碑**：Week 2 - 图书模块与订单模块

---

**记住**：学习的目标不是"完成项目"，而是"理解原理"。宁可慢一点把基础打牢，也不要急着往前赶。

**遇到问题时**：
1. 先查ROADMAP.md是否有说明
2. 阅读代码注释理解设计意图
3. 运行测试用例看预期行为
4. 仍有疑问再向导师提问

祝学习顺利！🚀
