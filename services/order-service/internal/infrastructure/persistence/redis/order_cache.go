package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/config"
)

// InitRedis 初始化Redis连接
//
// 教学要点：
// 1. Redis客户端类型：
//   - redis.Client：单机模式
//   - redis.ClusterClient：集群模式
//   - redis.Ring：分片模式
//
// 2. Phase 2使用单机模式，Phase 3会升级到集群
// 3. 连接池配置：
//   - PoolSize：最大连接数
//   - MinIdleConns：最小空闲连接（保持热连接）
func InitRedis(cfg *config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		// 连接超时配置
		DialTimeout:  5 * time.Second, // 建立连接超时
		ReadTimeout:  3 * time.Second, // 读取超时
		WriteTimeout: 3 * time.Second, // 写入超时
		// 连接池超时
		PoolTimeout: 4 * time.Second, // 从连接池获取连接的超时
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Redis连接失败: %v", err))
	}

	fmt.Println("✅ Redis连接成功")
	return client
}

// OrderCache 订单缓存接口
//
// 教学要点：
// 为什么需要订单缓存？
// 1. 性能优化：
//   - 热点订单（刚下单的订单）查询频繁
//   - Redis QPS > 100K，MySQL QPS < 10K
//   - 减轻数据库压力
//
// 2. 用户体验：
//   - 订单详情页秒开（< 50ms）
//   - 订单列表快速加载
//
// 3. 缓存策略：
//   - Cache-Aside：查询时先查缓存，未命中再查数据库
//   - TTL：5分钟（平衡命中率和数据新鲜度）
//
// 不缓存的场景：
// - 订单列表（数据量大，缓存效率低）
// - 历史订单（访问频率低）
type OrderCache interface {
	// GetOrder 获取订单缓存
	GetOrder(ctx context.Context, orderID uint) (string, error)

	// SetOrder 设置订单缓存
	SetOrder(ctx context.Context, orderID uint, orderJSON string, ttl time.Duration) error

	// DeleteOrder 删除订单缓存（更新/删除订单时调用）
	DeleteOrder(ctx context.Context, orderID uint) error

	// SetPendingOrder 添加到待支付订单集合（用于超时取消）
	// 教学要点：
	// 使用ZSet（有序集合）存储待支付订单
	// - score：订单创建时间 + 超时时间（时间戳）
	// - member：订单ID
	// 定时任务扫描：ZRANGEBYSCORE 0 当前时间
	SetPendingOrder(ctx context.Context, orderID uint, expireAt time.Time) error

	// GetExpiredOrders 获取已超时的订单ID列表
	GetExpiredOrders(ctx context.Context, limit int) ([]uint, error)

	// RemovePendingOrder 从待支付集合中移除（支付成功或已取消）
	RemovePendingOrder(ctx context.Context, orderID uint) error
}

type orderCache struct {
	client *redis.Client
}

// NewOrderCache 创建订单缓存实例
func NewOrderCache(client *redis.Client) OrderCache {
	return &orderCache{client: client}
}

// orderCacheKey 生成订单缓存键
//
// 教学要点：
// Redis Key命名规范：
// - 前缀：业务模块（order）
// - 实体：实体类型（detail）
// - 标识：实体ID
// - 示例：order:detail:123
//
// 好处：
// 1. 可读性：一目了然知道key的含义
// 2. 避免冲突：不同模块的key不会重复
// 3. 便于管理：KEYS order:* 查看所有订单相关key
func orderCacheKey(orderID uint) string {
	return fmt.Sprintf("order:detail:%d", orderID)
}

// pendingOrdersKey 待支付订单集合key
//
// 教学要点：
// 使用单个ZSet存储所有待支付订单
// - 好处：批量扫描效率高（一次ZRANGEBYSCORE）
// - 坏处：单key可能成为热点
// - 优化方案（Phase 3）：按时间分片（order:pending:20251106）
const pendingOrdersKey = "order:pending:zset"

// GetOrder 获取订单缓存
func (c *orderCache) GetOrder(ctx context.Context, orderID uint) (string, error) {
	key := orderCacheKey(orderID)

	// GET命令
	// 教学要点：
	// Result()返回(string, error)：
	// - 成功：返回值和nil
	// - Key不存在：返回""和redis.Nil
	// - 其他错误：返回""和error
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key不存在，返回空字符串（非错误）
			return "", nil
		}
		return "", fmt.Errorf("获取订单缓存失败: %w", err)
	}

	return val, nil
}

// SetOrder 设置订单缓存
//
// 教学要点：
// 为什么存储JSON字符串而非HASH？
// - JSON：单次操作，Get一次拿到完整数据
// - HASH：需要HGETALL，多次网络往返
// - 订单字段少（< 20个），JSON序列化开销小
//
// DO vs DON'T:
// ❌ DON'T: 永久缓存（Set key value）
//   - 内存无限增长
//   - 数据永远不会更新
//
// ✅ DO: 设置合理的TTL（SetEX key ttl value）
//   - 自动过期，释放内存
//   - 下次查询时重新加载最新数据
func (c *orderCache) SetOrder(ctx context.Context, orderID uint, orderJSON string, ttl time.Duration) error {
	key := orderCacheKey(orderID)

	// SETEX命令：SET key value EX ttl
	// 教学要点：
	// SetEX vs Set + Expire：
	// - SetEX：原子操作，一次命令完成
	// - Set + Expire：两次命令，中间可能宕机导致永久缓存
	err := c.client.SetEX(ctx, key, orderJSON, ttl).Err()
	if err != nil {
		return fmt.Errorf("设置订单缓存失败: %w", err)
	}

	return nil
}

