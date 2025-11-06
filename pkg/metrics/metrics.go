// Package metrics 提供基于Prometheus的指标收集框架
//
// # 什么是Metrics（指标）？
//
// Metrics是可观测性三支柱之一（Tracing、Metrics、Logging）：
// - **Tracing（追踪）**: 回答"为什么慢？"（Week 10已实现）
// - **Metrics（指标）**: 回答"有多少？多快？"（本模块）
// - **Logging（日志）**: 回答"发生了什么？"
//
// # 核心概念
//
// **1. Counter（计数器）**：只增不减的累计值
//   - 示例：HTTP请求总数、订单总数、错误总数
//   - 特点：只能调用Inc()递增
//
// **2. Gauge（仪表盘）**：可增可减的瞬时值
//   - 示例：当前在线用户数、goroutine数量、内存使用量
//   - 特点：可以调用Inc()、Dec()、Set()
//
// **3. Histogram（直方图）**：观测值的分布
//   - 示例：HTTP请求耗时、订单金额分布
//   - 特点：自动计算分位数（P50、P90、P99）
//
// **4. Summary（摘要）**：类似Histogram，但在客户端计算分位数
//   - 示例：请求耗时的P50、P90、P99
//   - 特点：更精确但消耗更多CPU
//
// # Prometheus架构
//
//	┌────────────────────────────────────────────────────────────┐
//	│  应用程序                                                    │
//	│  ├─ metrics.IncrementCounter("http_requests_total")         │
//	│  ├─ metrics.ObserveHistogram("http_duration", 0.123)        │
//	│  └─ /metrics端点暴露指标                                    │
//	└────────────────────────────────────────────────────────────┘
//	                         ↓
//	┌────────────────────────────────────────────────────────────┐
//	│  Prometheus Server                                          │
//	│  ├─ 每15秒抓取/metrics端点                                   │
//	│  ├─ 存储时序数据（时间序列数据库）                            │
//	│  └─ 提供PromQL查询语言                                       │
//	└────────────────────────────────────────────────────────────┘
//	                         ↓
//	┌────────────────────────────────────────────────────────────┐
//	│  Grafana                                                    │
//	│  ├─ 连接Prometheus数据源                                     │
//	│  ├─ 创建可视化Dashboard                                      │
//	│  └─ 配置告警规则                                             │
//	└────────────────────────────────────────────────────────────┘
//
// # DO/DON'T对比
//
// ❌ DON'T: 手动记录日志统计（无法聚合、查询困难）
//
//	func CreateOrder() {
//	    start := time.Now()
//	    // ... 业务逻辑 ...
//	    duration := time.Since(start)
//	    log.Printf("订单创建耗时: %v", duration) // ❌ 无法查询P99耗时
//	}
//
// 问题：
// 1. 无法聚合统计（总共多少个订单？平均耗时？）
// 2. 无法查询分位数（P99耗时是多少？）
// 3. 无法可视化（无法画图表）
// 4. 无法告警（耗时超过1秒无法自动告警）
//
// ✅ DO: 使用Prometheus指标
//
//	func CreateOrder() {
//	    start := time.Now()
//
//	    // ... 业务逻辑 ...
//
//	    // 记录耗时（自动计算P50、P90、P99）
//	    metrics.ObserveHistogram("order_creation_duration_seconds", time.Since(start).Seconds())
//
//	    // 递增订单计数
//	    metrics.IncrementCounter("orders_created_total")
//	}
//
// 优点：
// 1. ✅ 自动聚合（Prometheus每15秒抓取并存储）
// 2. ✅ 查询分位数（histogram_quantile(0.99, order_creation_duration_seconds)）
// 3. ✅ 可视化（Grafana创建图表）
// 4. ✅ 告警（Prometheus配置告警规则）
//
// # 使用示例
//
//	// 1. 初始化Metrics
//	metrics.InitMetrics()
//
//	// 2. 在HTTP服务中暴露/metrics端点
//	http.Handle("/metrics", promhttp.Handler())
//	go http.ListenAndServe(":9090", nil)
//
//	// 3. 在业务代码中记录指标
//	func CreateOrder(ctx context.Context) error {
//	    // 记录请求开始
//	    start := time.Now()
//
//	    // 递增处理中的请求数
//	    metrics.IncGauge("orders_in_progress")
//	    defer metrics.DecGauge("orders_in_progress")
//
//	    // 业务逻辑
//	    if err := doCreateOrder(ctx); err != nil {
//	        // 记录错误
//	        metrics.IncCounter("orders_failed_total")
//	        return err
//	    }
//
//	    // 记录成功
//	    metrics.IncCounter("orders_created_total")
//
//	    // 记录耗时
//	    metrics.ObserveHistogram("order_creation_duration_seconds", time.Since(start).Seconds())
//
//	    return nil
//	}
//
// # 常见指标命名规范
//
// 1. **Counter**: 以`_total`结尾
//   - `http_requests_total`（HTTP请求总数）
//   - `orders_created_total`（订单创建总数）
//
// 2. **Histogram**: 以单位结尾（`_seconds`、`_bytes`）
//   - `http_request_duration_seconds`（HTTP请求耗时）
//   - `order_amount_yuan`（订单金额）
//
// 3. **Gauge**: 使用现在时态
//   - `goroutines_running`（正在运行的goroutine数）
//   - `memory_usage_bytes`（内存使用量）
//
// # 最佳实践
//
//  1. **使用标签（Label）区分不同维度**：
//     ```go
//     metrics.IncCounterVec("http_requests_total", map[string]string{
//     "method": "POST",
//     "path":   "/api/orders",
//     "status": "200",
//     })
//     ```
//
// 2. **避免高基数标签（High Cardinality）**：
//   - ❌ 不要用user_id作为标签（百万级别）
//   - ✅ 用status、method作为标签（有限个值）
//
// 3. **选择合适的指标类型**：
//   - 计数用Counter：请求数、错误数、订单数
//   - 瞬时值用Gauge：连接数、队列长度、内存
//   - 分布用Histogram：耗时、大小、金额
//
// 4. **合理设置Histogram桶（Buckets）**：
//   - HTTP耗时：0.001, 0.01, 0.1, 0.5, 1, 5秒
//   - 订单金额：10, 50, 100, 500, 1000元
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// initialized 标记是否已初始化（防止重复注册）
	initialized bool

	// HTTP请求相关指标

	// HTTPRequestsTotal HTTP请求总数（Counter）
	// 标签：method（GET/POST）、path（/api/orders）、status（200/500）
	HTTPRequestsTotal *prometheus.CounterVec

	// HTTPRequestDuration HTTP请求耗时（Histogram）
	// 桶设置：1ms、10ms、100ms、500ms、1s、5s、10s
	HTTPRequestDuration *prometheus.HistogramVec

	// HTTPRequestsInProgress 正在处理的HTTP请求数（Gauge）
	HTTPRequestsInProgress prometheus.Gauge

	// 业务指标

	// OrdersCreatedTotal 订单创建总数（Counter）
	OrdersCreatedTotal prometheus.Counter

	// OrdersFailedTotal 订单创建失败总数（Counter）
	OrdersFailedTotal prometheus.Counter

	// OrderCreationDuration 订单创建耗时（Histogram）
	OrderCreationDuration prometheus.Histogram

	// OrdersInProgress 正在处理的订单数（Gauge）
	OrdersInProgress prometheus.Gauge

	// 熔断器指标

	// CircuitBreakerState 熔断器状态（Gauge）
	// 0=CLOSED, 1=OPEN, 2=HALF_OPEN
	CircuitBreakerState *prometheus.GaugeVec

	// CircuitBreakerRequests 熔断器请求总数（Counter）
	// 标签：name（熔断器名称）、result（success/failure/rejected）
	CircuitBreakerRequests *prometheus.CounterVec

	// Saga指标

	// SagaExecutionsTotal Saga执行总数（Counter）
	// 标签：result（success/failure）
	SagaExecutionsTotal *prometheus.CounterVec

	// SagaExecutionDuration Saga执行耗时（Histogram）
	SagaExecutionDuration prometheus.Histogram

	// SagaCompensationsTotal Saga补偿执行总数（Counter）
	SagaCompensationsTotal prometheus.Counter

	// 消息队列指标

	// MessagesPublishedTotal 消息发布总数（Counter）
	// 标签：exchange（交换机）、routing_key（路由键）
	MessagesPublishedTotal *prometheus.CounterVec

	// MessagesConsumedTotal 消息消费总数（Counter）
	// 标签：queue（队列名称）、result（success/failure）
	MessagesConsumedTotal *prometheus.CounterVec

	// MessageProcessingDuration 消息处理耗时（Histogram）
	MessageProcessingDuration prometheus.Histogram
)

