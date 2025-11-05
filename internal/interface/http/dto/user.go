package dto

// RegisterRequest HTTP层注册请求
// 说明：HTTP层的DTO，包含参数验证tag
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}

// UserResponse 用户响应（不包含密码）
type UserResponse struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}
