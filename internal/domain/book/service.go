package book

import (
	"context"
	"regexp"
)

// Service 图书领域服务接口
// 设计说明:
// 1. 领域服务封装跨实体的业务逻辑和业务规则校验
// 2. 不依赖具体的Repository实现(依赖倒置)
type Service interface {
	// PublishBook 发布图书(上架)
	// 业务规则:
	// - ISBN格式必须合法(10位或13位数字)
	// - 价格必须在1-999999分之间
	// - 库存必须>=0
	// - ISBN不能重复
	PublishBook(ctx context.Context, isbn, title, author, publisher string, price int64, stock int, coverURL, description string, publisherID uint) (*Book, error)

	// GetBookByID 根据ID获取图书详情
	GetBookByID(ctx context.Context, id uint) (*Book, error)

	// GetBookByISBN 根据ISBN获取图书
	GetBookByISBN(ctx context.Context, isbn string) (*Book, error)

	// UpdateBookInfo 更新图书信息
	// 业务规则:只有发布者本人可以修改
	UpdateBookInfo(ctx context.Context, id uint, userID uint, title, author, publisher, description string) error

	// UpdateBookPrice 更新图书价格
	// 业务规则:只有发布者本人可以修改,且价格必须合法
	UpdateBookPrice(ctx context.Context, id uint, userID uint, newPrice int64) error

	// DeleteBook 删除图书
	// 业务规则:只有发布者本人可以删除
	DeleteBook(ctx context.Context, id uint, userID uint) error

	// ListBooks 分页查询图书列表
	// 公开接口,不需要权限校验
	ListBooks(ctx context.Context, params ListParams) ([]*Book, int64, error)
}

// service 领域服务实现
type service struct {
	repo Repository
}

// NewService 创建图书领域服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// PublishBook 发布图书
func (s *service) PublishBook(ctx context.Context, isbn, title, author, publisher string, price int64, stock int, coverURL, description string, publisherID uint) (*Book, error) {
	// 1. ISBN格式校验
	if !isValidISBN(isbn) {
		return nil, ErrInvalidISBN
	}

	// 2. 价格范围校验(1分-9999.99元)
	if price < 1 || price > 999999 {
		return nil, ErrInvalidPrice
	}

	// 3. 库存校验
	if stock < 0 {
		return nil, ErrInvalidStock
	}

	// 4. 检查ISBN是否已存在(Repository会处理重复错误)
	existingBook, err := s.repo.FindByISBN(ctx, isbn)
	if err == nil && existingBook != nil {
		return nil, ErrISBNDuplicate
	}
	// 如果是ErrBookNotFound以外的错误,返回
	if err != nil && err != ErrBookNotFound {
		return nil, err
	}

	// 5. 创建图书实体
	book := NewBook(isbn, title, author, publisher, price, stock, coverURL, description, publisherID)

	// 6. 持久化
	if err := s.repo.Create(ctx, book); err != nil {
		return nil, err
	}

	return book, nil
}

// GetBookByID 根据ID获取图书
func (s *service) GetBookByID(ctx context.Context, id uint) (*Book, error) {
	return s.repo.FindByID(ctx, id)
}

// GetBookByISBN 根据ISBN获取图书
func (s *service) GetBookByISBN(ctx context.Context, isbn string) (*Book, error) {
	if !isValidISBN(isbn) {
		return nil, ErrInvalidISBN
	}
	return s.repo.FindByISBN(ctx, isbn)
}

// UpdateBookInfo 更新图书信息
func (s *service) UpdateBookInfo(ctx context.Context, id uint, userID uint, title, author, publisher, description string) error {
	// 1. 查询图书
	book, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. 权限检查:只有发布者可以修改
	if !book.IsOwnedBy(userID) {
		return ErrUnauthorized
	}

	// 3. 更新信息
	book.UpdateInfo(title, author, publisher, description)

	// 4. 持久化
	return s.repo.Update(ctx, book)
}

// UpdateBookPrice 更新图书价格
func (s *service) UpdateBookPrice(ctx context.Context, id uint, userID uint, newPrice int64) error {
	// 1. 价格范围校验
	if newPrice < 1 || newPrice > 999999 {
		return ErrInvalidPrice
	}

	// 2. 查询图书
	book, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// 3. 权限检查
	if !book.IsOwnedBy(userID) {
		return ErrUnauthorized
	}

	// 4. 更新价格
	if err := book.UpdatePrice(newPrice); err != nil {
		return err
	}

	// 5. 持久化
	return s.repo.Update(ctx, book)
}

// DeleteBook 删除图书
func (s *service) DeleteBook(ctx context.Context, id uint, userID uint) error {
	// 1. 查询图书
	book, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. 权限检查
	if !book.IsOwnedBy(userID) {
		return ErrUnauthorized
	}

	// 3. 执行删除(软删除)
	return s.repo.Delete(ctx, id)
}

// ListBooks 分页查询图书列表
func (s *service) ListBooks(ctx context.Context, params ListParams) ([]*Book, int64, error) {
	return s.repo.List(ctx, params)
}

// =========================================
// 辅助函数:业务规则校验
// =========================================

// isValidISBN 校验ISBN格式
// 支持:
// - ISBN-10: 10位数字,如9787115428028的前10位
// - ISBN-13: 13位数字,如9787115428028
// 简化实现:只检查位数和是否全为数字(生产环境应校验校验位)
func isValidISBN(isbn string) bool {
	// 去除可能的分隔符(如978-7-115-42802-8 → 9787115428028)
	re := regexp.MustCompile(`[^0-9]`)
	cleanISBN := re.ReplaceAllString(isbn, "")

	// 检查位数
	length := len(cleanISBN)
	if length != 10 && length != 13 {
		return false
	}

	// 检查是否全为数字(已通过正则替换保证)
	return true
}
