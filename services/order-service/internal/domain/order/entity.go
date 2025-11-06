package order

import (
	"time"
)

// Order 订单聚合根
//
// 教学要点：
// 1. 订单是聚合根（Aggregate Root），管理OrderItem集合
// 2. 订单号（OrderNo）vs 订单ID（ID）：
//   - ID：数据库主键（自增）
//   - OrderNo：业务主键（对外展示，如"20251106123456789"）
//
// 3. 为什么总金额存分（int64）而非元（float64）？
//   - 避免浮点数精度问题（0.1+0.2 != 0.3）
//   - 金融系统的行业惯例
//
// 设计模式：
// - 聚合模式（Aggregate）：Order + OrderItem是一个事务边界
// - 实体模式（Entity）：有唯一标识（ID）的领域对象
type Order struct {
	ID        uint        `gorm:"primaryKey;comment:订单ID"`
	OrderNo   string      `gorm:"uniqueIndex;size:32;not null;comment:订单号"`
	UserID    uint        `gorm:"index;not null;comment:用户ID"`
	Total     int64       `gorm:"not null;comment:总金额（分）"`
	Status    OrderStatus `gorm:"type:tinyint;not null;default:1;index;comment:订单状态"`
	CreatedAt time.Time   `gorm:"comment:创建时间"`
	UpdatedAt time.Time   `gorm:"comment:更新时间"`

	// Items 订单明细（聚合内的实体集合）
	// 教学要点：
	// - GORM关联关系：一对多（Order has many OrderItem）
	// - foreignKey:OrderID：指定外键字段
	// - constraint:OnDelete:CASCADE：级联删除（删订单时自动删明细）
	Items []OrderItem `gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// OrderItem 订单明细（值对象）
//
// 教学要点：
// 1. 为什么单价（Price）存在OrderItem中？
//   - 价格快照：记录下单时的价格，防止后续改价影响历史订单
//   - 不依赖Book表：即使Book被删除，订单明细仍可查询
//
// 2. 为什么不用外键关联Book？
//   - 跨服务调用：Phase 2中Book在catalog-service，订单在order-service
//   - 数据冗余换性能：避免每次查询订单都要跨服务查Book
//
// 值对象特征：
// - 无独立生命周期，依附于Order
// - 通过属性相等判断，而非ID
type OrderItem struct {
	ID        uint   `gorm:"primaryKey;comment:明细ID"`
	OrderID   uint   `gorm:"index;not null;comment:订单ID"`
	BookID    uint   `gorm:"index;not null;comment:图书ID"`
	BookTitle string `gorm:"size:200;comment:图书标题（冗余字段）"`
	Quantity  int    `gorm:"not null;default:1;comment:购买数量"`
	Price     int64  `gorm:"not null;comment:下单时的单价（分）"`
	CreatedAt time.Time
}

// OrderStatus 订单状态枚举
//
// 教学要点：
// 1. 使用iota实现枚举（Go无原生枚举类型）
// 2. 从1开始而非0：
//   - 0是Go的零值（默认值），容易混淆
//   - 1作为初始状态更明确
type OrderStatus int

const (
	OrderStatusPending   OrderStatus = 1 // 待支付
	OrderStatusPaid      OrderStatus = 2 // 已支付
	OrderStatusShipped   OrderStatus = 3 // 已发货
	OrderStatusCompleted OrderStatus = 4 // 已完成
	OrderStatusCancelled OrderStatus = 5 // 已取消
)

// String 实现Stringer接口（用于日志输出）
func (s OrderStatus) String() string {
	switch s {
	case OrderStatusPending:
		return "待支付"
	case OrderStatusPaid:
		return "已支付"
	case OrderStatusShipped:
		return "已发货"
	case OrderStatusCompleted:
		return "已完成"
	case OrderStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

// IsValid 检查状态值是否合法
func (s OrderStatus) IsValid() bool {
	return s >= OrderStatusPending && s <= OrderStatusCancelled
}

// TableName 指定表名（GORM约定）
func (Order) TableName() string {
	return "orders"
}

// TableName 指定表名
func (OrderItem) TableName() string {
	return "order_items"
}

// CalculateTotal 计算订单总金额
//
// 教学要点：
// 1. 为什么在实体方法中计算，而非应用层？
//   - 领域逻辑内聚：金额计算是订单的核心规则
//   - 避免重复：多个地方需要计算金额时，统一调用
//
// 2. 防御性编程：检查Items是否为空
func (o *Order) CalculateTotal() int64 {
	var total int64
	for _, item := range o.Items {
		total += int64(item.Quantity) * item.Price
	}
	return total
}

// CanTransitionTo 判断是否可以转换到目标状态
//
// 教学要点：
// 这是状态机模式的核心：定义状态之间的合法转换
// 好处：
// 1. 集中管理状态流转规则
// 2. 防止非法状态跳转（如待支付直接变已发货）
// 3. 便于测试和维护
//
// DO vs DON'T:
// ❌ DON'T: 在应用层if-else判断所有状态组合
// ✅ DO: 使用状态机模式，在实体中封装规则
func (o *Order) CanTransitionTo(target OrderStatus) bool {
	// 定义合法的状态转换映射
	//
	// 教学要点：
	// 为什么使用map而非switch？
	// - 扩展性：新增状态只需修改map
	// - 可读性：一目了然看出所有转换规则
	// - 可测试：便于表驱动测试
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending: {
			OrderStatusPaid,      // 支付成功
			OrderStatusCancelled, // 用户取消或超时
		},
		OrderStatusPaid: {
			OrderStatusShipped,   // 商家发货
			OrderStatusCancelled, // 退款（特殊场景）
		},
		OrderStatusShipped: {
			OrderStatusCompleted, // 用户确认收货或自动完成
		},
		// 已完成和已取消是终态，无后续转换
	}

	allowed, exists := transitions[o.Status]
	if !exists {
		return false // 当前状态无合法转换（如已完成）
	}

	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

// UpdateStatus 更新订单状态（带状态机校验）
//
// 教学要点：
// 1. 封装状态变更逻辑在实体方法中（DDD原则）
// 2. 先校验再修改（防御性编程）
// 3. 返回error而非panic（错误是业务异常，不是程序崩溃）
func (o *Order) UpdateStatus(target OrderStatus) error {
	if !o.CanTransitionTo(target) {
		return ErrInvalidStatusTransition
	}

	o.Status = target
	// UpdatedAt会由GORM自动更新
	return nil
}

// IsPending 判断是否待支付
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsPaid 判断是否已支付
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid
}

// IsCancelled 判断是否已取消
func (o *Order) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}

// IsCompleted 判断是否已完成
func (o *Order) IsCompleted() bool {
	return o.Status == OrderStatusCompleted
}

// CanCancel 判断是否可以取消
//
// 教学要点：
// 业务规则：只有待支付和已支付的订单可以取消
// - 待支付：直接取消
// - 已支付：需要退款流程
// - 已发货/已完成：不允许取消（需要走退货流程）
func (o *Order) CanCancel() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusPaid
}
