package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/xiebiao/bookstore/services/catalog-service/internal/domain/book"
	"gorm.io/gorm"
)

// bookRepository MySQL仓储实现
//
// 教学要点：
// 1. 实现领域层定义的Repository接口
// 2. 封装GORM细节，领域层不感知
// 3. 错误转换：GORM错误 → 领域错误
//
// DO（正确做法）：
// - 使用Context传递超时控制
// - 错误统一转换为领域错误
// - SQL注入防护（GORM自动参数化）
//
// DON'T（错误做法）：
// - 直接返回GORM错误（泄漏基础设施细节）
// - 忽略Context（无法取消长时间查询）
type bookRepository struct {
	db *gorm.DB
}

// NewBookRepository 创建图书仓储实例
func NewBookRepository(db *gorm.DB) book.Repository {
	return &bookRepository{db: db}
}

// Create 创建图书
func (r *bookRepository) Create(ctx context.Context, b *book.Book) error {
	// 教学要点：
	// 1. WithContext传递超时控制
	// 2. GORM会自动填充ID、CreatedAt、UpdatedAt
	// 3. ISBN唯一索引冲突会返回错误

	if err := r.db.WithContext(ctx).Create(b).Error; err != nil {
		// 检查是否是唯一索引冲突（ISBN重复）
		if isDuplicateError(err) {
			return book.ErrISBNDup
		}
		return fmt.Errorf("创建图书失败: %w", err)
	}

	return nil
}

// FindByID 根据ID查询图书
func (r *bookRepository) FindByID(ctx context.Context, id uint) (*book.Book, error) {
	var b book.Book

	// 教学要点：
	// 1. First查询单条记录
	// 2. GORM自动过滤DeletedAt不为NULL的记录（软删除）
	// 3. 记录不存在时返回gorm.ErrRecordNotFound
	if err := r.db.WithContext(ctx).First(&b, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, book.ErrBookNotFound
		}
		return nil, fmt.Errorf("查询图书失败: %w", err)
	}

	return &b, nil
}

// FindByISBN 根据ISBN查询图书
func (r *bookRepository) FindByISBN(ctx context.Context, isbn string) (*book.Book, error) {
	var b book.Book

	// 教学要点：Where条件查询，参数化防止SQL注入
	if err := r.db.WithContext(ctx).Where("isbn = ?", isbn).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, book.ErrBookNotFound
		}
		return nil, fmt.Errorf("查询图书失败: %w", err)
	}

	return &b, nil
}

// List 分页查询图书列表
func (r *bookRepository) List(ctx context.Context, page, pageSize int, sortBy, order string) ([]*book.Book, int64, error) {
	// 教学要点：
	// 1. 参数验证和默认值处理
	// 2. 分页计算：offset = (page - 1) * pageSize
	// 3. 同时返回总数（用于前端分页组件）
	// 4. 排序字段白名单（防止SQL注入）

	// 步骤1：参数验证和默认值
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 // 限制最大每页数量，防止大查询
	}

	// 排序字段白名单
	allowedSortFields := map[string]bool{
		"created_at": true,
		"price":      true,
		"id":         true,
	}
	if !allowedSortFields[sortBy] {
		sortBy = "created_at" // 默认按创建时间排序
	}

	// 排序方向白名单
	if order != "asc" && order != "desc" {
		order = "desc" // 默认降序
	}

	// 步骤2：查询总数
	var total int64
	if err := r.db.WithContext(ctx).Model(&book.Book{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询图书总数失败: %w", err)
	}

	// 步骤3：分页查询
	var books []*book.Book
	offset := (page - 1) * pageSize

	// 教学要点：链式调用构建查询
	// - Order: 排序
	// - Offset: 跳过前N条
	// - Limit: 返回M条
	if err := r.db.WithContext(ctx).
		Order(fmt.Sprintf("%s %s", sortBy, order)).
		Offset(offset).
		Limit(pageSize).
		Find(&books).Error; err != nil {
		return nil, 0, fmt.Errorf("查询图书列表失败: %w", err)
	}

	return books, total, nil
}

