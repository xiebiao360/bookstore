package handler

import (
	"github.com/gin-gonic/gin"

	appbook "github.com/xiebiao/bookstore/internal/application/book"
	"github.com/xiebiao/bookstore/internal/interface/http/dto"
	"github.com/xiebiao/bookstore/internal/interface/http/middleware"
	"github.com/xiebiao/bookstore/pkg/response"
)

// BookHandler 图书HTTP处理器
type BookHandler struct {
	publishBookUseCase *appbook.PublishBookUseCase
}

// NewBookHandler 创建图书处理器
func NewBookHandler(publishBookUseCase *appbook.PublishBookUseCase) *BookHandler {
	return &BookHandler{
		publishBookUseCase: publishBookUseCase,
	}
}

// PublishBook 发布图书(上架)
// @Summary      发布图书
// @Description  会员发布图书商品上架
// @Tags         图书
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.PublishBookRequest true "图书信息"
// @Success      200 {object} response.Response{data=dto.BookResponse}
// @Failure      400 {object} response.Response "参数错误"
// @Failure      401 {object} response.Response "未登录"
// @Failure      409 {object} response.Response "ISBN已存在"
// @Router       /api/v1/books [post]
func (h *BookHandler) PublishBook(c *gin.Context) {
	// 1. 参数绑定与验证
	var req dto.PublishBookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithCode(c, 40900, "参数错误: "+err.Error())
		return
	}

	// 2. 获取当前登录用户ID(从认证中间件注入的Context中获取)
	userID := middleware.MustGetUserID(c)

	// 3. 调用应用层用例
	result, err := h.publishBookUseCase.Execute(c.Request.Context(), appbook.PublishBookRequest{
		ISBN:        req.ISBN,
		Title:       req.Title,
		Author:      req.Author,
		Publisher:   req.Publisher,
		Price:       req.Price,
		Stock:       req.Stock,
		CoverURL:    req.CoverURL,
		Description: req.Description,
		PublisherID: userID, // 使用当前登录用户ID
	})

	if err != nil {
		response.Error(c, err)
		return
	}

	// 4. 构建HTTP响应
	response.Success(c, &dto.BookResponse{
		ID:          result.ID,
		ISBN:        result.ISBN,
		Title:       result.Title,
		Author:      result.Author,
		Publisher:   result.Publisher,
		Price:       result.Price,
		PriceYuan:   dto.FormatPriceYuan(result.Price),
		Stock:       result.Stock,
		CoverURL:    result.CoverURL,
		Description: result.Description,
		PublisherID: result.PublisherID,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.CreatedAt, // 新创建时UpdatedAt等于CreatedAt
	})
}
