// Package tracing 提供基于OpenTelemetry的分布式追踪框架
//
// # 什么是分布式追踪（Distributed Tracing）？
//
// 在微服务架构中，一个用户请求可能跨越多个服务：
//
//	用户请求 → API Gateway → 订单服务 → 库存服务 → 支付服务
//
// 问题：当请求变慢或失败时，如何定位是哪个服务的问题？
//
// # 分布式追踪核心概念
//
// 1. **Trace（追踪）**：一个完整的请求链路
//   - 示例：用户下单从开始到结束的全过程
//   - 包含多个Span
//
// 2. **Span（跨度）**：一个操作单元
//   - 示例：调用库存服务扣减库存
//   - 包含：操作名称、开始时间、结束时间、耗时、状态
//
// 3. **SpanContext（上下文）**：跨服务传递的元数据
//   - TraceID：标识整个请求链路（所有服务共享同一个TraceID）
//   - SpanID：标识当前操作
//   - ParentSpanID：标识父操作（构建调用树）
//
// # 追踪示例
//
//	Trace: 用户下单（TraceID=abc123）
//	├─ Span1: API Gateway处理请求（耗时10ms）
//	│  ├─ Span2: 订单服务创建订单（耗时50ms）
//	│  │  ├─ Span3: 库存服务扣减库存（耗时30ms）← 慢！
//	│  │  └─ Span4: 支付服务扣款（耗时15ms）
//	│  └─ Span5: 发送通知（耗时5ms）
//	总耗时：110ms，瓶颈在Span3
//
// # OpenTelemetry简介
//
// OpenTelemetry（简称OTel）是CNCF的可观测性标准，统一了：
// - Tracing（追踪）
// - Metrics（指标）
// - Logging（日志）
//
// **优点**：
// - 厂商中立（不绑定Jaeger、Zipkin、Datadog）
// - 自动注入（HTTP、gRPC、数据库调用自动创建Span）
// - 上下文传播（TraceID/SpanID自动跨服务传递）
//
// # DO/DON'T对比
//
// ❌ DON'T: 手动记录每个操作的耗时（无法关联）
//
//	func CreateOrder() {
//	    start := time.Now()
//	    // 调用库存服务
//	    inventoryClient.DeductStock()
//	    log.Printf("扣减库存耗时: %v", time.Since(start)) // ❌ 无法看到完整链路
//	}
//
// ✅ DO: 使用OpenTelemetry自动追踪
//
//	func CreateOrder(ctx context.Context) {
//	    // 创建Span（自动记录开始时间）
//	    ctx, span := tracer.Start(ctx, "CreateOrder")
//	    defer span.End() // 自动记录结束时间和耗时
//
//	    // 调用库存服务（自动传递TraceID/SpanID）
//	    inventoryClient.DeductStock(ctx) // ✅ ctx包含追踪信息
//
//	    // 在Jaeger UI可以看到完整的调用链路和每个步骤的耗时
//	}
//
// # 使用示例
//
//	// 1. 初始化全局Tracer Provider
//	shutdown, err := tracing.InitTracer(
//	    "order-service",                    // 服务名称
//	    "http://localhost:4318/v1/traces",  // Jaeger Collector地址
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(context.Background())
//
//	// 2. 在业务代码中使用
//	func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//	    // 创建根Span
//	    tracer := otel.Tracer("order-service")
//	    ctx, span := tracer.Start(ctx, "CreateOrder")
//	    defer span.End()
//
//	    // 添加自定义属性（便于筛选和调试）
//	    span.SetAttributes(
//	        attribute.String("user_id", req.UserID),
//	        attribute.Int("item_count", len(req.Items)),
//	    )
//
//	    // 调用其他服务（自动创建子Span）
//	    if err := inventoryService.DeductStock(ctx, req.Items); err != nil {
//	        span.RecordError(err) // 记录错误信息
//	        span.SetStatus(codes.Error, err.Error())
//	        return err
//	    }
//
//	    span.SetStatus(codes.Ok, "订单创建成功")
//	    return nil
//	}
//
// # 最佳实践
//
// 1. **Span命名规范**：
//   - 使用操作名而非变量值：`GetUser`（✅） vs `GetUser-123`（❌）
//   - 使用层级命名：`service.handler.GetUser`
//
// 2. **属性选择**：
//   - 添加有用的业务属性：user_id、order_id、item_count
//   - 避免敏感信息：密码、信用卡号、个人隐私
//
// 3. **错误处理**：
//   - 总是调用`span.RecordError(err)`记录错误
//   - 使用`span.SetStatus(codes.Error, ...)`标记失败
//
// 4. **资源清理**：
//   - 使用`defer span.End()`确保Span被关闭
//   - 程序退出时调用`shutdown()`刷新未发送的数据
//
// # 常见问题
//
// **Q1: Span太多会影响性能吗？**
// A: 轻微影响（每个Span约1-2微秒），但可通过采样控制：
//   - 开发环境：100%采样（AlwaysSample）
//   - 生产环境：1%采样（TraceIDRatioBased）
//
// **Q2: 如何在Jaeger UI查看追踪？**
// A: 访问http://localhost:16686，选择服务名，搜索TraceID或操作名
//
// **Q3: 如何关联日志和追踪？**
// A: 从Context提取TraceID，写入日志：
//
//	traceID := span.SpanContext().TraceID().String()
//	log.Printf("TraceID=%s, 订单创建成功", traceID)
//
// **Q4: HTTP客户端如何传递TraceID？**
// A: OpenTelemetry自动在HTTP Header中注入`traceparent`字段：
//
//	traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
//	              └─ TraceID ─────────────────────────┘ └─ SpanID ───┘
package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// InitTracer 初始化全局Tracer Provider
//
// 参数：
//   - serviceName: 服务名称（在Jaeger UI中显示）
//   - collectorURL: Jaeger Collector的OTLP端点（如：http://localhost:4318）
//
// 返回：
//   - shutdown: 关闭函数（程序退出时调用，确保数据刷新）
//   - error: 初始化失败时返回错误
//
// 设计要点：
// 1. 使用OTLP协议（OpenTelemetry Protocol）而非Jaeger原生协议
//   - 优点：厂商中立，未来可无缝切换到Zipkin、Datadog
//   - 缺点：需要Jaeger 1.35+支持OTLP
//
// 2. 采样策略：
//   - AlwaysSample（100%采样）：适合开发/测试环境
//   - 生产环境建议使用TraceIDRatioBased（如1%采样）
//
// 3. 资源属性：
//   - service.name: 服务名称（必需，用于在Jaeger UI中分组）
//   - service.version: 服务版本（可选，便于区分不同版本的性能）
//
// 示例：
//
//	shutdown, err := tracing.InitTracer(
//	    "order-service",
//	    "http://localhost:4318",
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(context.Background())
func InitTracer(serviceName, collectorURL string) (func(context.Context) error, error) {
	// 1. 创建OTLP gRPC Exporter
	// OTLP支持两种传输方式：
	// - gRPC（默认端口4317）：高性能，适合高吞吐场景
	// - HTTP（默认端口4318）：兼容性好，适合有防火墙限制的场景
	//
	// 注意：collectorURL不包含协议路径，OTLP gRPC会自动连接到 <host>:<port>
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"), // Jaeger OTLP gRPC端点
		otlptracegrpc.WithInsecure(),                 // 禁用TLS（生产环境应启用）
	)
	if err != nil {
		return nil, fmt.Errorf("创建OTLP exporter失败: %w", err)
	}

	// 2. 创建Resource（资源属性）
	// Resource描述产生遥测数据的实体（服务、主机、容器等）
	// 这些属性会附加到所有Span上，便于在Jaeger UI中筛选和分组
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			// service.name是必需属性，用于在Jaeger UI中标识服务
			semconv.ServiceName(serviceName),

			// 可选：添加更多属性
			// semconv.ServiceVersion("1.0.0"),
			// semconv.DeploymentEnvironment("production"),
			// semconv.HostName("server-01"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("创建资源属性失败: %w", err)
	}

	// 3. 创建Tracer Provider
	// TracerProvider是OpenTelemetry的核心组件，负责：
	// - 创建Tracer
	// - 管理Span的生命周期
	// - 应用采样策略
	// - 将Span批量发送到Exporter
	tp := sdktrace.NewTracerProvider(
		// 采样策略：AlwaysSample表示100%采样
		// 生产环境建议使用：
		// sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.01)) // 1%采样
		sdktrace.WithSampler(sdktrace.AlwaysSample()),

		// Span处理器：BatchSpanProcessor批量发送Span（性能优于SimpleSpanProcessor）
		// - 默认每2秒或512个Span发送一次
		// - 程序退出时调用shutdown()强制刷新剩余Span
		sdktrace.WithBatcher(exporter),

		// 资源属性
		sdktrace.WithResource(res),
	)

	// 4. 设置全局TracerProvider
	// 全局Provider的优点：
	// - 业务代码无需传递TracerProvider，直接使用otel.Tracer()获取
	// - 第三方库（HTTP、gRPC）自动使用全局Provider
	otel.SetTracerProvider(tp)

	// 5. 设置全局TextMapPropagator（上下文传播器）
	// Propagator负责在跨服务调用时传递TraceID/SpanID
	// - W3C Trace Context：标准的HTTP Header格式（traceparent、tracestate）
	// - Baggage：传递自定义键值对（如user_id、tenant_id）
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, // W3C Trace Context
			propagation.Baggage{},      // Baggage
		),
	)

	// 6. 返回关闭函数
	// shutdown确保所有Span被发送到Collector
	// 必须在程序退出前调用，否则可能丢失最后一批Span
	shutdown := func(ctx context.Context) error {
		// 设置5秒超时，防止shutdown阻塞过久
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}

	return shutdown, nil
}

