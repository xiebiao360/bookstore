package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/xiebiao/bookstore/services/order-service/internal/domain/order"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// orderRepository 订单仓储MySQL实现
//
// 教学要点：
// 1. 小写命名（私有）：只暴露接口，隐藏实现
// 2. 依赖注入：通过构造函数传入*gorm.DB
// 3. 实现接口：确保编译期检查接口实现
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储实例
//
// 教学要点：
// 为什么使用构造函数模式？
// - 控制实例创建（可以添加初始化逻辑）
// - 返回接口类型（便于测试和替换实现）
// - 清晰的依赖关系（显式传入db）
func NewOrderRepository(db *gorm.DB) order.Repository {
	return &orderRepository{db: db}
}

// Create 创建订单
//
// 教学要点：
// 1. 事务处理：订单和明细必须同时插入成功
//   - 使用db.Transaction()包裹
//   - 任一步骤失败，自动回滚
//
// 2. GORM自动填充：
//   - ID：自增主键，插入后自动回填
//   - CreatedAt/UpdatedAt：自动设置时间
//
// 3. 关联插入：
//   - db.Create(&order)会自动插入order.Items
//   - GORM识别外键关系，先插order，再插items
//
// DO vs DON'T:
// ❌ DON'T: 手动管理事务（Begin/Commit/Rollback）
//
//	db.Begin()
//	db.Create(&order)
//	db.Create(&items)
//	db.Commit() // 忘记Rollback会导致连接泄漏
//
// ✅ DO: 使用db.Transaction()自动管理
//
//	db.Transaction(func(tx *gorm.DB) error {
//	  return tx.Create(&order).Error
//	})
func (r *orderRepository) Create(ctx context.Context, o *order.Order) error {
	// 事务执行
	// 教学要点：
	// Transaction函数签名：func(fc func(tx *gorm.DB) error) error
	// - 成功：返回nil，自动Commit
	// - 失败：返回error，自动Rollback
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 插入订单（包含关联的Items）
		// GORM会自动：
		// 1. INSERT INTO orders (...)
		// 2. INSERT INTO order_items (...) VALUES (...), (...)
		if err := tx.Create(o).Error; err != nil {
			return fmt.Errorf("创建订单失败: %w", err)
		}
		return nil
	})
}

// FindByID 根据ID查询订单
//
// 教学要点：
// 1. Preload("Items")：预加载关联数据
//   - 不使用：查询order后，Items为空切片
//   - 使用：自动执行JOIN或额外查询，填充Items
//
// 2. First vs Find：
//   - First：查询单条，未找到返回ErrRecordNotFound
//   - Find：查询多条，未找到返回空切片
//
// 3. 错误处理：
//   - gorm.ErrRecordNotFound → order.ErrOrderNotFound
//   - 其他错误 → 原样返回
func (r *orderRepository) FindByID(ctx context.Context, id uint) (*order.Order, error) {
	var o order.Order

	// 查询订单并预加载明细
	// SQL执行：
	// 1. SELECT * FROM orders WHERE id = ? AND deleted_at IS NULL
	// 2. SELECT * FROM order_items WHERE order_id = ?
	err := r.db.WithContext(ctx).
		Preload("Items").   // 预加载明细
		First(&o, id).Error // 根据主键查询

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}

	return &o, nil
}

// FindByOrderNo 根据订单号查询
//
// 教学要点：
// Where vs First参数：
// - First(&o, id)：快捷方式，根据主键查询
// - Where("order_no = ?", orderNo).First(&o)：自定义条件
func (r *orderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*order.Order, error) {
	var o order.Order

	err := r.db.WithContext(ctx).
		Preload("Items").
		Where("order_no = ?", orderNo).
		First(&o).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, fmt.Errorf("查询订单失败: %w", err)
	}

	return &o, nil
}

// FindByUserID 查询用户的订单列表
//
// 教学要点：
// 1. 分页查询：
//   - Offset：跳过前N条
//   - Limit：取M条
//   - 计算：page=2, pageSize=10 → Offset=10, Limit=10
//
// 2. 条件筛选：
//   - status=0：查询所有状态
//   - status>0：筛选特定状态
//
// 3. 排序：
//   - Order("created_at DESC")：最新订单在前
//
// 4. 查询总数：
//   - Count(&total)：不受Offset/Limit影响
//   - 需要单独查询（性能优化：可以缓存）
//
// SQL示例：
// SELECT * FROM orders WHERE user_id=1 AND status=1 ORDER BY created_at DESC LIMIT 10 OFFSET 0
// SELECT COUNT(*) FROM orders WHERE user_id=1 AND status=1
func (r *orderRepository) FindByUserID(
	ctx context.Context,
	userID uint,
	page, pageSize int,
	status order.OrderStatus,
) ([]*order.Order, int64, error) {
	var orders []*order.Order
	var total int64

	// 构建查询条件
	query := r.db.WithContext(ctx).Model(&order.Order{}).
		Where("user_id = ?", userID)

	// 状态筛选（0表示查询所有）
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	// 查询总数（分页元信息）
	// 教学要点：
	// Count必须在Offset/Limit之前执行
	// 否则会计算分页后的数量
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询订单总数失败: %w", err)
	}

	// 分页查询
	// 教学要点：
	// 计算Offset：(page - 1) * pageSize
	// - page=1, pageSize=10 → Offset=0
	// - page=2, pageSize=10 → Offset=10
	offset := (page - 1) * pageSize
	err := query.
		Preload("Items").         // 预加载明细
		Order("created_at DESC"). // 按创建时间降序
		Offset(offset).
		Limit(pageSize).
		Find(&orders).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询订单列表失败: %w", err)
	}

	return orders, total, nil
}

