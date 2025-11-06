package inventory

import "context"

// Repository 库存仓储接口（领域层定义）
//
// 教学要点：
// 1. 依赖倒置原则（高层定义接口，低层实现）
// 2. 为什么需要独立的Repository？
//   - 领域层不依赖具体的数据库实现
//   - 便于单元测试（Mock）
//   - 支持多种存储方式（MySQL、Redis）
type Repository interface {
	// GetByBookID 根据图书ID获取库存
	GetByBookID(ctx context.Context, bookID uint) (*Inventory, error)

	// BatchGetByBookIDs 批量获取库存
	BatchGetByBookIDs(ctx context.Context, bookIDs []uint) (map[uint]*Inventory, error)

	// Create 创建库存记录
	Create(ctx context.Context, inv *Inventory) error

	// Update 更新库存
	Update(ctx context.Context, inv *Inventory) error

	// DeductStock 扣减库存（使用数据库悲观锁）
	// 教学要点：SELECT FOR UPDATE
	DeductStock(ctx context.Context, bookID uint, quantity int, orderID uint) error

	// ReleaseStock 释放库存
	ReleaseStock(ctx context.Context, bookID uint, quantity int, orderID uint, reason string) error

	// RestockInventory 补充库存
	RestockInventory(ctx context.Context, bookID uint, quantity int) error
}

// LogRepository 库存日志仓储接口
type LogRepository interface {
	// Create 创建日志
	Create(ctx context.Context, log *InventoryLog) error

	// ListByBookID 查询指定图书的库存日志
	ListByBookID(ctx context.Context, bookID uint, page, pageSize int) ([]*InventoryLog, int64, error)

	// ListByOrderID 查询指定订单的库存日志
	ListByOrderID(ctx context.Context, orderID uint) ([]*InventoryLog, error)
}
