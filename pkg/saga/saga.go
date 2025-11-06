// Package saga 实现通用的Saga分布式事务框架
//
// Saga模式核心思想：
// 1. 将长事务拆分为多个本地短事务
// 2. 每个短事务有对应的补偿操作
// 3. 如果某步失败，按逆序执行已完成步骤的补偿操作
//
// 教学要点：
// - Saga vs 2PC（两阶段提交）的区别
// - 补偿操作的幂等性设计
// - 超时控制与故障恢复
package saga

import (
	"context"
	"fmt"
	"time"
)

// Step 表示Saga中的一个步骤
//
// 设计要点：
// 1. Action是正向操作（如扣减库存、创建订单）
// 2. Compensate是补偿操作（如释放库存、取消订单）
// 3. 每个操作都必须支持幂等（允许重试）
type Step struct {
	Name       string                          // 步骤名称（用于日志和调试）
	Action     func(ctx context.Context) error // 正向操作
	Compensate func(ctx context.Context) error // 补偿操作
}

// Saga 表示一个Saga事务
type Saga struct {
	steps    []Step        // 所有步骤
	executed []Step        // 已执行的步骤（用于补偿）
	timeout  time.Duration // 整体超时时间
}

// NewSaga 创建一个新的Saga事务
//
// 参数：
//
//	timeout: 整体超时时间，防止长时间阻塞
//
// 示例：
//
//	saga := NewSaga(30 * time.Second)
//	saga.AddStep("锁定库存", lockInventory, unlockInventory)
//	saga.AddStep("创建订单", createOrder, cancelOrder)
//	err := saga.Execute(ctx)
func NewSaga(timeout time.Duration) *Saga {
	return &Saga{
		steps:   make([]Step, 0),
		timeout: timeout,
	}
}

// AddStep 添加一个Saga步骤
//
// 设计原则：
// 1. 步骤顺序很重要（按添加顺序执行，按逆序补偿）
// 2. Action和Compensate都可以为nil（如最后一步通常无需补偿）
// 3. 建议每个步骤都实现补偿操作，除非确实不需要
//
// ❌ DON'T: 补偿操作依赖后续步骤
// 错误示例：步骤A的补偿依赖步骤B的结果
//
// ✅ DO: 补偿操作完全独立
// 正确示例：每个步骤的补偿只依赖自己的Action结果
func (s *Saga) AddStep(name string, action, compensate func(ctx context.Context) error) {
	s.steps = append(s.steps, Step{
		Name:       name,
		Action:     action,
		Compensate: compensate,
	})
}

// Execute 执行Saga事务
//
// 执行流程：
// 1. 按顺序执行每个步骤的Action
// 2. 如果某步失败，触发补偿流程（逆序执行已完成步骤的Compensate）
// 3. 返回错误信息
//
// 超时处理：
// - 整体超时时间由NewSaga的timeout参数指定
// - 超时时会立即触发补偿流程
//
// 幂等性要求：
// - Action和Compensate都必须支持幂等
// - 原因：网络故障可能导致重试
//
// ⚠️ 注意事项：
// 1. 补偿操作可能失败（需要人工介入或重试机制）
// 2. Saga保证"最终一致性"，而非"强一致性"
// 3. 补偿期间数据可能处于中间状态（需要业务容忍）
func (s *Saga) Execute(ctx context.Context) error {
	// 创建带超时的Context
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	// 按顺序执行每个步骤的Action
	for i, step := range s.steps {
		select {
		case <-ctx.Done():
			// 超时，触发补偿
			s.compensate(context.Background()) // 使用新Context，避免补偿也超时
			return fmt.Errorf("saga超时: %w", ctx.Err())
		default:
		}

		// 执行正向操作
		if step.Action != nil {
			if err := step.Action(ctx); err != nil {
				// 执行失败，触发补偿
				s.compensate(context.Background())
				return fmt.Errorf("步骤[%d:%s]执行失败: %w", i, step.Name, err)
			}
		}

		// 记录已执行的步骤（用于补偿）
		s.executed = append(s.executed, step)
	}

	return nil
}

// compensate 执行补偿流程
//
// 补偿原则：
// 1. 按逆序执行已完成步骤的Compensate
// 2. 即使某个Compensate失败，也继续执行后续补偿（尽最大努力）
// 3. 记录所有补偿错误，返回聚合错误
//
// 为什么逆序？
//   - 依赖关系：后执行的步骤可能依赖先执行的步骤
//   - 示例：先"创建订单"，后"扣减库存"
//     补偿时应先"释放库存"，再"取消订单"
//
// 补偿失败的处理：
// - 记录日志（需要人工介入）
// - 可选：发送告警通知
// - 可选：写入死信队列，稍后重试
func (s *Saga) compensate(ctx context.Context) {
	// 逆序执行补偿操作
	for i := len(s.executed) - 1; i >= 0; i-- {
		step := s.executed[i]

		if step.Compensate != nil {
			if err := step.Compensate(ctx); err != nil {
				// ⚠️ 补偿失败：记录日志，继续执行后续补偿
				// 生产环境应该：
				// 1. 记录到专门的补偿失败表
				// 2. 发送告警通知运维人员
				// 3. 提供重试机制或人工介入接口
				fmt.Printf("⚠️ 补偿失败[步骤:%s]: %v\n", step.Name, err)
			}
		}
	}

	// 清空已执行列表
	s.executed = nil
}

