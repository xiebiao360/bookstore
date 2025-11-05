package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/xiebiao/bookstore/internal/domain/order"
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// orderRepository 订单仓储实现(MySQL)
// 教学要点:
// 1. Order和OrderItem是聚合关系,必须一起保存
// 2. 查询时使用Preload预加载明细,避免N+1问题
// 3. 事务通过context传递
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建订单仓储
func NewOrderRepository(db *gorm.DB) order.Repository {
	return &orderRepository{db: db}
}

// Create 创建订单
// 教学要点:
// 1. GORM会自动保存关联的Items(通过foreignKey)
// 2. 必须在事务中调用(通过getDB从context获取事务DB)
func (r *orderRepository) Create(ctx context.Context, o *order.Order) error {
	// 1. 领域实体 → GORM模型
	model := toOrderModel(o)

	// 2. 插入数据库(包含订单明细)
	// 教学要点:GORM的FullSaveAssociations会自动保存Items
	db := r.getDB(ctx)
	if err := db.Create(model).Error; err != nil {
		return apperrors.Wrap(err, "创建订单失败")
	}

	// 3. 回填自增ID
	o.ID = model.ID
	for i := range o.Items {
		o.Items[i].ID = model.Items[i].ID
	}

	return nil
}

// FindByID 根据ID查找订单
// 教学要点:使用Preload预加载Items,避免N+1查询
func (r *orderRepository) FindByID(ctx context.Context, id uint) (*order.Order, error) {
	var model OrderModel
	db := r.getDB(ctx)

	// Preload("Items")会执行:
	// 1. SELECT * FROM orders WHERE id = ?
	// 2. SELECT * FROM order_items WHERE order_id IN (?)
	err := db.Preload("Items").First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, apperrors.Wrap(err, "查询订单失败")
	}

	return toOrderEntity(&model), nil
}

// FindByOrderNo 根据订单号查找订单
func (r *orderRepository) FindByOrderNo(ctx context.Context, orderNo string) (*order.Order, error) {
	var model OrderModel
	db := r.getDB(ctx)
	err := db.Preload("Items").Where("order_no = ?", orderNo).First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, order.ErrOrderNotFound
		}
		return nil, apperrors.Wrap(err, "查询订单失败")
	}

	return toOrderEntity(&model), nil
}

// Update 更新订单
// 教学要点:主要用于状态更新,不更新Items
func (r *orderRepository) Update(ctx context.Context, o *order.Order) error {
	db := r.getDB(ctx)

	// 只更新Status和UpdatedAt
	result := db.Model(&OrderModel{}).Where("id = ?", o.ID).Updates(map[string]interface{}{
		"status":     int(o.Status),
		"updated_at": o.UpdatedAt,
	})

	if result.Error != nil {
		return apperrors.Wrap(result.Error, "更新订单失败")
	}

	if result.RowsAffected == 0 {
		return order.ErrOrderNotFound
	}

	return nil
}

// ListByUserID 查询用户的订单列表
func (r *orderRepository) ListByUserID(ctx context.Context, userID uint, page, pageSize int) ([]*order.Order, int64, error) {
	var models []OrderModel
	var total int64

	db := r.getDB(ctx)
	query := db.Model(&OrderModel{}).Where("user_id = ?", userID)

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询订单总数失败")
	}

	// 分页查询(包含明细)
	offset := (page - 1) * pageSize
	err := query.Preload("Items").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, apperrors.Wrap(err, "查询订单列表失败")
	}

	// 转换为领域实体
	orders := make([]*order.Order, len(models))
	for i, model := range models {
		orders[i] = toOrderEntity(&model)
	}

	return orders, total, nil
}

// =========================================
// 辅助函数:模型转换
// =========================================

// toOrderModel 领域实体 → GORM模型
func toOrderModel(o *order.Order) *OrderModel {
	items := make([]OrderItemModel, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemModel{
			ID:       item.ID,
			OrderID:  item.OrderID,
			BookID:   item.BookID,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	return &OrderModel{
		ID:        o.ID,
		OrderNo:   o.OrderNo,
		UserID:    o.UserID,
		Total:     o.Total,
		Status:    int(o.Status),
		Items:     items,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

// toOrderEntity GORM模型 → 领域实体
func toOrderEntity(model *OrderModel) *order.Order {
	items := make([]order.OrderItem, len(model.Items))
	for i, item := range model.Items {
		items[i] = order.OrderItem{
			ID:       item.ID,
			OrderID:  item.OrderID,
			BookID:   item.BookID,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	return &order.Order{
		ID:        model.ID,
		OrderNo:   model.OrderNo,
		UserID:    model.UserID,
		Total:     model.Total,
		Status:    order.OrderStatus(model.Status),
		Items:     items,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

// getDB 从context获取事务DB,如果没有则使用默认DB
// 教学要点:事务传递机制
func (r *orderRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		return tx
	}
	return r.db
}
