package user

import (
	"context"
	"regexp"

	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// Service 用户领域服务
// 设计说明：
// 1. Service包含不属于单个实体的业务逻辑（如密码加密、验证）
// 2. Service依赖Repository接口，不依赖具体实现（依赖倒置）
// 3. Service不处理HTTP请求，只处理业务逻辑
type Service interface {
	// Register 用户注册
	Register(ctx context.Context, email, password, nickname string) (*User, error)

	// Login 用户登录
	Login(ctx context.Context, email, password string) (*User, error)

	// ValidatePassword 验证密码
	ValidatePassword(hashedPassword, plainPassword string) error
}

type service struct {
	repo Repository
}

// NewService 创建用户服务
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Register 用户注册
// 业务规则：
// 1. 邮箱格式校验
// 2. 密码强度校验（8-20位，包含字母和数字）
// 3. 密码bcrypt加密（cost=12）
// 4. 邮箱唯一性由数据库UNIQUE索引保证
func (s *service) Register(ctx context.Context, email, password, nickname string) (*User, error) {
	// 1. 邮箱格式校验
	if !isValidEmail(email) {
		return nil, apperrors.New(apperrors.ErrCodeInvalidParams, "邮箱格式不正确")
	}

	// 2. 密码强度校验
	if err := validatePasswordStrength(password); err != nil {
		return nil, err
	}

	// 3. 昵称校验
	if len(nickname) < 2 || len(nickname) > 50 {
		return nil, apperrors.New(apperrors.ErrCodeInvalidParams, "昵称长度应为2-50个字符")
	}

	// 4. 密码加密
	// 学习要点：
	// - bcrypt自动加盐，每次加密结果都不同（即使密码相同）
	// - cost=12是推荐值，平衡安全性与性能（cost每+1，耗时翻倍）
	// - 不要使用MD5/SHA1，已被证明不安全（彩虹表攻击）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, apperrors.Wrap(err, "密码加密失败")
	}

	// 5. 创建用户实体
	user := NewUser(email, string(hashedPassword), nickname)

	// 6. 持久化到数据库
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err // Repository已转换为业务错误
	}

	return user, nil
}

// Login 用户登录
// 业务规则：
// 1. 邮箱必须存在
// 2. 密码必须正确
func (s *service) Login(ctx context.Context, email, password string) (*User, error) {
	// 1. 根据邮箱查找用户
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err // Repository已转换为ErrUserNotFound
	}

	// 2. 验证密码
	if err := s.ValidatePassword(user.Password, password); err != nil {
		return nil, err // 返回ErrInvalidPassword
	}

	return user, nil
}

// ValidatePassword 验证密码
// 说明：登录时使用，验证明文密码与哈希值是否匹配
func (s *service) ValidatePassword(hashedPassword, plainPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return apperrors.ErrInvalidPassword
		}
		return apperrors.Wrap(err, "密码验证失败")
	}
	return nil
}

// =========================================
// 辅助函数：业务规则校验
// =========================================

// isValidEmail 邮箱格式校验
// 简单的正则校验，生产环境可使用更严格的RFC 5322标准
func isValidEmail(email string) bool {
	// 正则表达式：用户名@域名.后缀
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// validatePasswordStrength 密码强度校验
// 规则：8-20位，必须包含字母和数字
func validatePasswordStrength(password string) error {
	// 长度校验
	if len(password) < 8 || len(password) > 20 {
		return apperrors.ErrWeakPassword
	}

	// 必须包含字母
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	// 必须包含数字
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter || !hasDigit {
		return apperrors.ErrWeakPassword
	}

	return nil
}

// =========================================
// 学习要点总结
// =========================================
//
// 1. 为什么使用bcrypt而非MD5？
//    - MD5是哈希算法，不是加密算法，无法解密
//    - MD5没有加盐，相同密码哈希值相同，容易被彩虹表攻击
//    - bcrypt自动加盐，且计算缓慢（抵抗暴力破解）
//
// 2. bcrypt的cost参数如何选择？
//    - cost=10: ~70ms（适合高并发场景）
//    - cost=12: ~250ms（推荐值，平衡安全与性能）
//    - cost=14: ~1s（高安全要求场景）
//    - cost每+1，耗时翻倍，根据硬件性能调整
//
// 3. 为什么邮箱唯一性不在Service层校验？
//    - 应用层校验存在并发问题（SELECT再INSERT有时间窗口）
//    - 数据库UNIQUE索引能保证原子性
//    - Repository捕获数据库错误，转换为ErrEmailDuplicate
//
// 4. 密码明文何时清除？
//    - Register完成后，password参数会被Go的GC回收
//    - 数据库只存储hashedPassword
//    - 日志中不应记录密码相关信息
