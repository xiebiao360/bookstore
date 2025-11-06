package mysql

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiebiao/bookstore/services/inventory-service/internal/domain/inventory"
)

// inventoryRepository MySQL库存仓储实现
//
// 教学要点：
// 1. MySQL作为持久化存储
//   - 数据可靠性（ACID）
//   - 对账需求（与Redis对比）
//   - 历史数据查询
//
// 2. 与Redis配合使用
//   - Redis：实时库存（高性能）
//   - MySQL：持久化存储（高可靠）
//   - 定时同步：Redis → MySQL
type inventoryRepository struct {
	db *gorm.DB
}

// NewInventoryRepository 创建库存仓储实例
func NewInventoryRepository(db *gorm.DB) inventory.Repository {
	return &inventoryRepository{db: db}
}

// GetByBookID 根据图书ID获取库存
func (r *inventoryRepository) GetByBookID(ctx context.Context, bookID uint) (*inventory.Inventory, error) {
	var inv inventory.Inventory

	if err := r.db.WithContext(ctx).Where("book_id = ?", bookID).First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, inventory.ErrInventoryNotFound
		}
		return nil, fmt.Errorf("查询库存失败: %w", err)
	}

	return &inv, nil
}

// BatchGetByBookIDs 批量获取库存
func (r *inventoryRepository) BatchGetByBookIDs(ctx context.Context, bookIDs []uint) (map[uint]*inventory.Inventory, error) {
	if len(bookIDs) == 0 {
		return make(map[uint]*inventory.Inventory), nil
	}

	var invs []*inventory.Inventory

	if err := r.db.WithContext(ctx).Where("book_id IN ?", bookIDs).Find(&invs).Error; err != nil {
		return nil, fmt.Errorf("批量查询库存失败: %w", err)
	}

	// 转换为map
	result := make(map[uint]*inventory.Inventory, len(invs))
	for _, inv := range invs {
		result[inv.BookID] = inv
	}

	return result, nil
}

// Create 创建库存记录
func (r *inventoryRepository) Create(ctx context.Context, inv *inventory.Inventory) error {
	if err := inv.Validate(); err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Create(inv).Error; err != nil {
		return fmt.Errorf("创建库存失败: %w", err)
	}

	return nil
}

// Update 更新库存
func (r *inventoryRepository) Update(ctx context.Context, inv *inventory.Inventory) error {
	if err := inv.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Model(&inventory.Inventory{}).
		Where("book_id = ?", inv.BookID).
		Updates(inv)

	if err := result.Error; err != nil {
		return fmt.Errorf("更新库存失败: %w", err)
	}

	if result.RowsAffected == 0 {
		return inventory.ErrInventoryNotFound
	}

	return nil
}

// DeductStock 扣减库存（使用悲观锁）
//
// 教学要点：
// 1. SELECT FOR UPDATE悲观锁
//   - 锁定查询的行，防止其他事务修改
//   - 事务提交后释放锁
//
// 2. 完整的扣减流程
//
//   - 锁定库存记录
//
//   - 检查库存是否充足
//
//   - 扣减库存
//
//   - 创建库存日志
//
//     3. DO vs DON'T
//     ✅ DO：使用事务 + SELECT FOR UPDATE
//     ❌ DON'T：直接UPDATE（并发问题）
func (r *inventoryRepository) DeductStock(ctx context.Context, bookID uint, quantity int, orderID uint) error {
	// 使用事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv inventory.Inventory

		// 步骤1：锁定库存记录（SELECT FOR UPDATE）
		// 教学要点：其他事务会等待此锁释放
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("book_id = ?", bookID).
			First(&inv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return inventory.ErrInventoryNotFound
			}
			return fmt.Errorf("锁定库存失败: %w", err)
		}

		// 步骤2：检查库存是否充足
		if !inv.CanDeduct(quantity) {
			return inventory.ErrInsufficientStock
		}

		// 步骤3：扣减库存
		beforeStock := inv.Stock
		inv.Stock -= quantity
		inv.TotalStock = inv.Stock + inv.LockedStock

		if err := tx.Save(&inv).Error; err != nil {
			return fmt.Errorf("扣减库存失败: %w", err)
		}

		// 步骤4：创建库存日志
		log := inventory.NewDeductLog(bookID, quantity, beforeStock, inv.Stock, orderID)
		if err := tx.Create(log).Error; err != nil {
			return fmt.Errorf("创建库存日志失败: %w", err)
		}

		return nil
	})
}

// ReleaseStock 释放库存
func (r *inventoryRepository) ReleaseStock(ctx context.Context, bookID uint, quantity int, orderID uint, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv inventory.Inventory

		// 锁定库存记录
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("book_id = ?", bookID).
			First(&inv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return inventory.ErrInventoryNotFound
			}
			return fmt.Errorf("锁定库存失败: %w", err)
		}

		// 释放库存
		beforeStock := inv.Stock
		inv.Stock += quantity
		inv.TotalStock = inv.Stock + inv.LockedStock

		if err := tx.Save(&inv).Error; err != nil {
			return fmt.Errorf("释放库存失败: %w", err)
		}

		// 创建库存日志
		log := inventory.NewReleaseLog(bookID, quantity, beforeStock, inv.Stock, orderID, reason)
		if err := tx.Create(log).Error; err != nil {
			return fmt.Errorf("创建库存日志失败: %w", err)
		}

		return nil
	})
}

// RestockInventory 补充库存
func (r *inventoryRepository) RestockInventory(ctx context.Context, bookID uint, quantity int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inv inventory.Inventory

		// 锁定库存记录
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("book_id = ?", bookID).
			First(&inv).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return inventory.ErrInventoryNotFound
			}
			return fmt.Errorf("锁定库存失败: %w", err)
		}

		// 补充库存
		beforeStock := inv.Stock
		inv.Stock += quantity
		inv.TotalStock = inv.Stock + inv.LockedStock

		if err := tx.Save(&inv).Error; err != nil {
			return fmt.Errorf("补充库存失败: %w", err)
		}

		// 创建库存日志
		log := inventory.NewRestockLog(bookID, quantity, beforeStock, inv.Stock)
		if err := tx.Create(log).Error; err != nil {
			return fmt.Errorf("创建库存日志失败: %w", err)
		}

		return nil
	})
}
