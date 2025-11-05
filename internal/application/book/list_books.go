package book

import (
	"context"

	"github.com/xiebiao/bookstore/internal/domain/book"
)

// ListBooksUseCase 图书列表查询用例
// 设计说明:
// 1. 支持分页、搜索、排序
// 2. 列表查询不返回description字段(减少数据传输量)
// 3. 为Phase 2迁移到ElasticSearch做准备
type ListBooksUseCase struct {
	bookService book.Service
}

// NewListBooksUseCase 创建列表查询用例
func NewListBooksUseCase(bookService book.Service) *ListBooksUseCase {
	return &ListBooksUseCase{
		bookService: bookService,
	}
}

// ListBooksRequest 列表查询请求DTO
type ListBooksRequest struct {
	Page     int    // 页码(从1开始)
	PageSize int    // 每页数量
	Keyword  string // 搜索关键词(搜索标题、作者、出版社)
	SortBy   string // 排序方式(price_asc, price_desc, created_at_desc)
}

// BookListItem 列表项DTO(不含description)
type BookListItem struct {
	ID        uint   `json:"id"`
	ISBN      string `json:"isbn"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Price     int64  `json:"price"` // 价格(分)
	Stock     int    `json:"stock"`
	CoverURL  string `json:"cover_url"`
	CreatedAt string `json:"created_at"`
}

// ListBooksResponse 列表查询响应DTO
type ListBooksResponse struct {
	List       []BookListItem `json:"list"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// Execute 执行列表查询用例
// 学习要点:
// 1. 参数默认值处理(page默认1, pageSize默认20)
// 2. 参数范围限制(pageSize最大100)
// 3. 调用Repository的List方法执行查询
func (uc *ListBooksUseCase) Execute(ctx context.Context, req ListBooksRequest) (*ListBooksResponse, error) {
	// 1. 参数默认值与范围限制
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20 // 默认每页20条
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 最大每页100条
	}

	// 2. 构建Repository查询参数
	params := book.ListParams{
		Page:     req.Page,
		PageSize: req.PageSize,
		Keyword:  req.Keyword,
		SortBy:   req.SortBy,
	}

	// 3. 调用Repository查询
	books, total, err := uc.bookService.ListBooks(ctx, params)
	if err != nil {
		return nil, err
	}

	// 4. 转换为DTO
	list := make([]BookListItem, len(books))
	for i, b := range books {
		list[i] = BookListItem{
			ID:        b.ID,
			ISBN:      b.ISBN,
			Title:     b.Title,
			Author:    b.Author,
			Publisher: b.Publisher,
			Price:     b.Price,
			Stock:     b.Stock,
			CoverURL:  b.CoverURL,
			CreatedAt: b.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	// 5. 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	return &ListBooksResponse{
		List:       list,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}