// Update 更新订单
//
// 教学要点：
// 1. Save vs Updates：
//   - Save：更新所有字段（包括零值）
//   - Updates：只更新非零值字段
//   - 本例使用Save，因为Status可能为0
//
// 2. 不更新OrderItem：
//   - 订单创建后明细不可修改（业务规则）
//   - 如需修改，应该取消订单重新下单
//
// 3. 乐观锁（可选）：
//   - 添加version字段
//   - 更新时WHERE version = ?
//   - 防止并发更新冲突
func (r *orderRepository) Update(ctx context.Context, o *order.Order) error {
	// Save会更新所有字段
	// SQL: UPDATE orders SET order_no=?, user_id=?, total=?, status=?, updated_at=? WHERE id=?
	result := r.db.WithContext(ctx).Save(o)

	if result.Error != nil {
		return fmt.Errorf("更新订单失败: %w", result.Error)
	}

	// 检查是否更新成功（RowsAffected=0表示订单不存在）
	if result.RowsAffected == 0 {
		return order.ErrOrderNotFound
	}

	return nil
}

// UpdateStatus 仅更新状态
//
// 教学要点：
// 为什么单独实现UpdateStatus？
// 1. 性能优化：只更新status字段，减少数据传输
// 2. 并发安全：避免覆盖其他字段的并发修改
// 3. 语义清晰：明确表示只变更状态
//
// DO vs DON'T:
// ❌ DON'T: 查询完整订单 → 修改Status → Save整个订单
//
//	order, _ := repo.FindByID(id)
//	order.Status = newStatus
//	repo.Update(order) // 可能覆盖并发修改的其他字段
//
// ✅ DO: 只更新status字段
//
//	repo.UpdateStatus(id, newStatus)
func (r *orderRepository) UpdateStatus(ctx context.Context, id uint, status order.OrderStatus) error {
	// Model(&order.Order{}).Where("id = ?", id)：指定更新目标
	// Update("status", status)：只更新status字段
	// SQL: UPDATE orders SET status=?, updated_at=? WHERE id=?
	result := r.db.WithContext(ctx).
		Model(&order.Order{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("更新订单状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return order.ErrOrderNotFound
	}

	return nil
}

// Delete 删除订单（软删除）
//
// 教学要点：
// 1. GORM软删除：
//   - Order结构体需要嵌入gorm.DeletedAt字段
//   - Delete操作：UPDATE orders SET deleted_at=? WHERE id=?
//   - 查询操作：自动添加WHERE deleted_at IS NULL
//
// 2. 级联删除：
//   - OrderItem定义了constraint:OnDelete:CASCADE
//   - 删除Order时自动软删除Items
//
// 3. Select("Items")：显式级联删除关联
func (r *orderRepository) Delete(ctx context.Context, id uint) error {
	// Clauses(clause.Returning{})：返回被删除的记录（MySQL 8.0+）
	// Select("Items")：级联删除关联的OrderItem
	result := r.db.WithContext(ctx).
		Select(clause.Associations). // 级联删除关联
		Delete(&order.Order{}, id)

	if result.Error != nil {
		return fmt.Errorf("删除订单失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return order.ErrOrderNotFound
	}

	return nil
}

// ============================================================
// ItemRepository实现
// ============================================================

type orderItemRepository struct {
	db *gorm.DB
}

// NewOrderItemRepository 创建订单明细仓储
func NewOrderItemRepository(db *gorm.DB) order.ItemRepository {
	return &orderItemRepository{db: db}
}

// FindByOrderID 查询订单的所有明细
func (r *orderItemRepository) FindByOrderID(ctx context.Context, orderID uint) ([]*order.OrderItem, error) {
	var items []*order.OrderItem

	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("查询订单明细失败: %w", err)
	}

	return items, nil
}

// FindByBookID 查询某本书的所有订单明细（用于统计销量）
//
// 教学要点：
// 为什么需要limit参数？
// - 某本畅销书可能有成千上万条订单明细
// - 通常只需要最近的N条（如最近100条）
// - 防止OOM
func (r *orderItemRepository) FindByBookID(ctx context.Context, bookID uint, limit int) ([]*order.OrderItem, error) {
	var items []*order.OrderItem

	query := r.db.WithContext(ctx).
		Where("book_id = ?", bookID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("查询图书订单明细失败: %w", err)
	}

	return items, nil
}