// ==================== DO/DON'T 对比示例 ====================

// ❌ DON'T: 补偿操作不幂等
//
// 错误示例：
// func compensateStock(ctx context.Context) error {
//     // 问题：如果重试，会多次增加库存
//     return db.Exec("UPDATE inventory SET stock = stock + 10 WHERE book_id = 1")
// }
//
// 后果：
// - 网络故障导致补偿操作重试
// - 库存被多次增加（原本扣10，补偿了20）

// ✅ DO: 补偿操作使用幂等键
//
// 正确示例：
// func compensateStock(ctx context.Context) error {
//     idempotencyKey := fmt.Sprintf("compensate-stock-%s", orderID)
//
//     // 检查幂等键是否已执行
//     var log CompensateLog
//     if db.Where("idempotency_key = ?", idempotencyKey).First(&log).Error == nil {
//         return nil // 已执行过，直接返回成功
//     }
//
//     // 事务：增加库存 + 记录幂等键
//     return db.Transaction(func(tx *gorm.DB) error {
//         if err := tx.Exec("UPDATE inventory SET stock = stock + 10 WHERE book_id = 1").Error; err != nil {
//             return err
//         }
//         return tx.Create(&CompensateLog{IdempotencyKey: idempotencyKey}).Error
//     })
// }

// ❌ DON'T: 补偿操作依赖外部状态
//
// 错误示例：
// var globalOrderID uint // 全局变量
//
// func compensateOrder(ctx context.Context) error {
//     // 问题：依赖全局变量，并发不安全
//     return db.Exec("UPDATE orders SET status = 'CANCELLED' WHERE id = ?", globalOrderID)
// }

// ✅ DO: 使用闭包捕获上下文
//
// 正确示例：
// func createOrderStep(orderID uint) Step {
//     return Step{
//         Name: "创建订单",
//         Action: func(ctx context.Context) error {
//             return db.Create(&Order{ID: orderID, Status: "PENDING"}).Error
//         },
//         Compensate: func(ctx context.Context) error {
//             // 闭包捕获orderID，线程安全
//             return db.Exec("UPDATE orders SET status = 'CANCELLED' WHERE id = ?", orderID).Error
//         },
//     }
// }

// ❌ DON'T: 忽略补偿失败
//
// 错误示例：
// func (s *Saga) compensate(ctx context.Context) {
//     for i := len(s.executed) - 1; i >= 0; i-- {
//         step := s.executed[i]
//         if step.Compensate != nil {
//             _ = step.Compensate(ctx) // 吞噬错误，不记录
//         }
//     }
// }
//
// 后果：
// - 补偿失败后无法追踪
// - 数据不一致无法发现

// ✅ DO: 记录补偿失败并告警
//
// 正确示例（生产环境）：
// func (s *Saga) compensate(ctx context.Context) {
//     failedSteps := make([]string, 0)
//
//     for i := len(s.executed) - 1; i >= 0; i-- {
//         step := s.executed[i]
//         if step.Compensate != nil {
//             if err := step.Compensate(ctx); err != nil {
//                 // 记录失败步骤
//                 failedSteps = append(failedSteps, step.Name)
//
//                 // 写入补偿失败表
//                 db.Create(&CompensateFailure{
//                     StepName: step.Name,
//                     Error:    err.Error(),
//                     SagaID:   s.id,
//                 })
//
//                 // 发送告警
//                 alerting.Send(fmt.Sprintf("Saga补偿失败: %s", step.Name))
//             }
//         }
//     }
//
//     if len(failedSteps) > 0 {
//         // 记录整体失败日志
//         log.Errorf("Saga补偿失败，需人工介入: %v", failedSteps)
//     }
// }

// ==================== 教学总结 ====================
//
// Saga vs 2PC（两阶段提交）：
//
// | 特性       | Saga                  | 2PC                        |
// |-----------|----------------------|----------------------------|
// | 一致性     | 最终一致性            | 强一致性                    |
// | 性能       | 高（无锁）            | 低（全局锁）                |
// | 可用性     | 高（允许部分失败）     | 低（一个节点故障全局阻塞）   |
// | 实现复杂度 | 高（需设计补偿逻辑）   | 低（数据库原生支持）         |
// | 适用场景   | 微服务、长事务         | 单体应用、短事务            |
//
// 关键学习点：
// 1. 补偿操作必须幂等（使用idempotency_key）
// 2. 补偿失败需要人工介入（记录日志、告警）
// 3. Saga期间数据可能不一致（业务需容忍）
// 4. 生产环境建议使用DTM等成熟框架（支持故障恢复）
