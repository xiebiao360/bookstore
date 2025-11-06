package order

import "errors"

// 领域层错误定义
//
// 教学要点：
// 1. 为什么在domain层定义错误？
//   - 领域驱动设计（DDD）：错误是业务规则的一部分
//   - 解耦：应用层和基础设施层都可以使用，避免循环依赖
//   - 统一：所有订单相关错误集中管理
//
// 2. 使用errors.New vs 自定义Error类型
//   - errors.New：简单场景，只需错误消息
//   - 自定义类型：需要携带错误码、上下文信息
//   - 本项目：Phase 1使用errors.New，Phase 2会扩展为自定义类型
//
// 3. 命名约定：Err前缀（Go社区惯例）
var (
	// ErrOrderNotFound 订单不存在
	// 场景：根据ID查询订单时未找到
	ErrOrderNotFound = errors.New("订单不存在")

	// ErrInvalidStatusTransition 非法的状态转换
	// 场景：尝试将订单从"已完成"改为"待支付"
	// 教学要点：这是状态机的核心约束
	ErrInvalidStatusTransition = errors.New("非法的订单状态转换")

	// ErrOrderAlreadyCancelled 订单已取消
	// 场景：尝试对已取消的订单执行操作
	ErrOrderAlreadyCancelled = errors.New("订单已取消")

	// ErrOrderNotCancellable 订单不可取消
	// 场景：已发货或已完成的订单不允许取消
	ErrOrderNotCancellable = errors.New("订单不可取消")

	// ErrInsufficientStock 库存不足
	// 场景：下单时调用inventory-service返回库存不足
	// 教学要点：
	// - 这是分布式系统的典型错误
	// - Phase 2会通过Saga模式处理（扣减失败自动补偿）
	ErrInsufficientStock = errors.New("库存不足")

	// ErrInvalidOrderAmount 订单金额异常
	// 场景：计算的总金额为0或负数
	// 防御性编程：防止恶意篡改价格
	ErrInvalidOrderAmount = errors.New("订单金额异常")

	// ErrEmptyOrderItems 订单明细为空
	// 场景：创建订单时未传入任何商品
	ErrEmptyOrderItems = errors.New("订单明细不能为空")

	// ErrInvalidQuantity 商品数量异常
	// 场景：数量 <= 0 或超过限制（如单次最多购买99件）
	ErrInvalidQuantity = errors.New("商品数量异常")

	// ErrOrderPermissionDenied 无权限操作订单
	// 场景：用户A尝试取消用户B的订单
	// 教学要点：安全设计，防止越权操作
	ErrOrderPermissionDenied = errors.New("无权限操作该订单")

	// ErrPaymentFailed 支付失败
	// 场景：调用payment-service支付时失败
	// Phase 2会细化为：余额不足、支付超时、渠道异常等
	ErrPaymentFailed = errors.New("支付失败")
)

// IsNotFoundError 判断是否为"未找到"类错误
//
// 教学要点：
// 为什么需要此函数？
// - 在HTTP层需要区分错误类型，返回不同状态码
//   - 未找到 → 404 Not Found
//   - 业务错误 → 400 Bad Request
//   - 系统错误 → 500 Internal Server Error
//
// - 使用errors.Is可以支持错误包装（errors.Wrap）
//
// DO vs DON'T:
// ❌ DON'T: err.Error() == "订单不存在"（字符串比较，脆弱）
// ✅ DO: errors.Is(err, ErrOrderNotFound)（语义化比较）
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrOrderNotFound)
}

// IsBusinessError 判断是否为业务逻辑错误（而非系统错误）
//
// 教学要点：
// 业务错误 vs 系统错误的处理策略：
// - 业务错误：返回给用户，提示如何解决（如"库存不足，请减少数量"）
// - 系统错误：记录日志，返回通用提示（如"系统繁忙，请稍后重试"）
func IsBusinessError(err error) bool {
	businessErrors := []error{
		ErrInvalidStatusTransition,
		ErrOrderAlreadyCancelled,
		ErrOrderNotCancellable,
		ErrInsufficientStock,
		ErrInvalidOrderAmount,
		ErrEmptyOrderItems,
		ErrInvalidQuantity,
		ErrOrderPermissionDenied,
		ErrPaymentFailed,
	}

	for _, e := range businessErrors {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
