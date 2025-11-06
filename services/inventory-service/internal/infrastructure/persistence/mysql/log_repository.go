package mysql

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/xiebiao/bookstore/services/inventory-service/internal/domain/inventory"
)

// logRepository 库存日志仓储实现
type logRepository struct {
	db *gorm.DB
}

// NewLogRepository 创建日志仓储实例
func NewLogRepository(db *gorm.DB) inventory.LogRepository {
	return &logRepository{db: db}
}

// Create 创建日志
func (r *logRepository) Create(ctx context.Context, log *inventory.InventoryLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("创建库存日志失败: %w", err)
	}
	return nil
}

// ListByBookID 查询指定图书的库存日志
func (r *logRepository) ListByBookID(ctx context.Context, bookID uint, page, pageSize int) ([]*inventory.InventoryLog, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询总数
	var total int64
	if err := r.db.WithContext(ctx).Model(&inventory.InventoryLog{}).
		Where("book_id = ?", bookID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询日志总数失败: %w", err)
	}

	// 分页查询
	var logs []*inventory.InventoryLog
	offset := (page - 1) * pageSize

	if err := r.db.WithContext(ctx).
		Where("book_id = ?", bookID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询库存日志失败: %w", err)
	}

	return logs, total, nil
}

// ListByOrderID 查询指定订单的库存日志
func (r *logRepository) ListByOrderID(ctx context.Context, orderID uint) ([]*inventory.InventoryLog, error) {
	var logs []*inventory.InventoryLog

	if err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("查询订单库存日志失败: %w", err)
	}

	return logs, nil
}
