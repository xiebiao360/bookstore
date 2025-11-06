package mysql

import (
	"fmt"
	"log"
	"time"

	"github.com/xiebiao/bookstore/services/payment-service/internal/domain/payment"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(dsn string) *gorm.DB {
	gormLogger := logger.Default.LogMode(logger.Info)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
		PrepareStmt: true,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("数据库Ping失败: %v", err)
	}

	if err := db.AutoMigrate(&payment.Payment{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("✅ 数据库连接成功")
	return db
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB失败: %w", err)
	}
	return sqlDB.Close()
}
