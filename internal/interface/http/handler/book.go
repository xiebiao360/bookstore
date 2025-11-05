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
	listBooksUseCase   *appbook.ListBooksUseCase
}

// NewBookHandler 创建图书处理器
func NewBookHandler(publishBookUseCase *appbook.PublishBookUseCase, listBooksUseCase *appbook.ListBooksUseCase) *BookHandler {
	return &BookHandler{
		publishBookUseCase: publishBookUseCase,
		listBooksUseCase:   listBooksUseCase,
	}
}

// PublishBook 发布图书(上架)
// @Summary      发布图书
// @Description  会员发布图书商品上架（需要登录）
// @Tags         图书模块
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.PublishBookRequest true "图书信息"
// @Success      200 {object} response.Response{data=dto.BookResponse} "上架成功"
// @Failure      400 {object} response.Response "参数错误（如ISBN格式错误、价格超出范围）"
// @Failure      401 {object} response.Response "未登录"
// @Failure      409 {object} response.Response "ISBN已存在"
// @Router       /books [post]
//
// 教学说明：@Security注释
// - @Security BearerAuth: 表示此接口需要JWT认证
// - BearerAuth在main.go中定义为securityDefinitions
// - Swagger UI会显示🔒图标，并提供Token输入框
// - 测试时需先调用/users/login获取token，然后点击Authorize按钮输入
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

// ListBooks 查询图书列表
// @Summary      图书列表
// @Description  分页查询图书列表，支持关键词搜索和排序（公开接口，无需登录）
// @Tags         图书模块
// @Accept       json
// @Produce      json
// @Param        page      query    int    false "页码（默认1）" default(1) minimum(1)
// @Param        page_size query    int    false "每页数量（默认20，最大100）" default(20) minimum(1) maximum(100)
// @Param        keyword   query    string false "搜索关键词（匹配标题/作者/出版社）"
// @Param        sort_by   query    string false "排序方式" Enums(price_asc, price_desc, created_at_desc) default(created_at_desc)
// @Success      200 {object} response.Response{data=dto.ListBooksResponse} "查询成功"
// @Failure      400 {object} response.Response "参数错误（如page_size超过100）"
// @Router       /books [get]
//
// 教学说明：Query参数注释
// - @Param的格式: name in type required comment [attributes]
// - in类型: query（URL参数）| path（路径参数）| body（请求体）| header（请求头）
// - attributes（可选）:
//   - default(value): 默认值
//   - minimum(value): 最小值
//   - maximum(value): 最大值
//   - Enums(v1,v2,v3): 枚举值
//
// - Swagger UI会根据这些属性生成友好的输入控件（如下拉框、数字输入框）
func (h *BookHandler) ListBooks(c *gin.Context) {
	// 1. 参数绑定与验证
	var req dto.ListBooksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ErrorWithCode(c, 40900, "参数错误: "+err.Error())
		return
	}

	// 2. 调用应用层用例
	result, err := h.listBooksUseCase.Execute(c.Request.Context(), appbook.ListBooksRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Keyword:  req.Keyword,
		SortBy:   req.SortBy,
	})

	if err != nil {
		response.Error(c, err)
		return
	}

	// 3. 构建HTTP响应
	// 将应用层DTO转换为HTTP层DTO(添加price_yuan字段)
	list := make([]dto.BookListItem, len(result.List))
	for i, item := range result.List {
		list[i] = dto.BookListItem{
			ID:        item.ID,
			ISBN:      item.ISBN,
			Title:     item.Title,
			Author:    item.Author,
			Publisher: item.Publisher,
			Price:     item.Price,
			PriceYuan: dto.FormatPriceYuan(item.Price),
			Stock:     item.Stock,
			CoverURL:  item.CoverURL,
			CreatedAt: item.CreatedAt,
		}
	}

	response.Success(c, &dto.ListBooksResponse{
		List:  list,
		Total: result.Total,
		Page:  result.Page,
		Size:  result.PageSize,
	})
}
