package handler

import (
	"context"
	"errors"

	userapp "github.com/xiebiao/bookstore/internal/application/user"
	userdomain "github.com/xiebiao/bookstore/internal/domain/user"
	redisstore "github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
	"github.com/xiebiao/bookstore/pkg/jwt"
	pb "github.com/xiebiao/bookstore/proto/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserServiceServer gRPC服务实现
//
// 教学重点：
// 1. 复用Phase 1的UseCase（RegisterUseCase、LoginUseCase等）
// 2. gRPC Handler只做协议转换（Protobuf ↔ DTO）
// 3. 业务逻辑全部在UseCase和Domain层
//
// 架构对比：
// Phase 1: HTTP Handler (Gin) → UseCase → Domain Service → Repository
// Phase 2: gRPC Handler → UseCase → Domain Service → Repository (复用同一套逻辑)
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	registerUC   *userapp.RegisterUseCase
	loginUC      *userapp.LoginUseCase
	logoutUC     *userapp.LogoutUseCase
	jwtManager   *jwt.Manager             // JWT管理器（用于ValidateToken、RefreshToken）
	sessionStore *redisstore.SessionStore // 会话存储（用于检查黑名单、会话状态）
	userService  userdomain.Service       // 用户领域服务（用于GetUser）
}

// NewUserServiceServer 创建gRPC服务实例
func NewUserServiceServer(
	registerUC *userapp.RegisterUseCase,
	loginUC *userapp.LoginUseCase,
	logoutUC *userapp.LogoutUseCase,
	jwtManager *jwt.Manager,
	sessionStore *redisstore.SessionStore,
	userService userdomain.Service,
) *UserServiceServer {
	return &UserServiceServer{
		registerUC:   registerUC,
		loginUC:      loginUC,
		logoutUC:     logoutUC,
		jwtManager:   jwtManager,
		sessionStore: sessionStore,
		userService:  userService,
	}
}

// Register 用户注册
//
// 教学要点：
// 1. Protobuf Request → UseCase Request (DTO)
// 2. 调用UseCase
// 3. UseCase Response (DTO) → Protobuf Response
func (s *UserServiceServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// 步骤1: 协议转换 Protobuf → UseCase DTO
	ucReq := userapp.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	}

	// 步骤2: 调用UseCase（复用Phase 1业务逻辑）
	ucResp, err := s.registerUC.Execute(ctx, ucReq)
	if err != nil {
		// 错误处理
		return nil, status.Errorf(codes.Internal, "注册失败: %v", err)
	}

	// 步骤3: 协议转换 UseCase DTO → Protobuf
	// 注意：Phase 1的RegisterResponse不包含Token
	// 这里简化处理，实际应该调用LoginUseCase生成Token
	return &pb.RegisterResponse{
		Code:    0,
		Message: "注册成功",
		UserId:  uint64(ucResp.ID),
		Token:   "", // TODO: 生成Token
	}, nil
}

// Login 用户登录
func (s *UserServiceServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 步骤1: Protobuf → UseCase DTO
	ucReq := userapp.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// 步骤2: 调用UseCase
	ucResp, err := s.loginUC.Execute(ctx, ucReq)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "登录失败: %v", err)
	}

	// 步骤3: UseCase DTO → Protobuf
	return &pb.LoginResponse{
		Code:         0,
		Message:      "登录成功",
		UserId:       uint64(ucResp.User.ID),
		Token:        ucResp.AccessToken,
		RefreshToken: ucResp.RefreshToken,
	}, nil
}

// ValidateToken 验证Token（供其他服务调用）
//
// 教学要点：
// 1. 微服务间调用：order-service调用此接口验证用户身份
// 2. 双重验证：JWT签名验证 + Redis黑名单检查
// 3. 返回用户信息供调用方使用
//
// DO（正确做法）：
// - 先验证JWT签名（防止伪造）
// - 再检查黑名单（处理登出场景）
// - 返回详细的错误信息（便于调用方处理）
//
// DON'T（错误做法）：
// - 只验证JWT不检查黑名名单（用户登出后Token仍有效）
// - 抛出Internal错误（应该返回valid=false）
func (s *UserServiceServer) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	// 步骤1: 解析并验证JWT签名
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		// Token过期或无效
		// 注意：返回nil error，业务错误在Response.Valid中体现
		return &pb.ValidateTokenResponse{
			Valid: false,
			// Token无效时不返回用户信息
		}, nil
	}

	// 步骤2: 检查Token是否在黑名单中（用户已登出）
	inBlacklist, err := s.sessionStore.IsInBlacklist(ctx, req.Token)
	if err != nil {
		// Redis错误属于系统错误，返回gRPC错误
		return nil, status.Errorf(codes.Internal, "检查黑名单失败: %v", err)
	}
	if inBlacklist {
		return &pb.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	// 步骤3: Token有效，返回用户信息
	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: uint64(claims.UserID),
		Email:  claims.Email,
	}, nil
}

