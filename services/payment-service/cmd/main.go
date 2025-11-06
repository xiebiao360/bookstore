package main

import (
	"fmt"
	"log"
	"net"

	"github.com/spf13/viper"
	paymentv1 "github.com/xiebiao/bookstore/proto/paymentv1"
	"github.com/xiebiao/bookstore/services/payment-service/internal/domain/payment"
	"github.com/xiebiao/bookstore/services/payment-service/internal/grpc/handler"
	"github.com/xiebiao/bookstore/services/payment-service/internal/infrastructure/persistence/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	v := viper.New()
	v.SetConfigFile("./config/config.yaml")
	v.ReadInConfig()

	dsn := v.GetString("database.dsn")
	port := v.GetInt("server.port")

	gormLogger := logger.Default.LogMode(logger.Info)
	db, err := gorm.Open(mysqlDriver.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	db.AutoMigrate(&payment.Payment{})
	log.Println("✅ payment_db迁移成功")

	repo := mysql.NewPaymentRepository(db)
	grpcServer := grpc.NewServer()
	paymentService := handler.NewPaymentServiceServer(repo)
	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentService)
	reflection.Register(grpcServer)

	lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	log.Printf("🚀 payment-service启动，端口:%d", port)
	log.Printf("💳 Mock支付：70%%成功率")
	grpcServer.Serve(lis)
}
