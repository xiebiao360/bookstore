// Package circuitbreaker 实现熔断器模式（Circuit Breaker Pattern）
//
// 熔断器核心思想：
// 1. 监控服务调用的成功率和响应时间
// 2. 当失败率超过阈值时，快速失败（打开熔断器）
// 3. 过一段时间后尝试恢复（半开状态）
//
// 为什么需要熔断器？
// - 防止雪崩效应：服务A依赖服务B，B故障导致A也阻塞
// - 快速失败：B故障时，A立即返回错误，不等待超时
// - 自动恢复：B恢复后，熔断器自动关闭
//
// 教学要点：
// - 理解熔断器的三种状态（CLOSED、OPEN、HALF_OPEN）
// - 掌握状态转换条件
// - 对比手写实现 vs Sentinel框架
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State 熔断器状态
type State int

const (
	// StateClosed 关闭状态（正常）
	// - 所有请求正常通过
	// - 统计失败次数/失败率
	// - 达到阈值时转为OPEN
	StateClosed State = iota

	// StateOpen 打开状态（熔断）
	// - 所有请求快速失败，不调用服务
	// - 过一段时间（timeout）后转为HALF_OPEN
	// - 目的：给下游服务恢复时间
	StateOpen

	// StateHalfOpen 半开状态（探测）
	// - 允许部分请求通过（探测下游是否恢复）
	// - 如果请求成功，转为CLOSED
	// - 如果请求失败，转回OPEN
	StateHalfOpen
)

// String 状态转字符串（便于日志）
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config 熔断器配置
type Config struct {
	// MaxRequests 半开状态下允许的最大请求数
	// 建议值：1-5（允许少量请求探测）
	MaxRequests uint32

	// Interval 统计时间窗口
	// 建议值：10s-60s
	// 示例：过去10秒内的失败率
	Interval time.Duration

	// Timeout 熔断超时时间（OPEN状态持续时间）
	// 建议值：30s-60s
	// 过了这个时间，转为HALF_OPEN尝试恢复
	Timeout time.Duration

	// ReadyToTrip 判断是否应该打开熔断器
	// 参数：counts 当前统计数据
	// 返回：true表示应该熔断
	//
	// 常见策略：
	// 1. 失败率：counts.ConsecutiveFailures >= 5
	// 2. 错误率：counts.FailureRate() > 0.5 (50%)
	// 3. 慢调用：counts.SlowRate() > 0.3 (30%)
	ReadyToTrip func(counts Counts) bool
}

// Counts 统计数据
type Counts struct {
	Requests             uint32 // 总请求数
	TotalSuccesses       uint32 // 总成功数
	TotalFailures        uint32 // 总失败数
	ConsecutiveSuccesses uint32 // 连续成功数
	ConsecutiveFailures  uint32 // 连续失败数
}

// FailureRate 计算失败率
func (c *Counts) FailureRate() float64 {
	if c.Requests == 0 {
		return 0
	}
	return float64(c.TotalFailures) / float64(c.Requests)
}

// Reset 重置统计
func (c *Counts) Reset() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// onSuccess 记录成功
func (c *Counts) onSuccess() {
	// 注意：Requests已经在beforeRequest中递增，这里不再重复
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0 // 重置连续失败
}

// onFailure 记录失败
func (c *Counts) onFailure() {
	// 注意：Requests已经在beforeRequest中递增，这里不再重复
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0 // 重置连续成功
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name          string        // 熔断器名称（用于日志）
	maxRequests   uint32        // 半开状态最大请求数
	interval      time.Duration // 统计时间窗口
	timeout       time.Duration // 熔断超时时间
	readyToTrip   func(counts Counts) bool
	state         State                                   // 当前状态
	generation    uint64                                  // 生成号（每次状态切换递增）
	counts        Counts                                  // 统计数据
	expiry        time.Time                               // 过期时间（用于重置统计窗口）
	mu            sync.Mutex                              // 保护并发访问
	onStateChange func(name string, from State, to State) // 状态变化回调
}

// ErrOpenState 熔断器打开错误
var ErrOpenState = errors.New("circuit breaker is open")

// NewCircuitBreaker 创建熔断器
//
// 参数：
//
//	name: 熔断器名称（用于日志）
//	config: 配置
//
// 示例：
//
//	cb := NewCircuitBreaker("inventory-service", Config{
//	    MaxRequests: 3,
//	    Interval:    10 * time.Second,
//	    Timeout:     30 * time.Second,
//	    ReadyToTrip: func(counts Counts) bool {
//	        return counts.ConsecutiveFailures >= 5
//	    },
//	})
func NewCircuitBreaker(name string, config Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		state:         StateClosed,
		counts:        Counts{},
		expiry:        time.Now().Add(config.Interval),
		onStateChange: func(name string, from State, to State) {}, // 默认空回调
	}

	return cb
}

// SetStateChangeCallback 设置状态变化回调
//
// 用途：
// - 记录日志
// - 发送告警
// - 更新监控指标
func (cb *CircuitBreaker) SetStateChangeCallback(fn func(name string, from State, to State)) {
	cb.onStateChange = fn
}