// GetUser 获取用户信息（供其他服务调用）
//
// 教学要点：
// 1. 微服务间调用场景：order-service需要获取用户昵称显示在订单中
// 2. 直接调用Domain Service（简单查询不需要UseCase封装）
// 3. 安全性：不返回密码等敏感信息
//
// 架构对比：
// Phase 1: HTTP Handler → UseCase → Domain Service → Repository
// Phase 2: gRPC Handler → Domain Service → Repository（简化）
//
// DO（正确做法）：
// - 返回用户基本信息（ID、Email、Nickname）
// - 区分"用户不存在"（NotFound）和"系统错误"（Internal）
// - 使用Protobuf定义清晰的返回结构
//
// DON'T（错误做法）：
// - 返回Password字段（安全风险）
// - 返回CreatedAt/UpdatedAt等内部字段（信息泄露）
// - 所有错误都返回Internal（调用方无法区分）
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// 调用Domain Service获取用户
	user, err := s.userService.GetByID(ctx, uint(req.UserId))
	if err != nil {
		// 判断错误类型，返回相应的gRPC状态码
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "用户不存在")
		}
		// 其他错误视为系统错误
		return nil, status.Errorf(codes.Internal, "查询用户失败: %v", err)
	}

	// 返回用户信息（不包含密码）
	// 注意：Protobuf中的User消息不包含password字段
	return &pb.GetUserResponse{
		Code:    0,
		Message: "查询成功",
		User: &pb.User{
			Id:       uint64(user.ID),
			Email:    user.Email,
			Nickname: user.Nickname,
			// 安全提示：Password字段不应该通过网络传输
		},
	}, nil
}

// RefreshToken 刷新Token
//
// 教学要点：
// 1. 双Token机制：Access Token（2小时）+ Refresh Token（7天）
// 2. 安全验证：Refresh Token有效性 + 用户会话存在性
// 3. 业务场景：前端Access Token过期后，用Refresh Token换取新的Access Token
//
// 工作流程：
// 步骤1: 验证Refresh Token签名和过期时间
// 步骤2: 检查用户会话是否存在（确保用户未登出）
// 步骤3: 生成新的Access Token（不生成新的Refresh Token）
//
// DO（正确做法）：
// - 验证Refresh Token有效性（防止伪造）
// - 检查用户会话（防止登出后仍能刷新）
// - 只返回新的Access Token（Refresh Token保持不变）
//
// DON'T（错误做法）：
// - 不检查会话直接刷新（用户登出后仍可刷新）
// - 每次刷新都生成新的Refresh Token（增加复杂度，无必要）
// - 不验证Refresh Token过期时间（安全风险）
//
// 安全考虑：
// - Refresh Token泄露的风险：攻击者可以持续刷新Access Token
// - 缓解措施：用户登出时删除会话，Refresh Token立即失效
// - 最佳实践：Refresh Token应存储在HttpOnly Cookie中（Phase 1实现）
func (s *UserServiceServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	// 步骤1: 验证Refresh Token签名和过期时间
	claims, err := s.jwtManager.ParseToken(req.RefreshToken)
	if err != nil {
		// Token过期或无效
		return nil, status.Errorf(codes.Unauthenticated, "Refresh Token无效: %v", err)
	}

	// 步骤2: 检查用户会话是否存在（确保用户未登出）
	// 重要：如果用户已登出，会话会被删除，此处会返回错误
	_, err = s.sessionStore.GetSession(ctx, claims.UserID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "会话已失效，请重新登录")
	}

	// 步骤3: 使用JWTManager生成新的Access Token
	// 注意：RefreshAccessToken方法会从Refresh Token的Claims中提取用户信息
	newAccessToken, err := s.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "刷新Token失败: %v", err)
	}

	// 返回新的Access Token
	// 注意：按照proto定义，需要返回code、message、token和refresh_token
	// 这里只刷新Access Token，Refresh Token保持不变（客户端继续使用原有的）
	return &pb.RefreshTokenResponse{
		Code:         0,
		Message:      "Token刷新成功",
		Token:        newAccessToken,
		RefreshToken: req.RefreshToken, // 返回原来的Refresh Token
	}, nil
}

// ============================================================
// 教学总结：UseCase模式的优势
// ============================================================
//
// 1. 单一职责：
//    - RegisterUseCase只负责注册流程
//    - LoginUseCase只负责登录流程
//
// 2. 可测试性：
//    - 每个UseCase可以独立测试
//    - Mock依赖（DomainService、Repository）
//
// 3. 可复用性：
//    - HTTP Handler和gRPC Handler共用同一套UseCase
//    - 未来可能有CLI、消息队列等其他入口
//
// 4. 扩展性：
//    - 未来可以在UseCase中添加：
//      * 事件发布（注册成功→发送欢迎邮件）
//      * 审计日志
//      * 分布式事务编排
//
// ============================================================
