# Sentinel-Golang集成指南

> **文档说明**: 本文档介绍如何将Sentinel-Golang集成到bookstore项目中  
> **前置知识**: 需要先理解Week 8手写的熔断器框架  
> **目标**: 学习生产级流量治理框架的使用

---

## 📚 什么是Sentinel？

**Sentinel** 是阿里巴巴开源的流量治理组件，提供：
- **流量控制**（Rate Limiting）：限制QPS/并发数
- **熔断降级**（Circuit Breaking）：服务故障时快速失败
- **系统自适应保护**：根据系统负载自动调整流量
- **热点参数限流**：针对热点数据限流

**Sentinel vs 手写熔断器**：

| 特性 | 手写熔断器（Week 8） | Sentinel |
|-----|------------------|----------|
| **熔断功能** | ✅ 基础熔断 | ✅ 高级熔断（慢调用比例、异常比例） |
| **限流功能** | ❌ 不支持 | ✅ 支持（QPS、并发、预热） |
| **热点限流** | ❌ 不支持 | ✅ 支持 |
| **监控面板** | ❌ 不支持 | ✅ Dashboard可视化 |
| **规则动态配置** | ❌ 硬编码 | ✅ 动态推送（Nacos/Apollo） |
| **学习成本** | 低（理解原理） | 中（学习API） |
| **生产就绪** | ❌ 需完善 | ✅ 阿里巴巴生产验证 |

---

## 🚀 快速开始

### 1. 安装依赖

```bash
cd /home/xiebiao/Workspace/bookstore
go get github.com/alibaba/sentinel-golang@v1.0.4
```

### 2. 初始化Sentinel

创建 `pkg/sentinel/sentinel.go`：

```go
package sentinel

import (
	"log"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/flow"
)

// Init 初始化Sentinel
//
// 教学要点：
// - Sentinel需要在main函数启动时初始化
// - 规则可以硬编码（学习）或动态推送（生产）
func Init() error {
	// 1. 初始化Sentinel核心组件
	if err := sentinel.InitDefault(); err != nil {
		return err
	}

	log.Println("✅ Sentinel初始化成功")

	// 2. 配置流控规则（可选）
	initFlowRules()

	// 3. 配置熔断规则（可选）
	initCircuitBreakerRules()

	return nil
}

// initFlowRules 配置流控规则
func initFlowRules() {
	// 示例：限制inventory-service的QPS为100
	_, err := flow.LoadRules([]*flow.Rule{
		{
			Resource:               "inventory-service",
			TokenCalculateStrategy: flow.Direct,  // 直接统计QPS
			ControlBehavior:        flow.Reject,  // 超过阈值直接拒绝
			Threshold:              100,          // QPS阈值
			StatIntervalInMs:       1000,         // 统计窗口1秒
		},
	})
	if err != nil {
		log.Printf("⚠️ 流控规则加载失败: %v", err)
		return
	}

	log.Println("✅ 流控规则加载成功")
}

// initCircuitBreakerRules 配置熔断规则
func initCircuitBreakerRules() {
	// 示例：inventory-service错误率超过50%时熔断
	_, err := circuitbreaker.LoadRules([]*circuitbreaker.Rule{
		{
			Resource:         "inventory-service",
			Strategy:         circuitbreaker.ErrorRatio,  // 错误率策略
			RetryTimeoutMs:   30000,                      // 熔断持续30秒
			MinRequestAmount: 10,                         // 最小请求数
			StatIntervalMs:   10000,                      // 统计窗口10秒
			Threshold:        0.5,                        // 错误率阈值50%
		},
	})
	if err != nil {
		log.Printf("⚠️ 熔断规则加载失败: %v", err)
		return
	}

	log.Println("✅ 熔断规则加载成功")
}
```

---

## 🔧 在order-service中集成Sentinel

### 1. 修改main.go

