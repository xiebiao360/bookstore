package mysql

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xiebiao/bookstore/internal/infrastructure/config"
)

// NewDB 创建数据库连接
// 设计说明：
// 1. 使用GORM v2作为ORM框架
// 2. 配置连接池参数（MaxOpenConns、MaxIdleConns、ConnMaxLifetime）
// 3. 开发环境开启SQL日志，生产环境关闭
// 4. 自动迁移表结构（AutoMigrate）
func NewDB(cfg *config.Config) (*gorm.DB, error) {
	// 1. 构建DSN连接字符串
	dsn := cfg.Database.DSN()

	// 2. 配置GORM日志
	logLevel := logger.Silent
	if cfg.Server.Mode == "debug" {
		logLevel = logger.Info // 开发环境打印SQL
	}

	// 3. 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			// 使用UTC+8时间（配合MySQL的TZ=Asia/Shanghai）
			return time.Now()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 4. 配置连接池
	// 学习要点：合理的连接池配置对性能至关重要
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取SQL DB失败: %w", err)
	}

	// 最大打开连接数（建议：CPU核数 * 2 + 磁盘数量）
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)

	// 最大空闲连接数（建议：MaxOpenConns的1/4到1/2）
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	// 连接最大存活时间（防止数据库主动断开连接）
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// 5. 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("✓ 数据库连接成功")

	// 6. 自动迁移表结构（开发环境）
	// 注意：生产环境应使用专门的迁移工具（如golang-migrate）
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// autoMigrate 自动迁移表结构
// 学习要点：
// 1. AutoMigrate只会创建表、添加字段，不会删除或修改现有字段
// 2. 生产环境应使用版本化的迁移脚本，不要依赖AutoMigrate
func autoMigrate(db *gorm.DB) error {
	// 定义需要迁移的模型
	// 注意：这里需要使用GORM的模型定义（带tag），不是domain层的实体
	return db.AutoMigrate(
		&UserModel{},
		&BookModel{},      // Week 2添加
		&OrderModel{},     // Week 2添加
		&OrderItemModel{}, // Week 2添加
	)
}

// UserModel GORM用户模型
// 设计说明：
// 1. 这是infrastructure层的数据模型，包含GORM tag
// 2. domain/user/entity.go是领域实体，不依赖GORM
// 3. Repository负责两者之间的转换
type UserModel struct {
	ID        uint           `gorm:"primaryKey"`
	Email     string         `gorm:"uniqueIndex;size:100;not null;comment:邮箱"`
	Password  string         `gorm:"size:255;not null;comment:密码（bcrypt加密）"`
	Nickname  string         `gorm:"size:50;not null;comment:昵称"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间（软删除）"`
}

// TableName 指定表名
func (UserModel) TableName() string {
	return "users"
}

// BookModel GORM图书模型
// 设计说明:
// 1. 价格使用int64存储"分"为单位(避免浮点数精度问题)
// 2. ISBN有唯一索引,防止重复
// 3. PublisherID关联用户表,支持查询某用户发布的所有图书
// 4. 添加复合索引优化列表查询性能
type BookModel struct {
	ID          uint           `gorm:"primaryKey"`
	ISBN        string         `gorm:"uniqueIndex;size:20;not null;comment:ISBN号"`
	Title       string         `gorm:"index:idx_search;size:200;not null;comment:书名"` // 搜索索引
	Author      string         `gorm:"index:idx_search;size:100;not null;comment:作者"` // 搜索索引
	Publisher   string         `gorm:"size:100;not null;comment:出版社"`
	Price       int64          `gorm:"index:idx_list;not null;comment:价格(分)"` // 排序索引
	Stock       int            `gorm:"default:0;comment:库存数量"`
	CoverURL    string         `gorm:"size:500;comment:封面图片URL"`
	Description string         `gorm:"type:text;comment:图书描述"`
	PublisherID uint           `gorm:"index;not null;comment:发布者用户ID"`
	CreatedAt   time.Time      `gorm:"index:idx_list;comment:创建时间"` // 排序索引
	UpdatedAt   time.Time      `gorm:"comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"index;comment:删除时间(软删除)"`
}

// TableName 指定表名
func (BookModel) TableName() string {
	return "books"
}

// OrderModel GORM订单模型
// 教学要点:
// 1. 与OrderItemModel是一对多关系
// 2. OrderNo有唯一索引(业务主键)
// 3. Status使用int存储(节省空间,便于索引)
type OrderModel struct {
	ID        uint             `gorm:"primaryKey"`
	OrderNo   string           `gorm:"uniqueIndex;size:32;not null;comment:订单号"`
	UserID    uint             `gorm:"index;not null;comment:买家用户ID"`
	Total     int64            `gorm:"not null;comment:订单总金额(分)"`
	Status    int              `gorm:"index;type:tinyint;default:1;comment:订单状态(1待支付2已支付3已发货4已完成5已取消)"`
	Items     []OrderItemModel `gorm:"foreignKey:OrderID"` // 一对多关联
	CreatedAt time.Time        `gorm:"index;comment:创建时间"`
	UpdatedAt time.Time        `gorm:"comment:更新时间"`
}

// TableName 指定表名
func (OrderModel) TableName() string {
	return "orders"
}

// OrderItemModel GORM订单明细模型
// 教学要点:
// 1. 记录下单时的价格快照(Price字段)
// 2. OrderID外键关联orders表
type OrderItemModel struct {
	ID       uint  `gorm:"primaryKey"`
	OrderID  uint  `gorm:"index;not null;comment:订单ID"`
	BookID   uint  `gorm:"index;not null;comment:图书ID"`
	Quantity int   `gorm:"not null;comment:购买数量"`
	Price    int64 `gorm:"not null;comment:下单时单价(分)"`
}

// TableName 指定表名
func (OrderItemModel) TableName() string {
	return "order_items"
}
