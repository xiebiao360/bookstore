package order

import "context"

// Repository 订单仓储接口
//
// 教学要点：
// 1. 为什么定义接口而非直接使用GORM？
//   - 依赖倒置原则（DIP）：高层（领域层）不依赖低层（基础设施层）
//   - 可测试性：可以Mock Repository进行单元测试
//   - 灵活性：可以切换存储（MySQL → PostgreSQL → MongoDB）
//
// 2. Repository vs DAO：
//   - Repository：面向领域对象（Order），提供集合语义（Add/FindByID）
//   - DAO：面向数据库表（orders），提供CRUD操作（Insert/Select）
//   - DDD推荐使用Repository
//
// 3. 接口设计原则：
//   - 最小化：只暴露必要的方法
//   - 语义化：方法名反映业务意图（Create vs Insert）
//   - 返回领域对象：Order而非map或DTO
//
// DO vs DON'T:
// ❌ DON'T: 定义通用的CRUD接口（如GenericRepository<T>）
// ✅ DO: 针对每个聚合根定义专用接口
type Repository interface {
	// Create 创建订单（含订单明细）
	//
	// 教学要点：
	// 1. 为什么是Create而非Insert？
	//    - Create是业务语言（Ubiquitous Language）
	//    - Insert是技术术语
	// 2. 为什么传指针*Order？
	//    - 需要回填ID（GORM会自动赋值）
	//    - 避免大对象复制
	// 3. 事务边界在哪里？
	//    - Repository实现中会开启事务
	//    - 保证Order和OrderItem同时插入
	Create(ctx context.Context, order *Order) error

	// FindByID 根据ID查询订单（含订单明细）
	//
	// 教学要点：
	// 1. 为什么返回(*Order, error)而非(Order, error)？
	//    - 可以返回nil表示未找到
	//    - 避免零值Order的歧义（ID=0是否表示不存在？）
	// 2. 是否预加载Items？
	//    - 订单和明细是强关联，通常一起查询
	//    - GORM: db.Preload("Items").First(&order)
	FindByID(ctx context.Context, id uint) (*Order, error)

	// FindByOrderNo 根据订单号查询
	//
	// 教学要点：
	// 订单号是对外展示的业务主键，用户常用此查询
	// 需要在order_no字段上建唯一索引（UNIQUE INDEX）
	FindByOrderNo(ctx context.Context, orderNo string) (*Order, error)

	// FindByUserID 查询用户的订单列表
	//
	// 教学要点：
	// 1. 为什么需要分页参数？
	//    - 用户可能有成百上千个订单
	//    - 一次性加载会OOM
	// 2. status=0表示查询所有状态
	//    - 筛选特定状态：status=1（待支付）
	//    - 查询全部：status=0
	FindByUserID(ctx context.Context, userID uint, page, pageSize int, status OrderStatus) ([]*Order, int64, error)

	// Update 更新订单
	//
	// 教学要点：
	// 1. 更新哪些字段？
	//    - 通常只更新Status（状态变更）
	//    - GORM会自动更新UpdatedAt
	// 2. 是否更新OrderItem？
	//    - 订单创建后明细通常不可修改
	//    - 如需修改，应该取消订单重新下单
	Update(ctx context.Context, order *Order) error

	// UpdateStatus 仅更新状态（性能优化）
	//
	// 教学要点：
	// Update会更新所有字段，UpdateStatus只更新status字段
	// SQL: UPDATE orders SET status=?, updated_at=? WHERE id=?
	// 好处：
	// 1. 减少数据传输
	// 2. 避免并发更新冲突
	UpdateStatus(ctx context.Context, id uint, status OrderStatus) error

	// Delete 删除订单（软删除）
	//
	// 教学要点：
	// 1. 软删除 vs 硬删除：
	//    - 软删除：设置deleted_at字段，查询时过滤
	//    - 硬删除：DELETE FROM orders WHERE id=?
	// 2. 电商系统通常使用软删除：
	//    - 保留历史数据用于分析
	//    - 支持数据恢复
	//    - 满足审计要求
	// 3. GORM软删除：
	//    - 添加gorm.DeletedAt字段
	//    - Delete自动设置deleted_at
	//    - 查询自动过滤deleted_at IS NULL
	Delete(ctx context.Context, id uint) error
}

// ItemRepository 订单明细仓储接口
//
// 教学要点：
// 为什么单独定义ItemRepository？
// - OrderItem通常通过Order操作（聚合模式）
// - 但某些场景需要单独查询明细：
//   - 统计某本书的销量
//   - 查询用户购买过的所有图书
//
// - 这是聚合设计的权衡：便利性 vs 严格性
//
// 注意：
// - 不提供Create/Update/Delete方法
// - 只提供查询方法（明细的修改通过Order完成）
type ItemRepository interface {
	// FindByOrderID 查询订单的所有明细
	FindByOrderID(ctx context.Context, orderID uint) ([]*OrderItem, error)

	// FindByBookID 查询某本书的所有订单明细（用于统计销量）
	FindByBookID(ctx context.Context, bookID uint, limit int) ([]*OrderItem, error)
}
