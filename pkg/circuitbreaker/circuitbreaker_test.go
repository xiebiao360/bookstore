package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

// TestCircuitBreaker_ClosedState 测试关闭状态（正常）
func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	// 执行成功请求
	for i := 0; i < 10; i++ {
		err := cb.Execute(func() error {
			return nil // 模拟成功
		})
		if err != nil {
			t.Fatalf("期望成功，实际失败: %v", err)
		}
	}

	// 验证状态
	if cb.State() != StateClosed {
		t.Errorf("期望状态为CLOSED，实际%s", cb.State())
	}

	// 验证统计
	counts := cb.Counts()
	if counts.TotalSuccesses != 10 {
		t.Errorf("期望成功10次，实际%d次", counts.TotalSuccesses)
	}
}

// TestCircuitBreaker_OpenState 测试打开状态（熔断）
func TestCircuitBreaker_OpenState(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			// 连续失败5次触发熔断
			return counts.ConsecutiveFailures >= 5
		},
	})

	// 执行5次失败请求
	for i := 0; i < 5; i++ {
		_ = cb.Execute(func() error {
			return errors.New("service unavailable")
		})
	}

	// 验证状态变为OPEN
	if cb.State() != StateOpen {
		t.Errorf("期望状态为OPEN，实际%s", cb.State())
	}

	// 第6次请求应该立即失败（不调用实际函数）
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	if err != ErrOpenState {
		t.Errorf("期望返回ErrOpenState，实际%v", err)
	}

	if called {
		t.Error("熔断器打开时不应该调用实际函数")
	}
}

// TestCircuitBreaker_HalfOpenState 测试半开状态（探测）
func TestCircuitBreaker_HalfOpenState(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond, // 短超时方便测试
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	// 触发熔断（3次失败）
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("fail")
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("期望状态为OPEN，实际%s", cb.State())
	}

	// 等待超时，转为半开状态
	time.Sleep(150 * time.Millisecond)

	// 第一次请求应该被允许（探测）
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil // 成功
	})

	if err != nil {
		t.Errorf("半开状态第一次请求期望成功，实际%v", err)
	}

	if !called {
		t.Error("半开状态应该允许请求通过")
	}

	// 成功后应该转为CLOSED
	if cb.State() != StateClosed {
		t.Errorf("期望状态转为CLOSED，实际%s", cb.State())
	}
}

// TestCircuitBreaker_HalfOpenToOpen 测试半开状态失败后转回打开
func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	// 触发熔断
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("fail")
		})
	}

	// 等待转为半开
	time.Sleep(150 * time.Millisecond)

	// 半开状态下请求失败
	_ = cb.Execute(func() error {
		return errors.New("still fail")
	})

	// 应该立即转回OPEN
	if cb.State() != StateOpen {
		t.Errorf("期望状态转回OPEN，实际%s", cb.State())
	}
}

// TestCircuitBreaker_StateChangeCallback 测试状态变化回调
func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	stateChanges := make([]string, 0)

	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	cb.SetStateChangeCallback(func(name string, from State, to State) {
		stateChanges = append(stateChanges, from.String()+"->"+to.String())
	})

	// 触发状态变化：CLOSED -> OPEN
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return errors.New("fail")
		})
	}

	// 等待状态变化：OPEN -> HALF_OPEN
	time.Sleep(150 * time.Millisecond)
	_ = cb.Execute(func() error {
		return nil // 成功
	})

	// 验证状态变化记录
	expectedChanges := []string{
		"CLOSED->OPEN",
		"OPEN->HALF_OPEN",
		"HALF_OPEN->CLOSED",
	}

	if len(stateChanges) != len(expectedChanges) {
		t.Errorf("期望%d次状态变化，实际%d次: %v", len(expectedChanges), len(stateChanges), stateChanges)
	}

	for i, expected := range expectedChanges {
		if i >= len(stateChanges) {
			break
		}
		if stateChanges[i] != expected {
			t.Errorf("第%d次状态变化期望%s，实际%s", i, expected, stateChanges[i])
		}
	}
}

// TestCircuitBreaker_FailureRate 测试基于失败率的熔断
func TestCircuitBreaker_FailureRate(t *testing.T) {
	cb := NewCircuitBreaker("test", Config{
		MaxRequests: 3,
		Interval:    1 * time.Hour, // 使用长时间窗口，避免被重置
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			// 失败率超过50%且请求数>=10时熔断
			return counts.Requests >= 10 && counts.FailureRate() > 0.5
		},
	})

	// 执行10次请求：4次成功，6次失败（失败率60%）
	for i := 0; i < 10; i++ {
		index := i
		_ = cb.Execute(func() error {
			// 索引0,1,2,3成功，4,5,6,7,8,9失败（6次失败）
			if index < 4 {
				return nil // 成功
			}
			return errors.New("fail") // 失败
		})

		// 每次执行后打印状态
		if i == 9 {
			counts := cb.Counts()
			t.Logf("第10次请求后: 总请求=%d, 成功=%d, 失败=%d, 失败率=%.2f, 状态=%s",
				counts.Requests, counts.TotalSuccesses, counts.TotalFailures, counts.FailureRate(), cb.State())
		}
	}

	// 验证状态（应该在第10次请求后转为OPEN）
	if cb.State() != StateOpen {
		t.Errorf("期望状态为OPEN（失败率超过50%%），实际%s", cb.State())
	}
}

// ==================== 实战示例 ====================

// MockInventoryClient 模拟库存服务客户端
type MockInventoryClient struct {
	failCount int
	callCount int
}

func (c *MockInventoryClient) DeductStock(bookID, quantity int) error {
	c.callCount++
	if c.callCount <= c.failCount {
		return errors.New("inventory service unavailable")
	}
	return nil
}

// TestCircuitBreaker_RealWorld 真实场景测试
func TestCircuitBreaker_RealWorld(t *testing.T) {
	client := &MockInventoryClient{
		failCount: 5, // 前5次调用失败
	}

	cb := NewCircuitBreaker("inventory-service", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     200 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	// 记录熔断状态变化
	cb.SetStateChangeCallback(func(name string, from State, to State) {
		t.Logf("[%s] 状态变化: %s -> %s", name, from, to)
	})

	// 模拟10次下单请求
	for i := 1; i <= 10; i++ {
		err := cb.Execute(func() error {
			return client.DeductStock(1, 2)
		})

		if err == ErrOpenState {
			t.Logf("请求#%d: 熔断器打开，快速失败", i)
		} else if err != nil {
			t.Logf("请求#%d: 调用失败 (%v)", i, err)
		} else {
			t.Logf("请求#%d: 调用成功", i)
		}
	}

	// 验证：前5次失败触发熔断，6-10次快速失败
	if client.callCount != 5 {
		t.Errorf("期望实际调用5次，实际调用%d次", client.callCount)
	}

	// 等待熔断器恢复
	t.Log("等待熔断器超时...")
	time.Sleep(250 * time.Millisecond)

	// 半开状态下再次尝试
	err := cb.Execute(func() error {
		return client.DeductStock(1, 2)
	})

	if err != nil {
		t.Errorf("半开状态下期望成功，实际失败: %v", err)
	}

	// 验证状态恢复为CLOSED
	if cb.State() != StateClosed {
		t.Errorf("期望状态恢复为CLOSED，实际%s", cb.State())
	}

	t.Logf("熔断器已恢复，最终调用次数: %d", client.callCount)
}

// BenchmarkCircuitBreaker 性能基准测试
func BenchmarkCircuitBreaker(b *testing.B) {
	cb := NewCircuitBreaker("bench", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}
}
