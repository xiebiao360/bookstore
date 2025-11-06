package payment

import "time"

// Payment 支付聚合根
//
// 教学要点：
// 1. 支付是独立的聚合根（与Order解耦）
// 2. 支付流水号（PaymentNo）vs 订单号（OrderID）：
//   - PaymentNo：支付系统内部流水号
//   - OrderID：关联的订单ID（外键）
//
// 3. 第三方支付流水号（ThirdPartyNo）：
//   - Mock模式：为空
//   - 真实支付：支付宝/微信返回的交易号
type Payment struct {
	ID            uint          `gorm:"primaryKey;comment:支付ID"`
	PaymentNo     string        `gorm:"uniqueIndex;size:32;not null;comment:支付流水号"`
	OrderID       uint          `gorm:"uniqueIndex;not null;comment:订单ID"`
	Amount        int64         `gorm:"not null;comment:支付金额（分）"`
	Status        PaymentStatus `gorm:"type:tinyint;not null;default:1;index;comment:支付状态"`
	PaymentMethod string        `gorm:"size:20;not null;comment:支付方式"`
	ThirdPartyNo  string        `gorm:"size:64;comment:第三方支付流水号"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PaymentStatus 支付状态枚举
//
// 教学要点：
// 支付状态机比订单简单（只有4种状态）
type PaymentStatus int

const (
	PaymentStatusPending  PaymentStatus = 1 // 待支付
	PaymentStatusPaid     PaymentStatus = 2 // 已支付
	PaymentStatusRefunded PaymentStatus = 3 // 已退款
	PaymentStatusFailed   PaymentStatus = 4 // 失败
)

func (s PaymentStatus) String() string {
	switch s {
	case PaymentStatusPending:
		return "待支付"
	case PaymentStatusPaid:
		return "已支付"
	case PaymentStatusRefunded:
		return "已退款"
	case PaymentStatusFailed:
		return "失败"
	default:
		return "未知状态"
	}
}

// TableName 指定表名
func (Payment) TableName() string {
	return "payments"
}

// CanRefund 判断是否可以退款
//
// 教学要点：
// 业务规则：只有已支付的才能退款
func (p *Payment) CanRefund() bool {
	return p.Status == PaymentStatusPaid
}

// UpdateStatus 更新支付状态
func (p *Payment) UpdateStatus(status PaymentStatus) error {
	// 简化：允许任意状态转换（真实场景需要状态机）
	p.Status = status
	return nil
}
