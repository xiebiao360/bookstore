package handler

import (
	"github.com/gin-gonic/gin"

	apporder "github.com/xiebiao/bookstore/internal/application/order"
	"github.com/xiebiao/bookstore/internal/interface/http/dto"
	"github.com/xiebiao/bookstore/internal/interface/http/middleware"
	"github.com/xiebiao/bookstore/pkg/response"
)

// OrderHandler 订单HTTP处理器
type OrderHandler struct {
	createOrderUseCase *apporder.CreateOrderUseCase
}

// NewOrderHandler 创建订单处理器
func NewOrderHandler(createOrderUseCase *apporder.CreateOrderUseCase) *OrderHandler {
	return &OrderHandler{
		createOrderUseCase: createOrderUseCase,
	}
}

// CreateOrder 创建订单
// @Summary      创建订单
// @Description  用户下单购买图书（需要登录），使用悲观锁防止超卖
// @Tags         订单模块
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateOrderRequest true "订单信息"
// @Success      200 {object} response.Response{data=dto.CreateOrderResponse} "下单成功"
// @Failure      400 {object} response.Response "参数错误（如商品数量超过999）"
// @Failure      401 {object} response.Response "未登录"
// @Failure      404 {object} response.Response "图书不存在"
// @Failure      50001 {object} response.Response "库存不足"
// @Router       /orders [post]
//
// 教学说明：防超卖的核心逻辑
// 本接口是整个项目的核心功能之一，演示了如何在高并发场景下防止库存超卖。
//
// 实现方案：悲观锁（SELECT FOR UPDATE）
// 1. 开启数据库事务
// 2. 使用SELECT FOR UPDATE锁定库存行
// 3. 检查库存是否充足
// 4. 创建订单
// 5. 扣减库存
// 6. 提交事务
//
// 为什么不用乐观锁（Version字段）？
// - 高并发场景下，乐观锁会导致大量重试，用户体验差
// - 悲观锁虽然性能略低，但能保证一次成功，更适合秒杀场景
//
// 测试方法：
// 1. 创建库存为10的图书
// 2. 启动10个并发请求，每个购买5本
// 3. 预期结果：只有2个请求成功（10÷5=2），其他8个返回库存不足
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	// 1. 参数绑定与验证
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithCode(c, 40900, "参数错误: "+err.Error())
		return
	}

	// 2. 获取当前登录用户ID
	userID := middleware.MustGetUserID(c)

	// 3. 转换为应用层DTO
	items := make([]apporder.CreateOrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = apporder.CreateOrderItem{
			BookID:   item.BookID,
			Quantity: item.Quantity,
		}
	}

	// 4. 调用应用层用例
	result, err := h.createOrderUseCase.Execute(c.Request.Context(), apporder.CreateOrderRequest{
		UserID: userID,
		Items:  items,
	})

	if err != nil {
		response.Error(c, err)
		return
	}

	// 5. 构建HTTP响应
	response.Success(c, &dto.CreateOrderResponse{
		OrderID:   result.OrderID,
		OrderNo:   result.OrderNo,
		Total:     result.Total,
		TotalYuan: result.TotalYuan,
		Status:    result.Status,
		CreatedAt: result.CreatedAt,
	})
}
