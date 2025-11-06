package book

import "time"

// Book 图书实体（领域模型）
//
// 教学要点：
// 1. 实体 vs 值对象
//   - 实体有唯一标识（ID），有生命周期
//   - 值对象无标识，可替换（如Price可以是值对象）
//
// 2. 与数据库表的关系
//   - ORM会将此结构体映射到catalog_db.books表
//   - GORM tag定义字段约束和索引
//
// 3. Phase 1 vs Phase 2 对比
//   - Phase 1：位于 internal/domain/book/entity.go
//   - Phase 2：位于 services/catalog-service/internal/domain/book/entity.go
//   - 代码相同，但服务隔离（catalog-service独立数据库）
type Book struct {
	// 主键ID（自增）
	ID uint `gorm:"primaryKey" json:"id"`

	// ISBN（国际标准书号，唯一索引）
	// 教学要点：业务唯一标识，用于防止重复出版
	ISBN string `gorm:"uniqueIndex;size:20;not null" json:"isbn"`

	// 书名
	Title string `gorm:"size:200;not null;index:idx_title" json:"title"`

	// 作者
	Author string `gorm:"size:100;not null" json:"author"`

	// 出版社
	Publisher string `gorm:"size:100" json:"publisher"`

	// 价格（单位：分）
	// 教学要点：为什么用int64而非float64？
	// - 浮点数有精度问题（0.1 + 0.2 != 0.3）
	// - 金额计算必须精确，使用整数（分）
	Price int64 `gorm:"not null;index:idx_price" json:"price"`

	// 封面URL
	CoverURL string `gorm:"size:500" json:"cover_url"`

	// 图书描述
	Description string `gorm:"type:text" json:"description"`

	// 发布者用户ID
	// 教学要点：微服务拆分后，不能用外键关联user表
	// 原因：catalog-service和user-service是独立数据库
	PublisherID uint `gorm:"index:idx_publisher" json:"publisher_id"`

	// 创建时间
	CreatedAt time.Time `gorm:"index:idx_created_at" json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// 软删除字段（GORM自动处理）
	// 教学要点：软删除 vs 硬删除
	// - 软删除：标记deleted_at，查询时自动过滤
	// - 硬删除：物理删除，数据不可恢复
	// - 图书系统应使用软删除（保留历史记录、审计需求）
	DeletedAt *time.Time `gorm:"index:idx_deleted_at" json:"-"`
}

// TableName 指定表名
// GORM默认会将结构体名转为复数形式（Book -> books）
// 这里显式指定表名，避免歧义
func (Book) TableName() string {
	return "books"
}

// Validate 验证图书实体
//
// 教学要点：
// 1. 业务规则验证应该在领域层完成
// 2. 不要依赖数据库约束（数据库约束是最后一道防线）
// 3. 验证失败返回业务错误（而非数据库错误）
//
// DO（正确做法）：
// - 在创建实体时调用Validate
// - 返回清晰的业务错误信息
//
// DON'T（错误做法）：
// - 依赖数据库NOT NULL约束（错误信息不友好）
// - 在HTTP层验证业务规则（违反分层原则）
func (b *Book) Validate() error {
	if b.ISBN == "" {
		return ErrISBNRequired
	}

	if b.Title == "" {
		return ErrTitleRequired
	}

	if b.Author == "" {
		return ErrAuthorRequired
	}

	if b.Price <= 0 {
		return ErrInvalidPrice
	}

	// ISBN格式验证（简化版，生产环境需要更严格的验证）
	// 标准ISBN-13：978-7-111-54742-6（13位，带连字符）
	if len(b.ISBN) < 10 || len(b.ISBN) > 20 {
		return ErrInvalidISBN
	}

	return nil
}

// IsPublishedBy 判断图书是否由指定用户发布
// 教学要点：领域方法封装业务逻辑，避免外部直接访问字段
func (b *Book) IsPublishedBy(userID uint) bool {
	return b.PublisherID == userID
}
