package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xiebiao/bookstore/services/catalog-service/internal/domain/book"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/infrastructure/config"
)

// NewDB 创建数据库连接
//
// 教学要点：
// 1. GORM连接配置
//   - DSN格式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
//   - charset=utf8mb4：支持emoji和特殊字符
//   - parseTime=True：自动解析time.Time
//   - loc=Local：使用本地时区
//
// 2. 连接池配置
//   - MaxIdleConns：最大空闲连接数（避免频繁创建销毁）
//   - MaxOpenConns：最大打开连接数（防止连接耗尽）
//   - ConnMaxLifetime：连接最大生命周期（避免长连接问题）
//
// 3. 日志配置
//   - 开发环境：打印所有SQL（便于调试）
//   - 生产环境：只打印慢查询和错误
//
// DO（正确做法）：
// - 连接池配置合理（根据并发量调整）
// - 错误处理完整（连接失败、Ping失败）
// - 自动迁移表结构（AutoMigrate）
//
// DON'T（错误做法）：
// - MaxOpenConns设置过大（数据库连接数有限）
// - 生产环境打印所有SQL（性能损耗、日志爆炸）
// - 不处理连接错误（服务启动失败无提示）
func NewDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 步骤1：配置GORM日志
	var logLevel logger.LogLevel
	if cfg.LogMode {
		logLevel = logger.Info // 打印所有SQL
	} else {
		logLevel = logger.Warn // 只打印慢查询和错误
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		// 禁用外键约束（微服务架构不推荐使用外键）
		// 原因：跨服务无法使用外键，保持一致性
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// 步骤2：创建数据库连接
	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 步骤3：获取底层*sql.DB配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// 步骤4：Ping测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 步骤5：自动迁移表结构
	// 教学要点：
	// 1. AutoMigrate会创建表、索引、缺失的列
	// 2. 不会删除已存在的列（安全）
	// 3. 生产环境推荐使用migrate工具（版本控制）
	if err := db.AutoMigrate(&book.Book{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}
