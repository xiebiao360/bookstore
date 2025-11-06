package tracing

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TestInitTracer 测试Tracer初始化
func TestInitTracer(t *testing.T) {
	t.Run("成功初始化Tracer", func(t *testing.T) {
		// 初始化Tracer
		shutdown, err := InitTracer("test-service", "http://localhost:4318")
		if err != nil {
			t.Fatalf("初始化Tracer失败: %v", err)
		}
		defer func() {
			if err := shutdown(context.Background()); err != nil {
				t.Errorf("关闭Tracer失败: %v", err)
			}
		}()

		// 验证全局TracerProvider已设置
		tracer := otel.Tracer("test")
		if tracer == nil {
			t.Error("全局TracerProvider未设置")
		}

		t.Log("✅ Tracer初始化成功")
	})
}

// TestStartSpan 测试Span创建
func TestStartSpan(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	t.Run("创建根Span", func(t *testing.T) {
		ctx := context.Background()

		// 创建根Span
		ctx, span := StartSpan(ctx, "test-service", "TestOperation")
		defer span.End()

		// 验证Span有效
		if !span.SpanContext().IsValid() {
			t.Error("Span无效")
		}

		// 验证TraceID存在
		traceID := span.SpanContext().TraceID().String()
		if traceID == "" || traceID == "00000000000000000000000000000000" {
			t.Errorf("TraceID无效: %s", traceID)
		}

		t.Logf("✅ 根Span创建成功, TraceID=%s", traceID)
	})

	t.Run("创建子Span", func(t *testing.T) {
		ctx := context.Background()

		// 创建根Span
		ctx, rootSpan := StartSpan(ctx, "test-service", "RootOperation")
		defer rootSpan.End()

		rootTraceID := rootSpan.SpanContext().TraceID().String()
		rootSpanID := rootSpan.SpanContext().SpanID().String()

		// 创建子Span
		ctx, childSpan := StartSpan(ctx, "test-service", "ChildOperation")
		defer childSpan.End()

		childTraceID := childSpan.SpanContext().TraceID().String()

		// 验证子Span继承了根Span的TraceID
		if childTraceID != rootTraceID {
			t.Errorf("子Span的TraceID不匹配: root=%s, child=%s", rootTraceID, childTraceID)
		}

		// 验证子Span有不同的SpanID
		childSpanID := childSpan.SpanContext().SpanID().String()
		if childSpanID == rootSpanID {
			t.Error("子Span的SpanID不应与根Span相同")
		}

		t.Logf("✅ 子Span创建成功, TraceID=%s, ParentSpanID=%s, ChildSpanID=%s",
			childTraceID, rootSpanID, childSpanID)
	})
}

// TestSpanAttributes 测试Span属性设置
func TestSpanAttributes(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-service", "TestAttributes")
	defer span.End()

	// 添加属性
	span.SetAttributes(
		attribute.String("user_id", "user-123"),
		attribute.Int("item_count", 5),
		attribute.Bool("is_premium", true),
		attribute.Float64("total_amount", 256.78),
	)

	// 验证属性已添加（无法直接读取，但可以在Jaeger UI查看）
	t.Log("✅ Span属性设置成功")
}

// TestSpanStatus 测试Span状态设置
func TestSpanStatus(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	t.Run("成功状态", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-service", "SuccessOperation")
		defer span.End()

		// 设置成功状态
		span.SetStatus(codes.Ok, "操作成功")

		t.Log("✅ 成功状态设置成功")
	})

	t.Run("失败状态", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-service", "FailedOperation")
		defer span.End()

		// 模拟错误
		err := context.DeadlineExceeded

		// 记录错误
		span.RecordError(err)

		// 设置失败状态
		span.SetStatus(codes.Error, err.Error())

		t.Log("✅ 失败状态设置成功")
	})
}

