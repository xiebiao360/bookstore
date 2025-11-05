package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xiebiao/bookstore/internal/domain/book"
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// bookRepository 图书仓储实现(MySQL)
// 设计说明:
// 1. 实现domain/book/repository.go定义的接口
// 2. 负责domain实体与GORM模型之间的转换
// 3. 处理数据库特定的错误(如ISBN重复),转换为业务错误
type bookRepository struct {
	db *gorm.DB
}

// NewBookRepository 创建图书仓储
func NewBookRepository(db *gorm.DB) book.Repository {
	return &bookRepository{db: db}
}

// Create 创建图书
func (r *bookRepository) Create(ctx context.Context, b *book.Book) error {
	// 1. 领域实体 → GORM模型
	model := &BookModel{
		ISBN:        b.ISBN,
		Title:       b.Title,
		Author:      b.Author,
		Publisher:   b.Publisher,
		Price:       b.Price,
		Stock:       b.Stock,
		CoverURL:    b.CoverURL,
		Description: b.Description,
		PublisherID: b.PublisherID,
	}

	// 2. 插入数据库
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		// 检查是否为ISBN重复错误
		if isDuplicateError(err) {
			return book.ErrISBNDuplicate
		}
		return apperrors.Wrap(err, "创建图书失败")
	}

	// 3. 回填自增ID
	b.ID = model.ID
	b.CreatedAt = model.CreatedAt
	b.UpdatedAt = model.UpdatedAt

	return nil
}

// FindByID 根据ID查找图书
func (r *bookRepository) FindByID(ctx context.Context, id uint) (*book.Book, error) {
	var model BookModel
	err := r.db.WithContext(ctx).First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, book.ErrBookNotFound
		}
		return nil, apperrors.Wrap(err, "查询图书失败")
	}

	return toBookEntity(&model), nil
}

// FindByISBN 根据ISBN查找图书
func (r *bookRepository) FindByISBN(ctx context.Context, isbn string) (*book.Book, error) {
	var model BookModel
	err := r.db.WithContext(ctx).Where("isbn = ?", isbn).First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, book.ErrBookNotFound
		}
		return nil, apperrors.Wrap(err, "查询图书失败")
	}

	return toBookEntity(&model), nil
}

// Update 更新图书信息
func (r *bookRepository) Update(ctx context.Context, b *book.Book) error {
	model := &BookModel{
		ID:          b.ID,
		ISBN:        b.ISBN,
		Title:       b.Title,
		Author:      b.Author,
		Publisher:   b.Publisher,
		Price:       b.Price,
		Stock:       b.Stock,
		CoverURL:    b.CoverURL,
		Description: b.Description,
		PublisherID: b.PublisherID,
	}

	// 使用Save更新所有字段
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return apperrors.Wrap(err, "更新图书失败")
	}

	b.UpdatedAt = model.UpdatedAt
	return nil
}

// Delete 删除图书(软删除)
func (r *bookRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&BookModel{}, id)

	if result.Error != nil {
		return apperrors.Wrap(result.Error, "删除图书失败")
	}

	if result.RowsAffected == 0 {
		return book.ErrBookNotFound
	}

	return nil
}

// List 分页查询图书列表
// Week 2 Day 10-11会详细实现,此处提供基础版本
func (r *bookRepository) List(ctx context.Context, params book.ListParams) ([]*book.Book, int64, error) {
	var models []BookModel
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&BookModel{})

	// 关键词搜索(搜索标题、作者、出版社)
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("title LIKE ? OR author LIKE ? OR publisher LIKE ?", keyword, keyword, keyword)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询图书总数失败")
	}

	// 排序
	switch params.SortBy {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	case "created_at_desc":
		query = query.Order("created_at DESC")
	default:
		query = query.Order("created_at DESC") // 默认按创建时间降序
	}

	// 分页
	offset := (params.Page - 1) * params.PageSize
	query = query.Limit(params.PageSize).Offset(offset)

	// 查询数据
	if err := query.Find(&models).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询图书列表失败")
	}

	// 转换为领域实体
	books := make([]*book.Book, len(models))
	for i, model := range models {
		books[i] = toBookEntity(&model)
	}

	return books, total, nil
}

// LockByID 悲观锁查询图书(用于订单创建)
// Week 2 Day 12-14会用到
func (r *bookRepository) LockByID(ctx context.Context, id uint) (*book.Book, error) {
	var model BookModel
	// SELECT FOR UPDATE锁定行
	// 教学要点:必须使用getDB(ctx)从context获取事务DB
	db := r.getDB(ctx)
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, book.ErrBookNotFound
		}
		return nil, apperrors.Wrap(err, "锁定图书失败")
	}

	return toBookEntity(&model), nil
}

// UpdateStock 更新库存(原子操作)
func (r *bookRepository) UpdateStock(ctx context.Context, id uint, delta int) error {
	// 使用UPDATE语句原子性更新库存
	// UPDATE books SET stock = stock + delta WHERE id = ? AND stock + delta >= 0
	// 教学要点:必须使用getDB(ctx)参与事务
	db := r.getDB(ctx)
	result := db.Model(&BookModel{}).
		Where("id = ?", id).
		Where("stock + ? >= 0", delta). // 防止库存为负
		Update("stock", gorm.Expr("stock + ?", delta))

	if result.Error != nil {
		return apperrors.Wrap(result.Error, "更新库存失败")
	}

	if result.RowsAffected == 0 {
		// 可能是图书不存在,或者库存不足
		// 再查一次确定原因
		var model BookModel
		db := r.getDB(ctx)
		if err := db.First(&model, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return book.ErrBookNotFound
			}
			return apperrors.Wrap(err, "查询图书失败")
		}
		// 图书存在,说明是库存不足
		return book.ErrInsufficientStock
	}

	return nil
}

// =========================================
// 辅助函数:模型转换
// =========================================

// toBookEntity GORM模型 → 领域实体
func toBookEntity(model *BookModel) *book.Book {
	return &book.Book{
		ID:          model.ID,
		ISBN:        model.ISBN,
		Title:       model.Title,
		Author:      model.Author,
		Publisher:   model.Publisher,
		Price:       model.Price,
		Stock:       model.Stock,
		CoverURL:    model.CoverURL,
		Description: model.Description,
		PublisherID: model.PublisherID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

// getDB 从context获取事务DB,如果没有则使用默认DB
// 教学要点:事务传递机制
func (r *bookRepository) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		return tx
	}
	return r.db
}
