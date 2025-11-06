package saga

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestSaga_Execute_Success 测试所有步骤成功的场景
func TestSaga_Execute_Success(t *testing.T) {
	executed := make([]string, 0)

	saga := NewSaga(5 * time.Second)

	// 添加步骤1：锁定库存
	saga.AddStep("锁定库存",
		func(ctx context.Context) error {
			executed = append(executed, "锁定库存")
			return nil
		},
		func(ctx context.Context) error {
			executed = append(executed, "释放库存")
			return nil
		},
	)

	// 添加步骤2：创建订单
	saga.AddStep("创建订单",
		func(ctx context.Context) error {
			executed = append(executed, "创建订单")
			return nil
		},
		func(ctx context.Context) error {
			executed = append(executed, "取消订单")
			return nil
		},
	)

	// 执行Saga
	err := saga.Execute(context.Background())
	if err != nil {
		t.Fatalf("Saga执行失败: %v", err)
	}

	// 验证执行顺序
	if len(executed) != 2 {
		t.Errorf("期望执行2个步骤，实际执行%d个", len(executed))
	}

	if executed[0] != "锁定库存" || executed[1] != "创建订单" {
		t.Errorf("执行顺序错误: %v", executed)
	}
}

// TestSaga_Execute_FailureAndCompensate 测试步骤失败触发补偿
func TestSaga_Execute_FailureAndCompensate(t *testing.T) {
	executed := make([]string, 0)

	saga := NewSaga(5 * time.Second)

	// 步骤1：锁定库存（成功）
	saga.AddStep("锁定库存",
		func(ctx context.Context) error {
			executed = append(executed, "锁定库存")
			return nil
		},
		func(ctx context.Context) error {
			executed = append(executed, "释放库存")
			return nil
		},
	)

	// 步骤2：创建订单（成功）
	saga.AddStep("创建订单",
		func(ctx context.Context) error {
			executed = append(executed, "创建订单")
			return nil
		},
		func(ctx context.Context) error {
			executed = append(executed, "取消订单")
			return nil
		},
	)

	// 步骤3：扣款（失败）
	saga.AddStep("扣款",
		func(ctx context.Context) error {
			executed = append(executed, "扣款")
			return errors.New("余额不足") // 模拟扣款失败
		},
		func(ctx context.Context) error {
			executed = append(executed, "退款")
			return nil
		},
	)

	// 执行Saga（应该失败）
	err := saga.Execute(context.Background())
	if err == nil {
		t.Fatal("Saga应该失败但返回成功")
	}

	// 验证执行顺序：正向3步 + 补偿2步（逆序）
	// 期望：锁定库存 → 创建订单 → 扣款（失败） → 取消订单 → 释放库存
	expected := []string{"锁定库存", "创建订单", "扣款", "取消订单", "释放库存"}

	if len(executed) != len(expected) {
		t.Errorf("期望执行%d个步骤，实际执行%d个: %v", len(expected), len(executed), executed)
	}

	for i, step := range expected {
		if executed[i] != step {
			t.Errorf("步骤%d期望'%s'，实际'%s'", i, step, executed[i])
		}
	}
}

// TestSaga_Execute_Timeout 测试超时触发补偿
func TestSaga_Execute_Timeout(t *testing.T) {
	executed := make([]string, 0)

	saga := NewSaga(100 * time.Millisecond) // 设置100ms超时

	// 步骤1：快速执行
	saga.AddStep("快速步骤",
		func(ctx context.Context) error {
			executed = append(executed, "快速步骤")
			return nil
		},
		func(ctx context.Context) error {
			executed = append(executed, "快速步骤补偿")
			return nil
		},
	)

	// 步骤2：慢速执行（超过超时时间）
	saga.AddStep("慢速步骤",
		func(ctx context.Context) error {
			select {
			case <-time.After(200 * time.Millisecond):
				executed = append(executed, "慢速步骤")
				return nil
			case <-ctx.Done():
				// Context超时，返回错误
				return ctx.Err()
			}
		},
		func(ctx context.Context) error {
			executed = append(executed, "慢速步骤补偿")
			return nil
		},
	)

	// 执行Saga（应该超时）
	err := saga.Execute(context.Background())
	if err == nil {
		t.Fatal("Saga应该超时但返回成功")
	}

	// 验证触发了补偿
	if len(executed) < 2 {
		t.Errorf("超时后应该触发补偿，实际执行: %v", executed)
	}

	if executed[len(executed)-1] != "快速步骤补偿" {
		t.Errorf("期望最后一步是补偿，实际: %v", executed)
	}
}

