package user

import (
	"context"

	"github.com/xiebiao/bookstore/internal/domain/user"
)

// RegisterUseCase 用户注册用例
// 设计说明：
// 1. Application层负责用例编排，协调多个领域服务
// 2. 当前注册用例比较简单，只调用一个领域服务
// 3. 未来可能扩展：发送欢迎邮件、记录审计日志、触发事件等
type RegisterUseCase struct {
	userService user.Service
}

// NewRegisterUseCase 创建注册用例
func NewRegisterUseCase(userService user.Service) *RegisterUseCase {
	return &RegisterUseCase{
		userService: userService,
	}
}

// Execute 执行注册
// 返回：RegisterResponse（应用层DTO，不是领域实体）
func (uc *RegisterUseCase) Execute(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	// 1. 调用领域服务执行注册
	user, err := uc.userService.Register(ctx, req.Email, req.Password, req.Nickname)
	if err != nil {
		return nil, err
	}

	// 2. 领域实体 → 应用层DTO
	// 说明：不直接返回领域实体，而是转换为DTO
	// 好处：领域模型变更不影响API契约
	return &RegisterResponse{
		ID:       user.ID,
		Email:    user.Email,
		Nickname: user.Nickname,
	}, nil
}

// =========================================
// 应用层DTO（数据传输对象）
// =========================================

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string
	Password string
	Nickname string
}

// RegisterResponse 注册响应
// 说明：不返回密码字段（安全考虑）
type RegisterResponse struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}