// StartSpan 创建一个新的Span（便捷函数）
//
// 参数：
//   - ctx: 父Context（包含父Span信息）
//   - tracerName: Tracer名称（通常是服务名或模块名）
//   - spanName: Span名称（操作名称，如"CreateOrder"）
//
// 返回：
//   - context.Context: 包含新Span的Context（传递给下游调用）
//   - trace.Span: Span对象（用于添加属性、记录错误、设置状态）
//
// 设计要点：
// 1. Span命名规范：
//   - 使用操作名：GetUser、CreateOrder、DeductStock
//   - 避免动态值：GetUser-123（❌），应使用属性：span.SetAttributes(attribute.String("user_id", "123"))
//
// 2. Span层级：
//   - 根Span：第一个Span，没有父Span（如HTTP请求处理）
//   - 子Span：嵌套在根Span下（如数据库查询、RPC调用）
//
// 3. Context传递：
//   - 必须使用返回的ctx调用下游函数，否则无法构建调用树
//
// 示例：
//
//	func CreateOrder(ctx context.Context) error {
//	    // 创建根Span
//	    ctx, span := tracing.StartSpan(ctx, "order-service", "CreateOrder")
//	    defer span.End()
//
//	    // 添加属性
//	    span.SetAttributes(attribute.String("user_id", "123"))
//
//	    // 调用子操作（自动成为子Span）
//	    if err := deductStock(ctx); err != nil {
//	        span.RecordError(err)
//	        return err
//	    }
//
//	    return nil
//	}
//
//	func deductStock(ctx context.Context) error {
//	    // 创建子Span
//	    ctx, span := tracing.StartSpan(ctx, "order-service", "DeductStock")
//	    defer span.End()
//
//	    // ... 业务逻辑 ...
//	    return nil
//	}
func StartSpan(ctx context.Context, tracerName, spanName string) (context.Context, trace.Span) {
	// 从全局Provider获取Tracer
	// tracerName用于在Jaeger UI中标识Span的来源（服务或模块）
	tracer := otel.Tracer(tracerName)

	// 创建Span
	// - 如果ctx包含父Span，新Span会自动成为子Span
	// - 如果ctx不包含父Span，新Span成为根Span
	return tracer.Start(ctx, spanName)
}

// ExtractTraceID 从Context提取TraceID（用于关联日志）
//
// 参数：
//   - ctx: 包含Span的Context
//
// 返回：
//   - string: TraceID的十六进制字符串（32位，如"4bf92f3577b34da6a3ce929d0e0e4736"）
//
// 使用场景：
// 在日志中记录TraceID，便于从日志快速定位到Jaeger追踪：
//
//	traceID := tracing.ExtractTraceID(ctx)
//	log.Printf("TraceID=%s, 订单创建成功, OrderID=%s", traceID, orderID)
//
// 然后在Jaeger UI搜索TraceID，查看完整的调用链路
func ExtractTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// ExtractSpanID 从Context提取SpanID
//
// 参数：
//   - ctx: 包含Span的Context
//
// 返回：
//   - string: SpanID的十六进制字符串（16位，如"00f067aa0ba902b7"）
//
// 使用场景：
// 在分布式日志系统（如ELK）中关联Span：
//
//	spanID := tracing.ExtractSpanID(ctx)
//	log.WithFields(log.Fields{
//	    "trace_id": tracing.ExtractTraceID(ctx),
//	    "span_id":  spanID,
//	}).Info("订单创建成功")
func ExtractSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}
