package dto

import "fmt"

// PublishBookRequest HTTP上架请求
// validator tag说明:
// - required: 必填字段
// - min/max: 数值范围校验
// - isbn: 自定义ISBN格式校验(需在pkg/validator中注册)
type PublishBookRequest struct {
	ISBN        string `json:"isbn" binding:"required" example:"9787115428028"`
	Title       string `json:"title" binding:"required,max=200" example:"Go语言实战"`
	Author      string `json:"author" binding:"required,max=100" example:"威廉·肯尼迪"`
	Publisher   string `json:"publisher" binding:"required,max=100" example:"人民邮电出版社"`
	Price       int64  `json:"price" binding:"required,min=1,max=999999" example:"5900"` // 价格(分),59.00元
	Stock       int    `json:"stock" binding:"min=0" example:"100"`
	CoverURL    string `json:"cover_url" binding:"omitempty,url,max=500" example:"https://example.com/cover.jpg"`
	Description string `json:"description" binding:"max=5000" example:"这是一本关于Go语言的实战书籍"`
}

// BookResponse HTTP图书响应
// 用于单个图书详情返回
type BookResponse struct {
	ID          uint   `json:"id" example:"1"`
	ISBN        string `json:"isbn" example:"9787115428028"`
	Title       string `json:"title" example:"Go语言实战"`
	Author      string `json:"author" example:"威廉·肯尼迪"`
	Publisher   string `json:"publisher" example:"人民邮电出版社"`
	Price       int64  `json:"price" example:"5900"`       // 价格(分)
	PriceYuan   string `json:"price_yuan" example:"59.00"` // 价格(元),方便前端显示
	Stock       int    `json:"stock" example:"100"`
	CoverURL    string `json:"cover_url" example:"https://example.com/cover.jpg"`
	Description string `json:"description" example:"这是一本关于Go语言的实战书籍"`
	PublisherID uint   `json:"publisher_id" example:"1"`
	CreatedAt   string `json:"created_at" example:"2024-01-15 10:30:00"`
	UpdatedAt   string `json:"updated_at" example:"2024-01-15 10:30:00"`
}

// BookListItem HTTP图书列表项
// 列表查询时不返回Description字段(减少数据传输量)
type BookListItem struct {
	ID        uint   `json:"id" example:"1"`
	ISBN      string `json:"isbn" example:"9787115428028"`
	Title     string `json:"title" example:"Go语言实战"`
	Author    string `json:"author" example:"威廉·肯尼迪"`
	Publisher string `json:"publisher" example:"人民邮电出版社"`
	Price     int64  `json:"price" example:"5900"`
	PriceYuan string `json:"price_yuan" example:"59.00"`
	Stock     int    `json:"stock" example:"100"`
	CoverURL  string `json:"cover_url" example:"https://example.com/cover.jpg"`
	CreatedAt string `json:"created_at" example:"2024-01-15 10:30:00"`
}

// ListBooksRequest HTTP图书列表请求
// Week 2 Day 10-11会用到
type ListBooksRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Keyword  string `form:"keyword" binding:"omitempty,max=100" example:"Go"`
	SortBy   string `form:"sort_by" binding:"omitempty,oneof=price_asc price_desc created_at_desc" example:"created_at_desc"`
}

// ListBooksResponse HTTP图书列表响应
type ListBooksResponse struct {
	List  []BookListItem `json:"list"`
	Total int64          `json:"total" example:"100"`
	Page  int            `json:"page" example:"1"`
	Size  int            `json:"size" example:"20"`
}

// FormatPriceYuan 格式化价格(分→元)
// 工具函数:将价格从分转换为元的字符串表示
// 例如:5900分 → "59.00"
func FormatPriceYuan(priceFen int64) string {
	yuan := float64(priceFen) / 100.0
	return fmt.Sprintf("%.2f", yuan)
}

// =========================================
// 订单相关DTO
// =========================================

// CreateOrderRequest HTTP下单请求
type CreateOrderRequest struct {
	Items []CreateOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// CreateOrderItemRequest 订单明细项
type CreateOrderItemRequest struct {
	BookID   uint `json:"book_id" binding:"required"`
	Quantity int  `json:"quantity" binding:"required,min=1,max=999"`
}

// CreateOrderResponse HTTP下单响应
type CreateOrderResponse struct {
	OrderID   uint   `json:"order_id" example:"1"`
	OrderNo   string `json:"order_no" example:"ORD1699248000123456"`
	Total     int64  `json:"total" example:"11800"`
	TotalYuan string `json:"total_yuan" example:"118.00"`
	Status    string `json:"status" example:"待支付"`
	CreatedAt string `json:"created_at" example:"2024-11-06 10:30:00"`
}