// Execute 执行请求（核心方法）
//
// 参数：
//
//	req: 实际的业务请求函数
//
// 返回：
//
//	error: 业务错误 或 熔断器错误(ErrOpenState)
//
// 执行流程：
// 1. 检查当前状态
// 2. 根据状态决定是否执行请求
// 3. 记录请求结果
// 4. 更新状态
//
// 示例：
//
//	err := cb.Execute(func() error {
//	    return inventoryClient.DeductStock(ctx, bookID, quantity)
//	})
func (cb *CircuitBreaker) Execute(req func() error) error {
	// 步骤1：before request（检查是否允许执行）
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	// 步骤2：执行实际请求
	err = req()

	// 步骤3：after request（记录结果，更新状态）
	cb.afterRequest(generation, err == nil)

	return err
}

// beforeRequest 请求前检查
//
// 返回：
//
//	generation: 当前生成号（用于afterRequest）
//	error: 如果熔断器打开，返回ErrOpenState
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		// 熔断器打开，快速失败
		return generation, ErrOpenState
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		// 半开状态，已达到最大请求数
		return generation, ErrOpenState
	}

	// 允许请求通过
	cb.counts.Requests++
	return generation, nil
}

// afterRequest 请求后处理
//
// 参数：
//
//	before: beforeRequest返回的生成号
//	success: 请求是否成功
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	// 检查生成号是否匹配（防止状态已切换）
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess 处理成功请求
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	cb.counts.onSuccess()

	if state == StateHalfOpen {
		// 半开状态下成功，转为关闭状态
		cb.setState(StateClosed, now)
	}
}

// onFailure 处理失败请求
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	cb.counts.onFailure()

	switch state {
	case StateClosed:
		// 关闭状态下失败，检查是否应该熔断
		if cb.readyToTrip(cb.counts) {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		// 半开状态下失败，立即转回打开状态
		cb.setState(StateOpen, now)
	}
}

// currentState 获取当前状态
//
// 处理状态过期逻辑：
// - CLOSED状态：统计窗口过期时重置计数
// - OPEN状态：超时后转为HALF_OPEN
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		// 检查统计窗口是否过期
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			// 重置统计窗口
			cb.counts.Reset()
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		// 检查是否应该转为半开状态
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}

	return cb.state, cb.generation
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.generation++
	cb.counts.Reset()

	// 设置过期时间
	switch state {
	case StateClosed:
		cb.expiry = now.Add(cb.interval)
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	case StateHalfOpen:
		cb.expiry = time.Time{} // 半开状态没有过期时间
	}

	// 触发回调
	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}
}

// State 获取当前状态（只读）
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts 获取当前统计数据（只读）
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return cb.counts
}

// ==================== DO/DON'T 对比 ====================

// ❌ DON'T: 不使用熔断器，直接调用服务
//
// 问题场景：
// 1. inventory-service宕机
// 2. order-service每次调用都等待超时（3秒）
// 3. 100个并发请求 = 100 * 3秒 = 300秒才能全部失败
// 4. order-service的goroutine堆积，最终OOM
//
// func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//     // 直接调用，没有熔断保护
//     resp, err := inventoryClient.DeductStock(ctx, req.BookID, req.Quantity)
//     if err != nil {
//         // 每次都等待超时，浪费资源
//         return err
//     }
//     // ...
// }

// ✅ DO: 使用熔断器保护服务调用
//
// 优点：
// 1. inventory-service宕机后，熔断器在5次失败后打开
// 2. 后续请求立即失败（<1ms），不等待超时
// 3. 30秒后自动尝试恢复（半开状态）
// 4. 保护order-service不被拖垮
//
// func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//     err := inventoryCB.Execute(func() error {
//         resp, err := inventoryClient.DeductStock(ctx, req.BookID, req.Quantity)
//         return err
//     })
//
//     if err == circuitbreaker.ErrOpenState {
//         // 熔断器打开，快速失败
//         return errors.New("inventory-service不可用，请稍后重试")
//     }
//
//     return err
// }

// ==================== 教学总结 ====================
//
// 熔断器 vs 超时控制：
//
// | 特性       | 熔断器                        | 超时控制                |
// |-----------|------------------------------|------------------------|
// | 触发条件   | 连续失败次数/失败率            | 单次请求超时            |
// | 保护范围   | 整个服务                      | 单个请求                |
// | 恢复机制   | 自动尝试恢复（半开状态）        | 无自动恢复              |
// | 失败速度   | 立即失败（<1ms）               | 等待超时（3s）          |
// | 适用场景   | 下游服务不稳定                 | 单个慢请求              |
//
// 关键学习点：
// 1. 熔断器保护的是调用方，不是被调用方
// 2. 三种状态的转换条件（CLOSED → OPEN → HALF_OPEN → CLOSED）
// 3. 半开状态的作用：探测下游是否恢复
// 4. 熔断器应该与降级策略配合使用
