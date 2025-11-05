package user

import (
	"time"
)

// User 用户实体（聚合根）
// DDD设计说明：
// 1. User是用户聚合的根实体，包含用户的核心属性
// 2. 密码已加密存储（bcrypt），不应该有GetPassword()等方法暴露明文
// 3. 领域实体不依赖GORM tag（infrastructure层的Repository实现时会处理映射）
type User struct {
	ID        uint
	Email     string
	Password  string // bcrypt哈希值
	Nickname  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewUser 创建新用户（工厂方法）
// hashedPassword必须是bcrypt加密后的密码
func NewUser(email, hashedPassword, nickname string) *User {
	now := time.Now()
	return &User{
		Email:     email,
		Password:  hashedPassword,
		Nickname:  nickname,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsValidPassword 验证密码（由Service层调用bcrypt.CompareHashAndPassword）
// 说明：此方法仅作为占位，实际密码验证在Service层完成
func (u *User) IsValidPassword(plainPassword string) bool {
	// 实际实现在user.Service中，此处保留接口方法
	return false
}

// UpdateNickname 更新昵称（领域行为）
func (u *User) UpdateNickname(nickname string) {
	u.Nickname = nickname
	u.UpdatedAt = time.Now()
}