// DeleteOrder 删除订单缓存
//
// 教学要点：
// 何时删除缓存？
// 1. 订单状态更新：避免缓存脏数据
// 2. 订单取消：立即失效缓存
// 3. 订单删除：清理无用数据
//
// Cache Invalidation策略：
// - Write Through：更新时同时更新缓存（复杂，容易出错）
// - Cache Aside：更新时删除缓存（简单，推荐）
func (c *orderCache) DeleteOrder(ctx context.Context, orderID uint) error {
	key := orderCacheKey(orderID)

	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除订单缓存失败: %w", err)
	}

	return nil
}

// SetPendingOrder 添加到待支付订单集合
//
// 教学要点：
// 使用ZSet（Sorted Set）实现定时任务：
// 1. 数据结构：
//   - member：订单ID（字符串）
//   - score：过期时间戳（float64）
//     2. 示例：
//     ZADD order:pending:zset 1699257600 "123"
//     含义：订单123将在2023-11-06 12:00:00过期
//     3. 查询过期订单：
//     ZRANGEBYSCORE order:pending:zset 0 当前时间戳
//     返回：所有score <= 当前时间的订单ID
//
// 为什么用ZSet而非List？
// - List：无法按时间排序，需要遍历所有元素
// - ZSet：O(log N)复杂度，快速查询范围
func (c *orderCache) SetPendingOrder(ctx context.Context, orderID uint, expireAt time.Time) error {
	// ZAdd：添加到有序集合
	// Z结构：
	// - Score：过期时间戳（用于排序和范围查询）
	// - Member：订单ID
	member := &redis.Z{
		Score:  float64(expireAt.Unix()), // 时间戳作为score
		Member: fmt.Sprintf("%d", orderID),
	}

	err := c.client.ZAdd(ctx, pendingOrdersKey, member).Err()
	if err != nil {
		return fmt.Errorf("添加待支付订单失败: %w", err)
	}

	return nil
}

// GetExpiredOrders 获取已超时的订单ID列表
//
// 教学要点：
// ZRANGEBYSCORE命令：
// - 按score范围查询
// - 语法：ZRANGEBYSCORE key min max [LIMIT offset count]
// - 示例：ZRANGEBYSCORE order:pending:zset 0 1699257600 LIMIT 0 100
//
// 为什么需要limit？
// - 防止一次返回过多数据（可能有上万个超时订单）
// - 分批处理，避免长时间阻塞
// - 定时任务每次处理100个，下次继续处理剩余的
func (c *orderCache) GetExpiredOrders(ctx context.Context, limit int) ([]uint, error) {
	now := time.Now().Unix()

	// ZRANGEBYSCORE key 0 now LIMIT 0 limit
	// 查询score在[0, now]范围内的前limit个member
	vals, err := c.client.ZRangeByScore(ctx, pendingOrdersKey, &redis.ZRangeBy{
		Min:    "0",                    // 最小score（从最早的订单开始）
		Max:    fmt.Sprintf("%d", now), // 最大score（当前时间）
		Offset: 0,                      // 从第0个开始
		Count:  int64(limit),           // 最多返回limit个
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("获取超时订单失败: %w", err)
	}

	// 将字符串转换为uint
	// 教学要点：
	// Redis存储的是字符串，需要手动转换类型
	orderIDs := make([]uint, 0, len(vals))
	for _, val := range vals {
		var id uint
		// 从JSON解析（如果member存储的是JSON）
		// 这里简化为直接解析整数
		if _, err := fmt.Sscanf(val, "%d", &id); err != nil {
			// 跳过无效数据
			continue
		}
		orderIDs = append(orderIDs, id)
	}

	return orderIDs, nil
}

// RemovePendingOrder 从待支付集合中移除
//
// 教学要点：
// 何时调用此方法？
// 1. 用户支付成功：订单变为已支付状态
// 2. 用户主动取消：订单变为已取消状态
// 3. 系统自动取消：超时取消任务执行后
//
// ZREM命令：
// - 从有序集合中移除指定member
// - 如果member不存在，返回0（不报错）
func (c *orderCache) RemovePendingOrder(ctx context.Context, orderID uint) error {
	member := fmt.Sprintf("%d", orderID)

	err := c.client.ZRem(ctx, pendingOrdersKey, member).Err()
	if err != nil {
		return fmt.Errorf("移除待支付订单失败: %w", err)
	}

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// MarshalOrder 将订单对象序列化为JSON
//
// 教学要点：
// 为什么单独封装序列化函数？
// - 统一处理：所有序列化逻辑集中管理
// - 便于扩展：后续可以添加压缩、加密等
// - 错误处理：统一的错误格式
func MarshalOrder(order interface{}) (string, error) {
	data, err := json.Marshal(order)
	if err != nil {
		return "", fmt.Errorf("订单序列化失败: %w", err)
	}
	return string(data), nil
}

// UnmarshalOrder 将JSON反序列化为订单对象
func UnmarshalOrder(data string, order interface{}) error {
	if err := json.Unmarshal([]byte(data), order); err != nil {
		return fmt.Errorf("订单反序列化失败: %w", err)
	}
	return nil
}
