package user

import (
	"context"
)

// Repository 用户仓储接口
// DDD设计说明：
// 1. 接口定义在domain层（依赖倒置原则）
// 2. 具体实现在infrastructure/persistence/mysql层
// 3. 这样domain层不依赖任何外部框架（GORM、sqlx等）
// 4. 便于单元测试（Mock此接口）
type Repository interface {
	// Create 创建用户
	// 注意：如果邮箱已存在，应返回errors.ErrEmailDuplicate
	Create(ctx context.Context, user *User) error

	// FindByID 根据ID查找用户
	// 如果不存在，返回errors.ErrUserNotFound
	FindByID(ctx context.Context, id uint) (*User, error)

	// FindByEmail 根据邮箱查找用户
	// 如果不存在，返回errors.ErrUserNotFound
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Update 更新用户信息
	Update(ctx context.Context, user *User) error

	// Delete 删除用户（软删除）
	Delete(ctx context.Context, id uint) error
}
