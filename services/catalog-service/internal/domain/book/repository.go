package book

import "context"

// Repository 图书仓储接口（领域层定义）
//
// 教学要点：
// 1. 依赖倒置原则（DIP）
//   - 高层模块（领域层）定义接口
//   - 低层模块（基础设施层）实现接口
//   - 依赖抽象而非具体实现
//
// 2. 为什么要定义接口？
//   - 便于单元测试（Mock仓储）
//   - 可以切换数据库实现（MySQL → PostgreSQL）
//   - 领域层不依赖GORM等外部库
//
// 3. Phase 1 vs Phase 2 对比
//   - Phase 1：仓储实现在同一服务内
//   - Phase 2：每个微服务有独立的仓储实现
type Repository interface {
	// Create 创建图书
	// 教学要点：返回完整的实体（包含自增ID）
	Create(ctx context.Context, book *Book) error

	// FindByID 根据ID查询图书
	// 教学要点：
	// - 返回指针（避免大对象拷贝）
	// - 不存在时返回ErrBookNotFound
	FindByID(ctx context.Context, id uint) (*Book, error)

	// FindByISBN 根据ISBN查询图书
	// 教学要点：ISBN是业务唯一标识，常用于查重
	FindByISBN(ctx context.Context, isbn string) (*Book, error)

	// List 分页查询图书列表
	// 教学要点：
	// - page从1开始（用户友好）
	// - pageSize默认10，最大100（防止大查询）
	// - sortBy支持：created_at（默认）、price
	// - order支持：desc（默认）、asc
	List(ctx context.Context, page, pageSize int, sortBy, order string) ([]*Book, int64, error)

	// Search 搜索图书
	// 教学要点：
	// - 使用LIKE查询（Phase 2简化实现）
	// - Week 7会引入ElasticSearch（全文搜索）
	// - 搜索字段：title、author、publisher
	Search(ctx context.Context, keyword string, page, pageSize int) ([]*Book, int64, error)

	// Update 更新图书
	// 教学要点：
	// - 只更新非零值字段（使用GORM的Updates）
	// - 避免覆盖未传入的字段
	Update(ctx context.Context, book *Book) error

	// Delete 软删除图书
	// 教学要点：
	// - GORM自动处理DeletedAt字段
	// - 查询时自动过滤已删除记录
	Delete(ctx context.Context, id uint) error

	// BatchFindByIDs 批量查询图书（供order-service调用）
	// 教学要点：
	// - 避免N+1查询（一次查询多本书）
	// - 返回map便于按ID查找
	BatchFindByIDs(ctx context.Context, ids []uint) (map[uint]*Book, error)
}