```go
// services/order-service/cmd/main.go
package main

import (
	"log"
	
	sentinelPkg "github.com/xiebiao/bookstore/pkg/sentinel"
	// ... 其他导入
)

func main() {
	// 1. 初始化Sentinel（在其他服务之前）
	if err := sentinelPkg.Init(); err != nil {
		log.Fatalf("Sentinel初始化失败: %v", err)
	}

	// 2. 初始化配置、数据库等
	// ...

	// 3. 启动gRPC服务
	// ...
}
```

### 2. 在gRPC Handler中使用Sentinel

```go
// services/order-service/internal/grpc/handler/order_handler.go
package handler

import (
	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
)

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	// ... 参数校验 ...

	// 构建Saga
	sagaCtx := &CreateOrderSagaContext{...}
	orderSaga := s.buildCreateOrderSagaWithSentinel(sagaCtx)

	// 执行Saga
	if err := orderSaga.Execute(ctx); err != nil {
		return &CreateOrderResponse{Code: 50000, Message: err.Error()}, nil
	}

	return &CreateOrderResponse{...}, nil
}

// buildCreateOrderSagaWithSentinel 使用Sentinel保护的Saga流程
func (s *OrderServiceServer) buildCreateOrderSagaWithSentinel(sagaCtx *CreateOrderSagaContext) *saga.Saga {
	orderSaga := saga.NewSaga(30 * time.Second)

	// 步骤1：查询图书信息
	orderSaga.AddStep("查询图书信息",
		func(ctx context.Context) error {
			// 使用Sentinel保护catalog-service调用
			return s.callWithSentinel("catalog-service", func() error {
				for _, item := range sagaCtx.items {
					bookResp, err := s.catalogClient.GetBook(ctx, uint(item.BookId), timeout)
					if err != nil {
						return fmt.Errorf("图书[%d]不存在", item.BookId)
					}
					// ... 构建orderItems ...
				}
				return nil
			})
		},
		nil,
	)

	// 步骤2：扣减库存（Sentinel保护）
	orderSaga.AddStep("扣减库存",
		func(ctx context.Context) error {
			return s.callWithSentinel("inventory-service", func() error {
				for _, item := range sagaCtx.items {
					resp, err := s.inventoryClient.DeductStock(ctx, ...)
					if err != nil {
						return fmt.Errorf("库存不足[图书:%d]", item.BookId)
					}
					sagaCtx.deductedBookIDs = append(sagaCtx.deductedBookIDs, uint(item.BookId))
				}
				return nil
			})
		},
		func(ctx context.Context) error {
			// 补偿：释放库存（也可以加Sentinel保护）
			return s.callWithSentinel("inventory-service", func() error {
				for _, bookID := range sagaCtx.deductedBookIDs {
					s.inventoryClient.ReleaseStock(ctx, bookID, quantity, ...)
				}
				return nil
			})
		},
	)

	// ... 其他步骤 ...

	return orderSaga
}

// callWithSentinel Sentinel保护的服务调用封装
//
// 教学要点：
// - Entry/Exit模式是Sentinel的核心用法
// - 支持流控、熔断、系统保护等多种功能
// - 错误处理要区分业务错误和Sentinel错误
func (s *OrderServiceServer) callWithSentinel(resource string, fn func() error) error {
	// 1. 尝试获取Entry（检查是否限流/熔断）
	entry, blockErr := sentinel.Entry(
		resource,
		sentinel.WithTrafficType(base.Outbound), // 出站流量
	)

	if blockErr != nil {
		// 2. 被限流或熔断
		if blockErr.BlockType() == base.BlockTypeFlow {
			return fmt.Errorf("服务[%s]限流：QPS超过阈值", resource)
		} else if blockErr.BlockType() == base.BlockTypeCircuitBreaking {
			return fmt.Errorf("服务[%s]熔断：错误率过高", resource)
		}
		return fmt.Errorf("服务[%s]被拒绝: %v", resource, blockErr)
	}

	// 3. 允许通过，执行业务逻辑
	defer entry.Exit()

	err := fn()

	// 4. 记录错误（用于熔断统计）
	if err != nil {
		sentinel.TraceError(entry, err)
	}

	return err
}
```

---

## 📊 Sentinel熔断规则详解

