package book

import (
	"context"
)

// Repository 图书仓储接口(依赖倒置原则)
// 设计说明:
// 1. 由domain层定义接口,infrastructure层实现
// 2. 便于Mock测试,不依赖具体数据库实现
// 3. Phase 2拆分微服务时,可以更换为gRPC调用而不影响domain层
type Repository interface {
	// Create 创建图书
	Create(ctx context.Context, book *Book) error

	// FindByID 根据ID查找图书
	FindByID(ctx context.Context, id uint) (*Book, error)

	// FindByISBN 根据ISBN查找图书
	FindByISBN(ctx context.Context, isbn string) (*Book, error)

	// Update 更新图书信息
	Update(ctx context.Context, book *Book) error

	// Delete 删除图书(软删除)
	Delete(ctx context.Context, id uint) error

	// List 分页查询图书列表(Week 2 Day 10-11会用到)
	// params包含:page, pageSize, keyword, sortBy等
	List(ctx context.Context, params ListParams) ([]*Book, int64, error)

	// LockByID 悲观锁查询图书(用于订单创建时锁定库存)
	// 使用SELECT FOR UPDATE锁定行,防止并发超卖
	// Week 2 Day 12-14会用到
	LockByID(ctx context.Context, id uint) (*Book, error)

	// UpdateStock 更新库存(原子操作)
	// delta为正数表示增加,负数表示减少
	// 内部会检查库存是否充足,不足则返回ErrInsufficientStock
	UpdateStock(ctx context.Context, id uint, delta int) error
}

// ListParams 列表查询参数
// 用于Week 2 Day 10-11的图书列表与搜索功能
type ListParams struct {
	Page     int    // 页码(从1开始)
	PageSize int    // 每页数量
	Keyword  string // 搜索关键词(搜索标题、作者、出版社)
	SortBy   string // 排序字段(price_asc, price_desc, created_at_desc)
}
