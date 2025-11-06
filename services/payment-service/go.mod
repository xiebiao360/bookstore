module github.com/xiebiao/bookstore/services/payment-service

go 1.21

require (
	github.com/spf13/viper v1.16.0
	github.com/xiebiao/bookstore/proto/paymentv1 v0.0.0
	google.golang.org/grpc v1.59.0
	gorm.io/driver/mysql v1.5.1
	gorm.io/gorm v1.25.4
)

replace github.com/xiebiao/bookstore/proto/paymentv1 => ../../proto/paymentv1
