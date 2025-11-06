package inventory

import "time"

// Inventory 库存实体（领域模型）
//
// 教学要点：
// 1. 库存实体的核心字段设计
//   - Stock：当前可用库存
//   - LockedStock：已锁定库存（待支付订单）
//   - TotalStock：总库存 = Stock + LockedStock
//
//  2. 为什么需要LockedStock？
//     场景：用户下单后15分钟内需要完成支付
//     - 如果直接扣减Stock，用户不支付会占用库存
//     - 使用锁定机制：下单锁定 → 支付扣减 → 超时释放
//
//  3. Phase 1 vs Phase 2 对比
//     Phase 1：books表的stock字段（简单扣减）
//     Phase 2：独立的inventory表（支持锁定机制）
type Inventory struct {
	// 图书ID（主键）
	BookID uint `gorm:"primaryKey;column:book_id" json:"book_id"`

	// 可用库存
	// 教学要点：下单时扣减此字段，锁定到LockedStock
	Stock int `gorm:"not null;default:0;index:idx_stock" json:"stock"`

	// 已锁定库存（待支付订单）
	// 教学要点：支付成功后扣减，支付失败/超时后释放回Stock
	LockedStock int `gorm:"not null;default:0" json:"locked_stock"`

	// 总库存
	// 教学要点：TotalStock = Stock + LockedStock，用于库存盘点
	TotalStock int `gorm:"not null;default:0" json:"total_stock"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Inventory) TableName() string {
	return "inventory"
}

// Validate 验证库存实体
func (i *Inventory) Validate() error {
	if i.BookID == 0 {
		return ErrInvalidBookID
	}

	if i.Stock < 0 {
		return ErrNegativeStock
	}

	if i.LockedStock < 0 {
		return ErrNegativeLockedStock
	}

	// 检查总库存一致性
	if i.TotalStock != i.Stock+i.LockedStock {
		return ErrInconsistentTotalStock
	}

	return nil
}

// CanDeduct 判断是否可以扣减库存
// 教学要点：扣减前的业务规则验证
func (i *Inventory) CanDeduct(quantity int) bool {
	return i.Stock >= quantity && quantity > 0
}

// CanLock 判断是否可以锁定库存
func (i *Inventory) CanLock(quantity int) bool {
	return i.Stock >= quantity && quantity > 0
}

// IsLowStock 判断是否低库存（需要告警）
func (i *Inventory) IsLowStock(threshold int) bool {
	return i.Stock <= threshold && i.Stock > 0
}

// IsOutOfStock 判断是否缺货
func (i *Inventory) IsOutOfStock() bool {
	return i.Stock <= 0
}
