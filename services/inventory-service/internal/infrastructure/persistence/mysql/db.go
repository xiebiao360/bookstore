package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xiebiao/bookstore/services/inventory-service/internal/domain/inventory"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/infrastructure/config"
)

// NewDB 创建数据库连接
func NewDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if cfg.LogMode {
		logLevel = logger.Info
	} else {
		logLevel = logger.Warn
	}

	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&inventory.Inventory{}, &inventory.InventoryLog{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}