### 1. 错误率熔断（Error Ratio）

```go
circuitbreaker.Rule{
	Resource:         "inventory-service",
	Strategy:         circuitbreaker.ErrorRatio,  // 错误率策略
	RetryTimeoutMs:   30000,                      // 熔断30秒
	MinRequestAmount: 10,                         // 最小请求数（避免冷启动误判）
	StatIntervalMs:   10000,                      // 统计窗口10秒
	Threshold:        0.5,                        // 错误率50%
}
```

**触发条件**：
- 10秒内请求数 >= 10
- 错误率 >= 50%

**熔断后**：
- 所有请求快速失败（30秒）
- 30秒后进入半开状态（探测）

### 2. 慢调用比例熔断（Slow Request Ratio）

```go
circuitbreaker.Rule{
	Resource:         "payment-service",
	Strategy:         circuitbreaker.SlowRequestRatio,  // 慢调用比例
	RetryTimeoutMs:   30000,
	MinRequestAmount: 10,
	StatIntervalMs:   10000,
	Threshold:        0.3,                             // 慢调用比例30%
	MaxAllowedRtMs:   1000,                            // 响应时间>1秒算慢调用
}
```

**触发条件**：
- 10秒内请求数 >= 10
- 响应时间>1秒的请求比例 >= 30%

**适用场景**：
- 下游服务响应变慢（不一定报错）
- 数据库查询变慢

---

## 🎯 流控规则详解

### 1. QPS限流

```go
flow.Rule{
	Resource:               "create-order",
	TokenCalculateStrategy: flow.Direct,    // 直接统计QPS
	ControlBehavior:        flow.Reject,    // 超过阈值拒绝
	Threshold:              100,            // QPS=100
	StatIntervalInMs:       1000,           // 统计窗口1秒
}
```

**效果**：
- 每秒最多处理100个CreateOrder请求
- 超过部分直接拒绝（返回限流错误）

### 2. 并发数限流

```go
flow.Rule{
	Resource:               "create-order",
	TokenCalculateStrategy: flow.Direct,
	ControlBehavior:        flow.Reject,
	Threshold:              50,              // 并发数50
	RelationStrategy:       flow.CurrentResource,
	StatIntervalInMs:       1000,
}
```

**效果**：
- 同时最多处理50个CreateOrder请求
- 第51个请求被拒绝

### 3. 预热限流（Warm Up）

```go
flow.Rule{
	Resource:               "create-order",
	TokenCalculateStrategy: flow.WarmUp,    // 预热模式
	ControlBehavior:        flow.Reject,
	Threshold:              100,            // 目标QPS=100
	WarmUpPeriodSec:        30,             // 预热30秒
	StatIntervalInMs:       1000,
}
```

**效果**：
- 系统启动后，QPS从10逐渐增加到100（30秒内）
- 避免冷启动时流量突增导致系统崩溃

---

## 🔍 DO/DON'T 对比

### ❌ DON'T: 过度依赖Sentinel

```go
// 错误示例：所有代码都用Sentinel包裹
func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
	entry, _ := sentinel.Entry("create-order")
	defer entry.Exit()

	// 查询用户
	userEntry, _ := sentinel.Entry("get-user")
	user := getUserFromDB(req.UserID)
	userEntry.Exit()

	// 查询图书
	bookEntry, _ := sentinel.Entry("get-book")
	book := getBookFromDB(req.BookID)
	bookEntry.Exit()

	// ... 每个操作都包一层，代码冗余 ...
}
```

**问题**：
- 代码可读性差
- 过度限流（内部操作不需要限流）
- 性能开销

### ✅ DO: 只保护关键调用

```go
// 正确示例：只保护RPC调用和核心业务逻辑
func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
	// 1. 本地参数校验（不需要Sentinel）
	if err := validateRequest(req); err != nil {
		return err
	}

	// 2. 查询本地数据库（不需要Sentinel）
	user := getUserFromDB(req.UserID)

	// 3. RPC调用inventory-service（需要Sentinel保护）
	if err := callWithSentinel("inventory-service", func() error {
		return inventoryClient.DeductStock(ctx, req.BookID, req.Quantity)
	}); err != nil {
		return err
	}

	// 4. 创建订单（不需要Sentinel）
	return createOrderInDB(order)
}
```