// TestSaga_CompensateIdempotency 测试补偿幂等性示例
func TestSaga_CompensateIdempotency(t *testing.T) {
	// 模拟已执行补偿的记录
	compensateLog := make(map[string]bool)

	// 幂等的补偿函数
	createIdempotentCompensate := func(orderID string) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			idempotencyKey := "compensate-order-" + orderID

			// 检查是否已执行
			if compensateLog[idempotencyKey] {
				// 已执行过，直接返回成功
				return nil
			}

			// 执行补偿操作
			// ... 实际的业务逻辑 ...

			// 记录幂等键
			compensateLog[idempotencyKey] = true
			return nil
		}
	}

	saga := NewSaga(5 * time.Second)
	saga.AddStep("创建订单",
		func(ctx context.Context) error {
			return nil
		},
		createIdempotentCompensate("12345"),
	)

	// 第一次执行补偿
	saga.executed = saga.steps // 模拟步骤已执行
	saga.compensate(context.Background())

	if len(compensateLog) != 1 {
		t.Errorf("期望记录1条幂等日志，实际%d条", len(compensateLog))
	}

	// 第二次执行补偿（模拟重试）
	saga.executed = saga.steps
	saga.compensate(context.Background())

	// 验证幂等键只记录一次
	if len(compensateLog) != 1 {
		t.Errorf("幂等性失败：期望记录1条日志，实际%d条", len(compensateLog))
	}
}

// ==================== 实战示例：订单下单Saga ====================

// 模拟真实的下单场景
type OrderSagaExample struct {
	bookID   uint
	quantity int
	userID   uint
	orderID  uint
	locked   bool
	created  bool
	paid     bool
}

func (o *OrderSagaExample) CreateOrderSaga() *Saga {
	saga := NewSaga(30 * time.Second)

	// 步骤1：锁定库存
	saga.AddStep("锁定库存",
		func(ctx context.Context) error {
			// 调用inventory-service的LockStock
			// resp, err := inventoryClient.LockStock(ctx, o.bookID, o.quantity)
			o.locked = true
			return nil
		},
		func(ctx context.Context) error {
			// 调用inventory-service的UnlockStock
			// inventoryClient.UnlockStock(ctx, o.bookID, o.quantity)
			o.locked = false
			return nil
		},
	)

	// 步骤2：创建订单
	saga.AddStep("创建订单",
		func(ctx context.Context) error {
			// 调用order-service的CreateOrder
			// order, err := orderRepo.Create(ctx, &Order{...})
			o.created = true
			o.orderID = 12345 // 模拟生成的订单ID
			return nil
		},
		func(ctx context.Context) error {
			// 调用order-service的CancelOrder
			// orderRepo.UpdateStatus(ctx, o.orderID, StatusCancelled)
			o.created = false
			return nil
		},
	)

	// 步骤3：扣款
	saga.AddStep("扣款",
		func(ctx context.Context) error {
			// 调用payment-service的Pay
			// resp, err := paymentClient.Pay(ctx, o.orderID, amount)
			o.paid = true
			return nil
		},
		func(ctx context.Context) error {
			// 调用payment-service的Refund
			// paymentClient.Refund(ctx, o.orderID)
			o.paid = false
			return nil
		},
	)

	return saga
}

func TestOrderSagaExample_Success(t *testing.T) {
	example := &OrderSagaExample{
		bookID:   1,
		quantity: 2,
		userID:   100,
	}

	saga := example.CreateOrderSaga()
	err := saga.Execute(context.Background())

	if err != nil {
		t.Fatalf("订单Saga执行失败: %v", err)
	}

	// 验证所有步骤都成功
	if !example.locked || !example.created || !example.paid {
		t.Error("订单Saga未完全执行")
	}
}

func TestOrderSagaExample_PaymentFailed(t *testing.T) {
	example := &OrderSagaExample{
		bookID:   1,
		quantity: 2,
		userID:   100,
	}

	saga := example.CreateOrderSaga()

	// 修改扣款步骤，模拟失败
	saga.steps[2].Action = func(ctx context.Context) error {
		return errors.New("余额不足")
	}

	err := saga.Execute(context.Background())

	if err == nil {
		t.Fatal("扣款失败应该触发Saga失败")
	}

	// 验证补偿已执行（库存已释放、订单已取消）
	if example.locked || example.created || example.paid {
		t.Error("补偿未执行，数据状态错误")
	}
}

// ==================== 性能测试 ====================

// BenchmarkSaga_Execute 性能基准测试
func BenchmarkSaga_Execute(b *testing.B) {
	saga := NewSaga(5 * time.Second)

	saga.AddStep("步骤1", func(ctx context.Context) error { return nil }, nil)
	saga.AddStep("步骤2", func(ctx context.Context) error { return nil }, nil)
	saga.AddStep("步骤3", func(ctx context.Context) error { return nil }, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = saga.Execute(context.Background())
		// 重置执行状态
		saga.executed = nil
	}
}
