package order

import (
	"time"
)

// OrderStatus 订单状态
// 教学要点:
// 1. 使用int类型而非string(节省存储空间,便于索引)
// 2. 定义为类型别名,便于添加方法
// 3. 状态值设计:1-5递增,便于理解流转方向
type OrderStatus int

const (
	OrderStatusPending   OrderStatus = 1 // 待支付
	OrderStatusPaid      OrderStatus = 2 // 已支付
	OrderStatusShipped   OrderStatus = 3 // 已发货
	OrderStatusCompleted OrderStatus = 4 // 已完成
	OrderStatusCancelled OrderStatus = 5 // 已取消
)

// String 实现Stringer接口(方便日志输出)
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

// Order 订单实体(聚合根)
// 教学要点:
// 1. Order是聚合根,OrderItem是子实体
// 2. OrderNo使用雪花算法生成(全局唯一,时间有序)
// 3. Total价格冗余存储(避免重复计算,防止改价攻击)
type Order struct {
	ID        uint
	OrderNo   string      // 订单号(业务主键,全局唯一)
	UserID    uint        // 买家用户ID
	Total     int64       // 订单总金额(分),冗余字段
	Status    OrderStatus // 订单状态
	Items     []OrderItem // 订单明细(聚合内的子实体)
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrderItem 订单明细项
// 教学要点:
// 1. 不是独立聚合根,必须通过Order访问
// 2. Price字段记录"下单时的价格"(历史价格快照)
// 3. 不直接关联Book对象,只保存BookID(避免跨聚合引用)
type OrderItem struct {
	ID       uint
	OrderID  uint  // 所属订单ID
	BookID   uint  // 图书ID
	Quantity int   // 购买数量
	Price    int64 // 下单时的单价(分),防止商家改价后历史订单金额变化
}

// NewOrder 创建新订单(工厂方法)
// 教学要点:
// 1. 工厂方法封装创建逻辑,保证实体的有效性
// 2. 订单号由外部传入(可能使用雪花算法、UUID等)
// 3. 初始状态为Pending(待支付)
func NewOrder(orderNo string, userID uint, items []OrderItem, total int64) *Order {
	now := time.Now()
	return &Order{
		OrderNo:   orderNo,
		UserID:    userID,
		Total:     total,
		Status:    OrderStatusPending,
		Items:     items,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CanTransitionTo 检查是否可以转换到目标状态
// 教学要点:状态机设计,防止非法状态跳转
// 例如:不能从"已完成"直接跳到"待支付"
func (o *Order) CanTransitionTo(target OrderStatus) bool {
	// 定义合法的状态转换规则
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:   {OrderStatusPaid, OrderStatusCancelled},    // 待支付→已支付/已取消
		OrderStatusPaid:      {OrderStatusShipped, OrderStatusCancelled}, // 已支付→已发货/已取消(退款)
		OrderStatusShipped:   {OrderStatusCompleted},                     // 已发货→已完成
		OrderStatusCompleted: {},                                         // 已完成→无后续状态(终态)
		OrderStatusCancelled: {},                                         // 已取消→无后续状态(终态)
	}

	allowedTargets, exists := transitions[o.Status]
	if !exists {
		return false
	}

	for _, allowed := range allowedTargets {
		if allowed == target {
			return true
		}
	}
	return false
}

// TransitionTo 状态转换
// 教学要点:
// 1. 先检查是否可以转换(业务规则校验)
// 2. 转换成功更新UpdatedAt(审计追踪)
func (o *Order) TransitionTo(target OrderStatus) error {
	if !o.CanTransitionTo(target) {
		return ErrInvalidStatusTransition
	}
	o.Status = target
	o.UpdatedAt = time.Now()
	return nil
}

// Pay 支付订单(领域行为)
func (o *Order) Pay() error {
	return o.TransitionTo(OrderStatusPaid)
}

// Ship 发货(领域行为)
func (o *Order) Ship() error {
	return o.TransitionTo(OrderStatusShipped)
}

// Complete 完成订单(领域行为)
func (o *Order) Complete() error {
	return o.TransitionTo(OrderStatusCompleted)
}

// Cancel 取消订单(领域行为)
func (o *Order) Cancel() error {
	return o.TransitionTo(OrderStatusCancelled)
}

// CalculateTotal 计算订单总金额
// 教学要点:
// 1. 根据OrderItem列表实时计算
// 2. 用于创建订单时验证前端传递的total是否正确
func (o *Order) CalculateTotal() int64 {
	var total int64
	for _, item := range o.Items {
		total += item.Price * int64(item.Quantity)
	}
	return total
}

// IsOwnedBy 检查订单是否属于指定用户
// 教学要点:权限校验,防止用户访问他人订单
func (o *Order) IsOwnedBy(userID uint) bool {
	return o.UserID == userID
}