**原则**：
- ✅ 保护RPC调用（跨服务）
- ✅ 保护核心接口（HTTP入口、gRPC入口）
- ❌ 不保护内部函数调用

---

## 📈 监控与Dashboard

### 1. 启动Sentinel Dashboard

```bash
# 下载Dashboard
wget https://github.com/alibaba/Sentinel/releases/download/1.8.6/sentinel-dashboard-1.8.6.jar

# 启动（端口8080）
java -Dserver.port=8080 \
     -Dcsp.sentinel.dashboard.server=localhost:8080 \
     -jar sentinel-dashboard-1.8.6.jar
```

### 2. 客户端连接Dashboard

```go
// pkg/sentinel/sentinel.go
func Init() error {
	conf := config.NewDefaultConfig()
	
	// 连接Dashboard
	conf.Sentinel.App.Name = "bookstore-order-service"
	conf.Sentinel.Log.Dir = "/tmp/sentinel/logs"
	conf.Sentinel.Exporter.Metric.HttpAddr = ":8719"  // 暴露指标端口
	
	if err := sentinel.InitWithConfig(conf); err != nil {
		return err
	}

	log.Println("✅ Sentinel已连接Dashboard: http://localhost:8080")
	return nil
}
```

### 3. Dashboard功能

访问 `http://localhost:8080`：
- **实时监控**：QPS、响应时间、错误率
- **规则配置**：动态调整流控/熔断规则
- **链路追踪**：查看调用链路
- **机器列表**：查看接入的应用实例

---

## 🎓 学习路径建议

### 阶段1：理解原理（Week 8已完成）
- ✅ 手写熔断器框架
- ✅ 理解三态模型
- ✅ 掌握状态转换条件

### 阶段2：学习Sentinel（当前）
- 📖 阅读Sentinel官方文档
- 🔧 集成到order-service
- 🧪 测试流控和熔断功能

### 阶段3：生产实践（Week 9-10）
- 📊 集成Dashboard监控
- 🔄 动态规则配置（Nacos）
- 📈 性能压测验证

---

## 🚀 扩展阅读

### 1. Sentinel vs Hystrix

| 特性 | Sentinel | Hystrix（Netflix） |
|-----|---------|-------------------|
| **语言** | Java/Go/C++ | Java |
| **维护状态** | ✅ 活跃 | ❌ 停止维护（2018） |
| **限流** | ✅ 支持 | ❌ 不支持 |
| **热点限流** | ✅ 支持 | ❌ 不支持 |
| **Dashboard** | ✅ 功能丰富 | ✅ 基础功能 |
| **规则动态配置** | ✅ 支持 | ❌ 需重启 |

### 2. Sentinel-Golang GitHub

- **官方仓库**: https://github.com/alibaba/sentinel-golang
- **文档**: https://sentinelguard.io/zh-cn/docs/golang/quick-start.html
- **示例代码**: https://github.com/alibaba/sentinel-golang/tree/master/example

### 3. 阿里云AHAS

阿里云提供商业化的Sentinel服务：
- 托管Dashboard
- 规则持久化
- 多集群管理
- 告警通知

---

## ✅ 总结

**手写熔断器 vs Sentinel**：

| 维度 | 手写熔断器 | Sentinel |
|-----|----------|----------|
| **学习价值** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **生产就绪** | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **功能丰富度** | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **可维护性** | ⭐⭐⭐ | ⭐⭐⭐⭐ |

**建议**：
- 📚 **学习阶段**：先手写框架（理解原理）
- 🚀 **生产阶段**：使用Sentinel（功能完善、经过验证）
- 🔄 **渐进式**：Week 8手写 → Week 9集成Sentinel → Week 10生产优化

Sentinel不仅仅是熔断器，更是一个完整的**流量治理解决方案**，值得深入学习！