// InitMetrics 初始化所有Prometheus指标
//
// 必须在程序启动时调用一次，用于注册所有指标到全局Registry
//
// 设计要点：
// 1. 使用promauto.New*自动注册到默认Registry
// 2. Counter使用*Vec支持标签（多维度统计）
// 3. Histogram的Buckets根据业务场景定制
//
// 示例：
//
//	func main() {
//	    // 初始化指标
//	    metrics.InitMetrics()
//
//	    // 暴露/metrics端点
//	    http.Handle("/metrics", promhttp.Handler())
//	    go http.ListenAndServe(":9090", nil)
//
//	    // 启动业务服务
//	    startServer()
//	}
func InitMetrics() {
	// 防止重复初始化
	if initialized {
		return
	}
	initialized = true

	// HTTP请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP请求总数",
		},
		[]string{"method", "path", "status"}, // 标签：方法、路径、状态码
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP请求耗时（秒）",
			// 桶设置：1ms、10ms、100ms、500ms、1s、5s、10s
			// 覆盖大部分HTTP请求耗时范围
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5, 10},
		},
		[]string{"method", "path"}, // 标签：方法、路径
	)

	HTTPRequestsInProgress = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "正在处理的HTTP请求数",
		},
	)

	// 订单业务指标
	OrdersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "订单创建总数",
		},
	)

	OrdersFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_failed_total",
			Help: "订单创建失败总数",
		},
	)

	OrderCreationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "order_creation_duration_seconds",
			Help: "订单创建耗时（秒）",
			// 订单创建通常较慢（涉及多个服务调用）
			// 桶设置：10ms、50ms、100ms、500ms、1s、5s、10s
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		},
	)

	OrdersInProgress = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "orders_in_progress",
			Help: "正在处理的订单数",
		},
	)

	// 熔断器指标
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "熔断器状态（0=CLOSED, 1=OPEN, 2=HALF_OPEN）",
		},
		[]string{"name"}, // 标签：熔断器名称
	)

	CircuitBreakerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "熔断器请求总数",
		},
		[]string{"name", "result"}, // 标签：熔断器名称、结果（success/failure/rejected）
	)

	// Saga指标
	SagaExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saga_executions_total",
			Help: "Saga执行总数",
		},
		[]string{"result"}, // 标签：结果（success/failure）
	)

	SagaExecutionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "saga_execution_duration_seconds",
			Help: "Saga执行耗时（秒）",
			// Saga执行时间较长（涉及多个步骤）
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30},
		},
	)

	SagaCompensationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "saga_compensations_total",
			Help: "Saga补偿执行总数",
		},
	)

	// 消息队列指标
	MessagesPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_published_total",
			Help: "消息发布总数",
		},
		[]string{"exchange", "routing_key"}, // 标签：交换机、路由键
	)

	MessagesConsumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_consumed_total",
			Help: "消息消费总数",
		},
		[]string{"queue", "result"}, // 标签：队列名称、结果（success/failure）
	)

	MessageProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "message_processing_duration_seconds",
			Help:    "消息处理耗时（秒）",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5},
		},
	)
}

// IncCounter 递增Counter（便捷函数）
func IncCounter(counter prometheus.Counter) {
	counter.Inc()
}

// IncCounterVec 递增CounterVec（带标签）
func IncCounterVec(counter *prometheus.CounterVec, labels map[string]string) {
	counter.With(labels).Inc()
}

// IncGauge 递增Gauge
func IncGauge(gauge prometheus.Gauge) {
	gauge.Inc()
}

// DecGauge 递减Gauge
func DecGauge(gauge prometheus.Gauge) {
	gauge.Dec()
}

// SetGauge 设置Gauge值
func SetGauge(gauge prometheus.Gauge, value float64) {
	gauge.Set(value)
}

// SetGaugeVec 设置GaugeVec值（带标签）
func SetGaugeVec(gauge *prometheus.GaugeVec, labels map[string]string, value float64) {
	gauge.With(labels).Set(value)
}

// ObserveHistogram 记录Histogram观测值
func ObserveHistogram(histogram prometheus.Histogram, value float64) {
	histogram.Observe(value)
}

// ObserveHistogramVec 记录HistogramVec观测值（带标签）
func ObserveHistogramVec(histogram *prometheus.HistogramVec, labels map[string]string, value float64) {
	histogram.With(labels).Observe(value)
}