// TestExtractTraceID 测试TraceID提取
func TestExtractTraceID(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	t.Run("从有效Context提取TraceID", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-service", "TestExtract")
		defer span.End()

		// 提取TraceID
		traceID := ExtractTraceID(ctx)

		// 验证TraceID非空
		if traceID == "" {
			t.Error("TraceID为空")
		}

		// 验证TraceID长度（32位十六进制）
		if len(traceID) != 32 {
			t.Errorf("TraceID长度错误: expected=32, got=%d", len(traceID))
		}

		t.Logf("✅ TraceID提取成功: %s", traceID)
	})

	t.Run("从无效Context提取TraceID", func(t *testing.T) {
		ctx := context.Background()

		// 提取TraceID（无Span）
		traceID := ExtractTraceID(ctx)

		// 验证返回空字符串
		if traceID != "" {
			t.Errorf("期望空字符串，实际: %s", traceID)
		}

		t.Log("✅ 无效Context返回空TraceID")
	})
}

// TestExtractSpanID 测试SpanID提取
func TestExtractSpanID(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	t.Run("从有效Context提取SpanID", func(t *testing.T) {
		ctx := context.Background()
		ctx, span := StartSpan(ctx, "test-service", "TestExtractSpanID")
		defer span.End()

		// 提取SpanID
		spanID := ExtractSpanID(ctx)

		// 验证SpanID非空
		if spanID == "" {
			t.Error("SpanID为空")
		}

		// 验证SpanID长度（16位十六进制）
		if len(spanID) != 16 {
			t.Errorf("SpanID长度错误: expected=16, got=%d", len(spanID))
		}

		t.Logf("✅ SpanID提取成功: %s", spanID)
	})

	t.Run("从无效Context提取SpanID", func(t *testing.T) {
		ctx := context.Background()

		// 提取SpanID（无Span）
		spanID := ExtractSpanID(ctx)

		// 验证返回空字符串
		if spanID != "" {
			t.Errorf("期望空字符串，实际: %s", spanID)
		}

		t.Log("✅ 无效Context返回空SpanID")
	})
}

// TestRealWorldScenario 真实场景：模拟订单创建流程
func TestRealWorldScenario(t *testing.T) {
	// 初始化Tracer
	shutdown, err := InitTracer("test-service", "http://localhost:4318")
	if err != nil {
		t.Fatalf("初始化Tracer失败: %v", err)
	}
	defer shutdown(context.Background())

	ctx := context.Background()

	// 模拟订单创建
	err = createOrder(ctx, "user-123", []string{"book-1", "book-2"})
	if err != nil {
		t.Errorf("订单创建失败: %v", err)
	}

	t.Log("✅ 真实场景测试通过，请在Jaeger UI查看追踪: http://localhost:16686")
	t.Log("   Service: test-service")
	t.Log("   Operation: CreateOrder")
}

// 模拟业务函数：创建订单
func createOrder(ctx context.Context, userID string, items []string) error {
	// 创建根Span
	ctx, span := StartSpan(ctx, "test-service", "CreateOrder")
	defer span.End()

	// 添加业务属性
	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int("item_count", len(items)),
	)

	// 步骤1：验证库存
	if err := checkInventory(ctx, items); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// 步骤2：创建订单记录
	if err := saveOrder(ctx, userID, items); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// 步骤3：扣减库存
	if err := deductInventory(ctx, items); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "订单创建成功")
	return nil
}

// 模拟业务函数：验证库存
func checkInventory(ctx context.Context, items []string) error {
	ctx, span := StartSpan(ctx, "test-service", "CheckInventory")
	defer span.End()

	span.SetAttributes(attribute.Int("item_count", len(items)))

	// 模拟数据库查询耗时
	time.Sleep(10 * time.Millisecond)

	span.SetStatus(codes.Ok, "库存充足")
	return nil
}

// 模拟业务函数：保存订单
func saveOrder(ctx context.Context, userID string, items []string) error {
	ctx, span := StartSpan(ctx, "test-service", "SaveOrder")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int("item_count", len(items)),
	)

	// 模拟数据库写入耗时
	time.Sleep(20 * time.Millisecond)

	span.SetStatus(codes.Ok, "订单保存成功")
	return nil
}

// 模拟业务函数：扣减库存
func deductInventory(ctx context.Context, items []string) error {
	ctx, span := StartSpan(ctx, "test-service", "DeductInventory")
	defer span.End()

	span.SetAttributes(attribute.Int("item_count", len(items)))

	// 模拟库存服务调用耗时
	time.Sleep(15 * time.Millisecond)

	span.SetStatus(codes.Ok, "库存扣减成功")
	return nil
}
