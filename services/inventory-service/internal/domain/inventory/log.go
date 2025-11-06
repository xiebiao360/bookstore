package inventory

import "time"

// InventoryLog 库存变更日志（领域模型）
//
// 教学要点：
// 1. 为什么需要库存日志？
//   - 审计需求：所有库存变更必须可追溯
//   - 对账需求：库存与订单数据核对
//   - 排查需求：异常库存问题定位
//
// 2. 日志设计原则
//   - 只增不改（Append-Only）
//   - 记录变更前后状态
//   - 记录关联业务ID（订单ID）
type InventoryLog struct {
	// 主键ID
	ID uint `gorm:"primaryKey" json:"id"`

	// 图书ID
	BookID uint `gorm:"index:idx_book_id;not null" json:"book_id"`

	// 变更类型
	// DEDUCT: 扣减库存（支付成功）
	// RELEASE: 释放库存（订单取消、支付失败）
	// RESTOCK: 补充库存（补货）
	// LOCK: 锁定库存（下单）
	// UNLOCK: 解锁库存（订单取消）
	ChangeType ChangeType `gorm:"type:varchar(20);not null" json:"change_type"`

	// 变更数量（正数=增加，负数=减少）
	Quantity int `gorm:"not null" json:"quantity"`

	// 变更前库存
	BeforeStock int `gorm:"not null" json:"before_stock"`

	// 变更后库存
	AfterStock int `gorm:"not null" json:"after_stock"`

	// 关联订单ID（可选）
	OrderID uint `gorm:"index:idx_order_id" json:"order_id,omitempty"`

	// 备注
	Remark string `gorm:"type:varchar(255)" json:"remark,omitempty"`

	// 创建时间
	CreatedAt time.Time `gorm:"index:idx_created_at" json:"created_at"`
}

// TableName 指定表名
func (InventoryLog) TableName() string {
	return "inventory_logs"
}

// ChangeType 库存变更类型
type ChangeType string

const (
	ChangeTypeDeduct  ChangeType = "DEDUCT"  // 扣减
	ChangeTypeRelease ChangeType = "RELEASE" // 释放
	ChangeTypeRestock ChangeType = "RESTOCK" // 补充
	ChangeTypeLock    ChangeType = "LOCK"    // 锁定
	ChangeTypeUnlock  ChangeType = "UNLOCK"  // 解锁
)

// NewDeductLog 创建扣减日志
func NewDeductLog(bookID uint, quantity int, before, after int, orderID uint) *InventoryLog {
	return &InventoryLog{
		BookID:      bookID,
		ChangeType:  ChangeTypeDeduct,
		Quantity:    -quantity, // 负数表示减少
		BeforeStock: before,
		AfterStock:  after,
		OrderID:     orderID,
	}
}

// NewReleaseLog 创建释放日志
func NewReleaseLog(bookID uint, quantity int, before, after int, orderID uint, reason string) *InventoryLog {
	return &InventoryLog{
		BookID:      bookID,
		ChangeType:  ChangeTypeRelease,
		Quantity:    quantity, // 正数表示增加
		BeforeStock: before,
		AfterStock:  after,
		OrderID:     orderID,
		Remark:      reason,
	}
}

// NewRestockLog 创建补货日志
func NewRestockLog(bookID uint, quantity int, before, after int) *InventoryLog {
	return &InventoryLog{
		BookID:      bookID,
		ChangeType:  ChangeTypeRestock,
		Quantity:    quantity, // 正数表示增加
		BeforeStock: before,
		AfterStock:  after,
	}
}
