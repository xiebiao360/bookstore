package mysql

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/xiebiao/bookstore/internal/domain/user"
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// userRepository 用户仓储实现（MySQL）
// 设计说明：
// 1. 实现domain/user/repository.go定义的接口
// 2. 负责domain实体与GORM模型之间的转换
// 3. 处理数据库特定的错误（如邮箱重复），转换为业务错误
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储
// 注意：返回的是domain层的接口类型，不是具体类型（依赖倒置）
func NewUserRepository(db *gorm.DB) user.Repository {
	return &userRepository{db: db}
}

// Create 创建用户
// 学习要点：
// 1. 邮箱唯一性由数据库UNIQUE索引保证（而非应用层SELECT再INSERT）
// 2. 捕获MySQL的Duplicate Entry错误，转换为业务错误ErrEmailDuplicate
func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	// 1. 领域实体 → GORM模型
	model := &UserModel{
		Email:    u.Email,
		Password: u.Password,
		Nickname: u.Nickname,
	}

	// 2. 插入数据库
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		// 检查是否为邮箱重复错误
		// MySQL错误码1062: Duplicate entry
		if isDuplicateError(err) {
			return apperrors.ErrEmailDuplicate
		}
		return apperrors.Wrap(err, "创建用户失败")
	}

	// 3. 回填自增ID（GORM自动填充）
	u.ID = model.ID
	u.CreatedAt = model.CreatedAt
	u.UpdatedAt = model.UpdatedAt

	return nil
}

// FindByID 根据ID查找用户
func (r *userRepository) FindByID(ctx context.Context, id uint) (*user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).First(&model, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "查询用户失败")
	}

	return toEntity(&model), nil
}

// FindByEmail 根据邮箱查找用户
// 学习要点：
// 1. 邮箱字段有UNIQUE索引，查询效率高
// 2. 使用First而非Find，因为只需要一条记录
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "查询用户失败")
	}

	return toEntity(&model), nil
}

// Update 更新用户信息
func (r *userRepository) Update(ctx context.Context, u *user.User) error {
	model := &UserModel{
		ID:       u.ID,
		Email:    u.Email,
		Password: u.Password,
		Nickname: u.Nickname,
	}

	// 使用Save更新所有字段
	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return apperrors.Wrap(err, "更新用户失败")
	}

	u.UpdatedAt = model.UpdatedAt
	return nil
}

// Delete 删除用户（软删除）
// 学习要点：
// 1. GORM的软删除：DELETE操作会自动变成UPDATE deleted_at
// 2. 后续查询会自动过滤deleted_at不为NULL的记录
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&UserModel{}, id)

	if result.Error != nil {
		return apperrors.Wrap(result.Error, "删除用户失败")
	}

	if result.RowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

// =========================================
// 辅助函数：模型转换
// =========================================

// toEntity GORM模型 → 领域实体
// 说明：这是Repository的重要职责之一，隔离infrastructure层与domain层
func toEntity(model *UserModel) *user.User {
	return &user.User{
		ID:        model.ID,
		Email:     model.Email,
		Password:  model.Password,
		Nickname:  model.Nickname,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

// isDuplicateError 判断是否为MySQL唯一索引冲突错误
// MySQL错误码：
// - 1062: Duplicate entry 'xxx' for key 'yyy'
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	// GORM v2的错误判断
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// 兼容检查：错误信息包含"Duplicate entry"
	return strings.Contains(err.Error(), "Duplicate entry")
}
