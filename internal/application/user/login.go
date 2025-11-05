package user

import (
	"context"
	"time"

	"github.com/xiebiao/bookstore/internal/domain/user"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	"github.com/xiebiao/bookstore/pkg/jwt"
)

// LoginUseCase 用户登录用例
// 设计说明：
// 1. 验证邮箱密码
// 2. 生成JWT Token对
// 3. 保存会话到Redis
type LoginUseCase struct {
	userService  user.Service
	jwtManager   *jwt.Manager
	sessionStore *redis.SessionStore
}

// NewLoginUseCase 创建登录用例
func NewLoginUseCase(
	userService user.Service,
	jwtManager *jwt.Manager,
	sessionStore *redis.SessionStore,
) *LoginUseCase {
	return &LoginUseCase{
		userService:  userService,
		jwtManager:   jwtManager,
		sessionStore: sessionStore,
	}
}

// Execute 执行登录
func (uc *LoginUseCase) Execute(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	// 1. 验证邮箱密码（调用领域服务）
	user, err := uc.userService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	// 2. 生成JWT Token对
	tokenPair, err := uc.jwtManager.GenerateToken(user.ID, user.Email, user.Nickname)
	if err != nil {
		return nil, err
	}

	// 3. 保存会话到Redis
	sessionData := map[string]interface{}{
		"user_id":  user.ID,
		"email":    user.Email,
		"nickname": user.Nickname,
		"login_at": time.Now().Unix(),
		"ip":       extractIP(ctx), // 可以从Context获取请求IP
	}

	// 会话有效期 = Refresh Token有效期
	if err := uc.sessionStore.SaveSession(ctx, user.ID, sessionData, 7*24*time.Hour); err != nil {
		// 会话保存失败不影响登录，只记录日志
		// TODO: 记录日志
	}

	// 4. 返回登录响应
	return &LoginResponse{
		User: UserInfo{
			ID:       user.ID,
			Email:    user.Email,
			Nickname: user.Nickname,
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// LogoutUseCase 用户登出用例
type LogoutUseCase struct {
	sessionStore *redis.SessionStore
}

// NewLogoutUseCase 创建登出用例
func NewLogoutUseCase(sessionStore *redis.SessionStore) *LogoutUseCase {
	return &LogoutUseCase{sessionStore: sessionStore}
}

// Execute 执行登出
func (uc *LogoutUseCase) Execute(ctx context.Context, userID uint, accessToken string) error {
	// 1. 删除会话
	if err := uc.sessionStore.DeleteSession(ctx, userID); err != nil {
		return err
	}

	// 2. 将Access Token加入黑名单（防止Token在过期前继续使用）
	if err := uc.sessionStore.AddToBlacklist(ctx, accessToken, 2*time.Hour); err != nil {
		return err
	}

	return nil
}

// =========================================
// 应用层DTO
// =========================================

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string
	Password string
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         UserInfo `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"` // Access Token过期时间（秒）
}

// UserInfo 用户信息
type UserInfo struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// =========================================
// 辅助函数
// =========================================

// extractIP 从Context提取IP地址
// 实际项目中可以从Context获取Gin的ClientIP
func extractIP(ctx context.Context) string {
	// TODO: 从Context获取真实IP
	return "127.0.0.1"
}
