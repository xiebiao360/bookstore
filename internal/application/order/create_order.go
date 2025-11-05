package order

import (
	"context"
	"fmt"

	"github.com/xiebiao/bookstore/internal/domain/book"
	"github.com/xiebiao/bookstore/internal/domain/order"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/mysql"
)

// CreateOrderUseCase 创建订单用例
// 教学要点:这是整个项目最核心的用例之一
// 涉及:事务处理、并发控制、业务规则校验
type CreateOrderUseCase struct {
	orderRepo order.Repository
	bookRepo  book.Repository
	txManager *mysql.TxManager
}

// NewCreateOrderUseCase 创建下单用例
func NewCreateOrderUseCase(
	orderRepo order.Repository,
	bookRepo book.Repository,
	txManager *mysql.TxManager,
) *CreateOrderUseCase {
	return &CreateOrderUseCase{
		orderRepo: orderRepo,
		bookRepo:  bookRepo,
		txManager: txManager,
	}
}

// CreateOrderRequest 下单请求DTO
type CreateOrderRequest struct {
	UserID uint              // 买家用户ID(从JWT中提取)
	Items  []CreateOrderItem // 订单明细
}

// CreateOrderItem 订单明细项
type CreateOrderItem struct {
	BookID   uint // 图书ID
	Quantity int  // 购买数量
}

// CreateOrderResponse 下单响应DTO
type CreateOrderResponse struct {
	OrderID   uint   `json:"order_id"`
	OrderNo   string `json:"order_no"`
	Total     int64  `json:"total"`
	TotalYuan string `json:"total_yuan"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// Execute 执行下单用例
// 教学重点:防止超卖的完整流程
//
// 核心问题:库存超卖
// 场景:商品库存10个,100人同时下单
// 错误实现:
//  1. 查询库存 → 10个
//  2. 判断够不够 → 够
//  3. 扣减库存 → stock = stock - 1
//     结果:100个请求都通过了步骤2,最后卖出100个(超卖90个!)
//
// 正确实现:悲观锁
//  1. SELECT FOR UPDATE 锁定库存行
//  2. 判断库存是否充足
//  3. 扣减库存
//  4. 创建订单
//  5. COMMIT释放锁
func (uc *CreateOrderUseCase) Execute(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	// 1. 参数校验
	if len(req.Items) == 0 {
		return nil, order.ErrInvalidOrderItems
	}

	// 使用事务执行整个下单流程
	// 教学要点:事务保证原子性,要么全成功,要么全失败
	var result *order.Order
	err := uc.txManager.Transaction(ctx, func(txCtx context.Context) error {
		// ========================================
		// 步骤1:锁定库存(悲观锁,防止并发超卖)
		// ========================================
		// 教学要点:SELECT FOR UPDATE会锁定查询的行
		// 其他事务必须等待当前事务COMMIT或ROLLBACK后才能访问
		bookMap := make(map[uint]*book.Book)
		for _, item := range req.Items {
			if item.Quantity <= 0 {
				return order.ErrInvalidQuantity
			}

			// LockByID执行:SELECT * FROM books WHERE id = ? FOR UPDATE
			// 这会在books表的该行上加排他锁(X锁)
			b, err := uc.bookRepo.LockByID(txCtx, item.BookID)
			if err != nil {
				return err
			}

			// 检查库存是否充足
			// 教学要点:必须在锁定后检查,否则可能并发扣减导致超卖
			if b.Stock < item.Quantity {
				return fmt.Errorf("图书《%s》库存不足,当前库存:%d,需要:%d",
					b.Title, b.Stock, item.Quantity)
			}

			bookMap[item.BookID] = b
		}

		// ========================================
		// 步骤2:计算订单金额
		// ========================================
		// 教学要点:使用"锁定时的价格"而非前端传递的价格
		// 防止改价攻击:用户在前端修改价格提交
		var total int64
		orderItems := make([]order.OrderItem, len(req.Items))
		for i, item := range req.Items {
			b := bookMap[item.BookID]
			orderItems[i] = order.OrderItem{
				BookID:   item.BookID,
				Quantity: item.Quantity,
				Price:    b.Price, // 使用数据库中的当前价格
			}
			total += b.Price * int64(item.Quantity)
		}

		// ========================================
		// 步骤3:创建订单
		// ========================================
		orderNo := order.GenerateOrderNo()
		newOrder := order.NewOrder(orderNo, req.UserID, orderItems, total)

		// 持久化订单(包含订单明细)
		if err := uc.orderRepo.Create(txCtx, newOrder); err != nil {
			return err
		}

		// ========================================
		// 步骤4:扣减库存
		// ========================================
		// 教学要点:为什么不直接UPDATE books SET stock = stock - ?
		// 答:虽然也能防止负数(WHERE stock >= ?),但无法返回友好错误信息
		// 使用Repository的UpdateStock方法,内部会检查并返回ErrInsufficientStock
		for _, item := range req.Items {
			err := uc.bookRepo.UpdateStock(txCtx, item.BookID, -item.Quantity)
			if err != nil {
				// 如果扣减失败,整个事务会回滚
				// 订单不会创建,库存不会减少
				return err
			}
		}

		// ========================================
		// 步骤5:返回订单(事务自动COMMIT)
		// ========================================
		result = newOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 构建响应DTO
	return &CreateOrderResponse{
		OrderID:   result.ID,
		OrderNo:   result.OrderNo,
		Total:     result.Total,
		TotalYuan: formatPrice(result.Total),
		Status:    result.Status.String(),
		CreatedAt: result.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// formatPrice 格式化价格(分→元)
func formatPrice(priceFen int64) string {
	yuan := float64(priceFen) / 100.0
	return fmt.Sprintf("%.2f", yuan)
}
