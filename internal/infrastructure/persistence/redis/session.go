package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// SessionStore 会话存储
// 设计说明：
// 1. 使用Redis存储用户登录会话
// 2. 支持JWT黑名单（用户登出、强制下线）
// 3. Key设计：session:{user_id}、blacklist:{token}
type SessionStore struct {
	client *redis.Client
}

// NewSessionStore 创建会话存储
func NewSessionStore(client *redis.Client) *SessionStore {
	return &SessionStore{client: client}
}

// SaveSession 保存用户会话
// 学习要点：
// 1. 存储用户登录信息（登录时间、IP地址等）
// 2. 设置过期时间（与Refresh Token一致）
// 3. 可用于统计在线用户、强制下线等场景
func (s *SessionStore) SaveSession(ctx context.Context, userID uint, sessionData map[string]interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("session:%d", userID)

	// 使用HMSet存储多个字段
	if err := s.client.HMSet(ctx, key, sessionData).Err(); err != nil {
		return apperrors.Wrap(err, "保存会话失败")
	}

	// 设置过期时间
	if err := s.client.Expire(ctx, key, ttl).Err(); err != nil {
		return apperrors.Wrap(err, "设置会话过期时间失败")
	}

	return nil
}

// GetSession 获取用户会话
func (s *SessionStore) GetSession(ctx context.Context, userID uint) (map[string]string, error) {
	key := fmt.Sprintf("session:%d", userID)

	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, apperrors.Wrap(err, "获取会话失败")
	}

	if len(result) == 0 {
		return nil, apperrors.ErrUnauthorized
	}

	return result, nil
}

// DeleteSession 删除用户会话（用于登出）
func (s *SessionStore) DeleteSession(ctx context.Context, userID uint) error {
	key := fmt.Sprintf("session:%d", userID)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return apperrors.Wrap(err, "删除会话失败")
	}

	return nil
}

// AddToBlacklist 将Token加入黑名单
// 使用场景：
// 1. 用户登出
// 2. Token泄露后强制失效
// 3. 用户修改密码后强制所有Token失效
func (s *SessionStore) AddToBlacklist(ctx context.Context, token string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", token)

	// 存储到黑名单，值可以是失效原因
	if err := s.client.Set(ctx, key, "revoked", ttl).Err(); err != nil {
		return apperrors.Wrap(err, "添加Token到黑名单失败")
	}

	return nil
}

// IsInBlacklist 检查Token是否在黑名单中
func (s *SessionStore) IsInBlacklist(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)

	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, apperrors.Wrap(err, "检查黑名单失败")
	}

	return exists > 0, nil
}

// =========================================
// 学习要点总结
// =========================================
//
// 1. 为什么需要会话存储？
//    - JWT是无状态的，服务端无法主动让Token失效
//    - 通过Redis黑名单机制，可以实现Token的主动失效
//    - 可以记录用户登录设备、IP、时间等信息
//
// 2. Key设计规范
//    - session:{user_id}: 用户会话信息
//    - blacklist:{token}: Token黑名单
//    - 使用冒号分隔命名空间，便于管理和监控
//
// 3. 过期时间策略
//    - session过期时间 = Refresh Token有效期（7天）
//    - blacklist过期时间 = Access Token有效期（2小时）
//    - 过期后自动删除，无需手动清理
//
// 4. 性能优化
//    - 使用HMSet批量设置多个字段（减少网络往返）
//    - 合理设置连接池大小（PoolSize）
//    - 使用Pipeline批量操作（高并发场景）
