package book

import (
	"context"

	"github.com/xiebiao/bookstore/internal/domain/book"
)

// PublishBookUseCase 图书上架用例
// 设计说明:
// 1. 应用层负责用例编排,协调领域服务完成业务流程
// 2. 输入输出使用DTO(Data Transfer Object),与HTTP层解耦
// 3. 此用例比较简单,只需调用领域服务即可
type PublishBookUseCase struct {
	bookService book.Service
}

// NewPublishBookUseCase 创建上架用例
func NewPublishBookUseCase(bookService book.Service) *PublishBookUseCase {
	return &PublishBookUseCase{
		bookService: bookService,
	}
}

// PublishBookRequest 上架请求DTO
type PublishBookRequest struct {
	ISBN        string // ISBN号
	Title       string // 书名
	Author      string // 作者
	Publisher   string // 出版社
	Price       int64  // 价格(分)
	Stock       int    // 初始库存
	CoverURL    string // 封面图URL
	Description string // 图书描述
	PublisherID uint   // 发布者用户ID(从认证中间件获取)
}

// PublishBookResponse 上架响应DTO
type PublishBookResponse struct {
	ID          uint   `json:"id"`
	ISBN        string `json:"isbn"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	Price       int64  `json:"price"` // 价格(分)
	Stock       int    `json:"stock"`
	CoverURL    string `json:"cover_url"`
	Description string `json:"description"`
	PublisherID uint   `json:"publisher_id"`
	CreatedAt   string `json:"created_at"`
}

// Execute 执行上架用例
// 学习要点:
// 1. 应用层不直接操作Repository,通过领域服务间接操作
// 2. 业务规则校验由领域服务负责(ISBN格式、价格范围等)
// 3. 应用层只负责流程编排
func (uc *PublishBookUseCase) Execute(ctx context.Context, req PublishBookRequest) (*PublishBookResponse, error) {
	// 调用领域服务发布图书
	// 领域服务会处理:ISBN格式校验、价格范围校验、ISBN重复检查等
	book, err := uc.bookService.PublishBook(
		ctx,
		req.ISBN,
		req.Title,
		req.Author,
		req.Publisher,
		req.Price,
		req.Stock,
		req.CoverURL,
		req.Description,
		req.PublisherID,
	)
	if err != nil {
		return nil, err
	}

	// 构建响应DTO
	return &PublishBookResponse{
		ID:          book.ID,
		ISBN:        book.ISBN,
		Title:       book.Title,
		Author:      book.Author,
		Publisher:   book.Publisher,
		Price:       book.Price,
		Stock:       book.Stock,
		CoverURL:    book.CoverURL,
		Description: book.Description,
		PublisherID: book.PublisherID,
		CreatedAt:   book.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