// Search 搜索图书
func (r *bookRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]*book.Book, int64, error) {
	// 教学要点：
	// 1. 多字段模糊查询（LIKE）
	// 2. OR条件组合（title OR author OR publisher）
	// 3. Phase 2简化实现，Week 7会引入ElasticSearch
	//
	// LIKE查询的局限性：
	// - 无法分词（"Go语言"搜不到"Go 语言"）
	// - 无法相关性排序
	// - 大数据量性能差（全表扫描）
	// - 无法处理同义词、拼音搜索
	//
	// ElasticSearch优势：
	// - 分词搜索
	// - 相关性评分
	// - 高性能（倒排索引）
	// - 支持拼音、同义词

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 如果关键词为空，返回空结果
	if keyword == "" {
		return []*book.Book{}, 0, nil
	}

	// LIKE查询需要加通配符
	pattern := "%" + keyword + "%"

	// 查询总数
	var total int64
	query := r.db.WithContext(ctx).Model(&book.Book{}).
		Where("title LIKE ? OR author LIKE ? OR publisher LIKE ?", pattern, pattern, pattern)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询搜索结果总数失败: %w", err)
	}

	// 分页查询
	var books []*book.Book
	offset := (page - 1) * pageSize

	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&books).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索图书失败: %w", err)
	}

	return books, total, nil
}

// Update 更新图书
func (r *bookRepository) Update(ctx context.Context, b *book.Book) error {
	// 教学要点：
	// 1. Updates只更新非零值字段
	// 2. Save会更新所有字段（包括零值）
	// 3. 使用Updates避免误覆盖

	// DO（正确做法）：
	result := r.db.WithContext(ctx).Model(&book.Book{}).
		Where("id = ?", b.ID).
		Updates(b)

	// DON'T（错误做法）：
	// r.db.Save(b) // 会覆盖所有字段，包括零值

	if err := result.Error; err != nil {
		return fmt.Errorf("更新图书失败: %w", err)
	}

	// 检查是否更新了记录
	if result.RowsAffected == 0 {
		return book.ErrBookNotFound
	}

	return nil
}

// Delete 软删除图书
func (r *bookRepository) Delete(ctx context.Context, id uint) error {
	// 教学要点：
	// 1. GORM的Delete会自动设置DeletedAt字段（软删除）
	// 2. 后续查询会自动过滤已删除记录
	// 3. 永久删除需要使用Unscoped().Delete()

	result := r.db.WithContext(ctx).Delete(&book.Book{}, id)

	if err := result.Error; err != nil {
		return fmt.Errorf("删除图书失败: %w", err)
	}

	if result.RowsAffected == 0 {
		return book.ErrBookNotFound
	}

	return nil
}

// BatchFindByIDs 批量查询图书
func (r *bookRepository) BatchFindByIDs(ctx context.Context, ids []uint) (map[uint]*book.Book, error) {
	// 教学要点：
	// 1. 避免N+1查询问题
	//    ❌ 错误：循环调用FindByID（N次查询）
	//    ✅ 正确：一次性查询所有ID（1次查询）
	//
	// 2. 返回map便于调用方按ID查找
	//    - 时间复杂度：O(1)
	//    - 数组查找：O(N)

	if len(ids) == 0 {
		return make(map[uint]*book.Book), nil
	}

	var books []*book.Book

	// 使用IN查询批量获取
	// SQL: SELECT * FROM books WHERE id IN (1, 2, 3) AND deleted_at IS NULL
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&books).Error; err != nil {
		return nil, fmt.Errorf("批量查询图书失败: %w", err)
	}

	// 转换为map
	bookMap := make(map[uint]*book.Book, len(books))
	for _, b := range books {
		bookMap[b.ID] = b
	}

	return bookMap, nil
}

// isDuplicateError 判断是否是唯一索引冲突错误
//
// 教学要点：
// 1. MySQL错误码1062：Duplicate entry
// 2. 不同数据库错误码不同，需要适配
func isDuplicateError(err error) bool {
	// 简化判断：检查错误消息是否包含"Duplicate"
	// 生产环境应该检查具体错误码
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return contains(errMsg, "Duplicate") || contains(errMsg, "duplicate")
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
