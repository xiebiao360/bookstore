package order

import (
	"context"
)

// Repository 订单仓储接口(依赖倒置原则)
// 教学要点:
// 1. 由domain层定义接口,infrastructure层实现
// 2. 支持事务操作(通过context传递事务)
type Repository interface {
	// Create 创建订单(包含订单明细)
	// 教学要点:订单和明细必须在同一事务中创建
	Create(ctx context.Context, order *Order) error

	// FindByID 根据ID查找订单(包含订单明细)
	FindByID(ctx context.Context, id uint) (*Order, error)

	// FindByOrderNo 根据订单号查找订单
	FindByOrderNo(ctx context.Context, orderNo string) (*Order, error)

	// Update 更新订单(主要用于状态更新)
	Update(ctx context.Context, order *Order) error

	// ListByUserID 查询用户的订单列表
	// 教学要点:支持分页,避免一次性查询大量数据
	ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*Order, int64, error)
}
