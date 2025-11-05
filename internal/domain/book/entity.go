package book

import (
	"time"
)

// Book 图书实体(聚合根)
// DDD设计说明:
// 1. Book是图书聚合的根实体,包含图书的核心属性
// 2. 价格使用int64存储"分"为单位(避免浮点数精度问题)
// 3. ISBN作为业务唯一标识(数据库层保证唯一性)
// 4. PublisherID关联发布图书的用户(会员发布功能)
type Book struct {
	ID          uint
	ISBN        string // ISBN号(国际标准书号)
	Title       string // 书名
	Author      string // 作者
	Publisher   string // 出版社
	Price       int64  // 价格(单位:分,1元=100分)
	Stock       int    // 库存数量
	CoverURL    string // 封面图片URL
	Description string // 图书描述
	PublisherID uint   // 发布者用户ID(关联User表)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewBook 创建新图书(工厂方法)
// 参数说明:
// - isbn: ISBN号(需调用方先验证格式)
// - title: 书名
// - author: 作者
// - publisher: 出版社
// - price: 价格(分),必须>0
// - stock: 初始库存
// - coverURL: 封面图URL
// - description: 图书描述
// - publisherID: 发布者用户ID
func NewBook(isbn, title, author, publisher string, price int64, stock int, coverURL, description string, publisherID uint) *Book {
	now := time.Now()
	return &Book{
		ISBN:        isbn,
		Title:       title,
		Author:      author,
		Publisher:   publisher,
		Price:       price,
		Stock:       stock,
		CoverURL:    coverURL,
		Description: description,
		PublisherID: publisherID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// UpdatePrice 更新价格(领域行为)
// 业务规则:价格必须>0
func (b *Book) UpdatePrice(newPrice int64) error {
	if newPrice <= 0 {
		return ErrInvalidPrice
	}
	b.Price = newPrice
	b.UpdatedAt = time.Now()
	return nil
}

// UpdateStock 更新库存(领域行为)
// 业务规则:库存不能为负数
func (b *Book) UpdateStock(newStock int) error {
	if newStock < 0 {
		return ErrInvalidStock
	}
	b.Stock = newStock
	b.UpdatedAt = time.Now()
	return nil
}

// DecrStock 扣减库存(用于订单创建)
// 业务规则:扣减后库存不能为负数
func (b *Book) DecrStock(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if b.Stock < quantity {
		return ErrInsufficientStock
	}
	b.Stock -= quantity
	b.UpdatedAt = time.Now()
	return nil
}

// IncrStock 增加库存(用于订单取消、补货)
func (b *Book) IncrStock(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	b.Stock += quantity
	b.UpdatedAt = time.Now()
	return nil
}

// UpdateInfo 更新图书基本信息
func (b *Book) UpdateInfo(title, author, publisher, description string) {
	if title != "" {
		b.Title = title
	}
	if author != "" {
		b.Author = author
	}
	if publisher != "" {
		b.Publisher = publisher
	}
	if description != "" {
		b.Description = description
	}
	b.UpdatedAt = time.Now()
}

// IsOwnedBy 检查图书是否由指定用户发布
func (b *Book) IsOwnedBy(userID uint) bool {
	return b.PublisherID == userID
}
