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
// @Description  用户下单购买图书
// @Tags         订单
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateOrderRequest true "订单信息"
// @Success      200 {object} response.Response{data=dto.CreateOrderResponse}
// @Failure      400 {object} response.Response "参数错误"
// @Failure      401 {object} response.Response "未登录"
// @Failure      40001 {object} response.Response "库存不足"
// @Router       /api/v1/orders [post]
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
