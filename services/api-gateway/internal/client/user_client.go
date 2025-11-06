package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userv1 "github.com/xiebiao/bookstore/proto/userv1"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/config"
)

// UserClient user-service gRPC客户端封装
//
// 教学要点：
// 1. 封装gRPC客户端，隐藏连接管理细节
// 2. 统一超时控制（每个RPC调用都带Context超时）
// 3. 错误处理转换（gRPC错误 → 业务错误）
//
// 架构对比：
// Phase 1: 直接调用本地Service
// Phase 2: 通过gRPC调用远程Service（跨进程、跨网络）
type UserClient struct {
	client  userv1.UserServiceClient
	conn    *grpc.ClientConn
	timeout time.Duration
}

// NewUserClient 创建user-service客户端
//
// 教学说明：
// 1. Phase 2 Week 5: 使用直连模式（grpc.Dial）
// 2. Phase 2 Week 6: 将改为服务发现（grpc.Dial("consul://user-service")）
func NewUserClient(cfg config.ServiceConfig) (*UserClient, error) {
	// 步骤1: 连接gRPC服务器
	// 教学重点：
	// - insecure.NewCredentials(): 不使用TLS（开发环境）
	// - 生产环境应使用TLS证书
	conn, err := grpc.NewClient(
		cfg.Addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 后续添加：
		// grpc.WithBalancerName("round_robin"), // 负载均衡
		// grpc.WithUnaryInterceptor(interceptor), // 拦截器（日志、监控）
	)
	if err != nil {
		return nil, fmt.Errorf("连接user-service失败: %w", err)
	}

	// 步骤2: 创建gRPC客户端
	client := userv1.NewUserServiceClient(conn)

	return &UserClient{
		client:  client,
		conn:    conn,
		timeout: cfg.GetTimeout(),
	}, nil
}

// Close 关闭连接
func (c *UserClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Register 用户注册
//
// 教学要点：
// 1. 每个RPC调用都创建带超时的Context
// 2. gRPC调用可能失败（网络、服务宕机），需要错误处理
func (c *UserClient) Register(ctx context.Context, email, password, nickname string) (*userv1.RegisterResponse, error) {
	// 创建带超时的Context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 调用gRPC方法
	resp, err := c.client.Register(ctx, &userv1.RegisterRequest{
		Email:    email,
		Password: password,
		Nickname: nickname,
	})
	if err != nil {
		return nil, fmt.Errorf("注册失败: %w", err)
	}

	return resp, nil
}

// Login 用户登录
func (c *UserClient) Login(ctx context.Context, email, password string) (*userv1.LoginResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.Login(ctx, &userv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("登录失败: %w", err)
	}

	return resp, nil
}

// ValidateToken 验证Token
//
// 教学说明：
// Gateway的认证中间件调用此方法验证Token
func (c *UserClient) ValidateToken(ctx context.Context, token string) (*userv1.ValidateTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.ValidateToken(ctx, &userv1.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, fmt.Errorf("验证Token失败: %w", err)
	}

	return resp, nil
}

// GetUser 获取用户信息
func (c *UserClient) GetUser(ctx context.Context, userID uint64) (*userv1.GetUserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetUser(ctx, &userv1.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}

	return resp, nil
}

// RefreshToken 刷新Token
func (c *UserClient) RefreshToken(ctx context.Context, refreshToken string) (*userv1.RefreshTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.RefreshToken(ctx, &userv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("刷新Token失败: %w", err)
	}

	return resp, nil
}

// =========================================
// 教学总结：gRPC客户端最佳实践
// =========================================
//
// 1. 连接管理：
//    - 连接复用（一个服务一个连接）
//    - 优雅关闭（Close方法）
//    - 连接池（高并发场景可考虑）
//
// 2. 超时控制：
//    - 每个RPC调用都要设置超时
//    - 避免雪崩（一个慢请求阻塞所有goroutine）
//    - 超时时间应小于HTTP请求超时
//
// 3. 错误处理：
//    - gRPC错误码映射到业务错误
//    - 区分网络错误、服务错误、业务错误
//    - 后续添加重试机制（幂等操作）
//
// 4. 扩展点：
//    - 拦截器（日志、监控、链路追踪）
//    - 负载均衡（多实例）
//    - 熔断降级（Sentinel）
