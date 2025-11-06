package mysql

import (
	"fmt"
	"log"
	"time"

	"github.com/xiebiao/bookstore/services/order-service/internal/domain/order"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化数据库连接
//
// 教学要点：
// 1. GORM连接池配置：
//   - MaxIdleConns：最大空闲连接数（保持热连接，减少握手开销）
//   - MaxOpenConns：最大打开连接数（防止数据库连接耗尽）
//   - ConnMaxLifetime：连接最大存活时间（防止长连接被服务器关闭）
//
// 2. 为什么需要连接池？
//   - 复用连接：避免每次请求都建立TCP连接
//   - 并发控制：限制最大连接数，防止数据库过载
//   - 性能优化：减少连接建立和销毁的开销
//
// DO vs DON'T:
// ❌ DON'T: 每次查询都创建新连接（性能极差）
// ✅ DO: 使用连接池，复用连接
func InitDB(cfg *config.DatabaseConfig) *gorm.DB {
	// GORM日志配置
	// 教学要点：
	// - Silent：生产环境（避免日志膨胀）
	// - Info：开发环境（查看SQL语句，便于调试）
	// - Warn：只记录慢查询和错误
	var gormLogger logger.Interface
	if cfg.LogMode {
		gormLogger = logger.Default.LogMode(logger.Info) // 开发模式：打印所有SQL
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent) // 生产模式：静默
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: gormLogger,
		// NowFunc：自定义时间函数（用于测试时mock时间）
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
		// PrepareStmt：预编译SQL（性能优化）
		// 好处：
		// 1. SQL只解析一次，多次执行
		// 2. 防止SQL注入
		PrepareStmt: true,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 获取底层sql.DB配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取sql.DB失败: %v", err)
	}

	// 连接池配置
	// 教学要点：
	// 如何确定连接池大小？
	// - MaxIdleConns：通常为MaxOpenConns的1/4到1/2
	// - MaxOpenConns：根据数据库服务器性能和并发量
	//   - 小应用：10-50
	//   - 中等应用：50-200
	//   - 大应用：200-1000
	// - ConnMaxLifetime：通常为1小时
	//   - 太短：频繁重建连接
	//   - 太长：连接可能被服务器关闭
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("数据库Ping失败: %v", err)
	}

	log.Println("✅ 数据库连接成功")

	// 自动迁移表结构
	// 教学要点：
	// AutoMigrate vs 手动建表：
	// - AutoMigrate：开发便利，但生产环境需谨慎
	//   - 只会新增列，不会删除列（防止数据丢失）
	//   - 不会修改列类型（避免数据截断）
	// - 手动迁移：生产环境推荐（如Flyway、Liquibase）
	//
	// Phase 1使用AutoMigrate简化开发
	// Phase 3会引入专业的迁移工具
	if err := db.AutoMigrate(
		&order.Order{},
		&order.OrderItem{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("✅ 数据库表结构迁移成功")

	return db
}

// Close 关闭数据库连接
//
// 教学要点：
// 为什么需要Close？
// - 释放数据库连接
// - 优雅关闭（等待正在执行的查询完成）
// - 防止连接泄漏
//
// 使用场景：
// - 应用shutdown时调用
// - 集成测试cleanup时调用
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB失败: %w", err)
	}
	return sqlDB.Close()
}
