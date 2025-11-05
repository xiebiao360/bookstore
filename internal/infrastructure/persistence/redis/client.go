package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/xiebiao/bookstore/internal/infrastructure/config"
)

// NewClient 创建Redis客户端
// 设计说明：
// 1. 配置连接池参数（PoolSize、MinIdleConns）
// 2. 配置超时参数（DialTimeout、ReadTimeout、WriteTimeout）
// 3. 测试连接可用性
func NewClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	fmt.Println("✓ Redis连接成功")
	return client, nil
}
