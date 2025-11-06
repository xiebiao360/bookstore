package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// Lua脚本嵌入
// 教学要点：使用embed将Lua脚本嵌入到二进制文件
//
// 为什么Lua脚本放在redis包内？
// - go:embed不允许使用..向上引用（安全限制）
// - 只能引用当前包及子目录的文件
// - 实践中，将脚本与使用它的代码放在一起更符合内聚性原则
//
//go:embed deduct_stock.lua
var deductStockLua string

//go:embed release_stock.lua
var releaseStockLua string

//go:embed restock_inventory.lua
var restockInventoryLua string

// InventoryStore Redis库存存储
//
// 教学要点：
// 1. Redis作为高性能缓存层
//   - 热点数据存储（库存数量）
//   - Lua脚本保证原子性
//   - TPS > 10000（远超MySQL）
//
// 2. 数据一致性策略
//   - Redis为主（实时库存）
//   - MySQL为辅（持久化、对账）
//   - 定时同步Redis → MySQL
//
// 3. Key设计规范
//   - stock:{book_id}：库存数量
//   - deduct:stock:{book_id}:{order_id}：扣减记录（幂等性）
//   - release:stock:{book_id}:{order_id}：释放记录（幂等性）
type InventoryStore struct {
	client *redis.Client

	// Lua脚本SHA1（预加载优化）
	deductScriptSHA  string
	releaseScriptSHA string
	restockScriptSHA string
}

// NewInventoryStore 创建Redis库存存储实例
func NewInventoryStore(client *redis.Client) *InventoryStore {
	return &InventoryStore{
		client: client,
	}
}

// LoadScripts 预加载Lua脚本
//
// 教学要点：
// 1. SCRIPT LOAD预加载脚本到Redis
// 2. 后续使用EVALSHA调用（减少网络传输）
// 3. 性能优化：EVAL传输整个脚本，EVALSHA只传输SHA1
func (s *InventoryStore) LoadScripts(ctx context.Context) error {
	// 加载扣减脚本
	deductSHA, err := s.client.ScriptLoad(ctx, deductStockLua).Result()
	if err != nil {
		return fmt.Errorf("加载扣减脚本失败: %w", err)
	}
	s.deductScriptSHA = deductSHA

	// 加载释放脚本
	releaseSHA, err := s.client.ScriptLoad(ctx, releaseStockLua).Result()
	if err != nil {
		return fmt.Errorf("加载释放脚本失败: %w", err)
	}
	s.releaseScriptSHA = releaseSHA

	// 加载补货脚本
	restockSHA, err := s.client.ScriptLoad(ctx, restockInventoryLua).Result()
	if err != nil {
		return fmt.Errorf("加载补货脚本失败: %w", err)
	}
	s.restockScriptSHA = restockSHA

	return nil
}

// GetStock 获取库存
func (s *InventoryStore) GetStock(ctx context.Context, bookID uint) (int, error) {
	key := s.stockKey(bookID)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// 库存不存在，返回0
			return 0, nil
		}
		return 0, fmt.Errorf("获取库存失败: %w", err)
	}

	stock, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("解析库存失败: %w", err)
	}

	return stock, nil
}

// SetStock 设置库存
// 教学要点：初始化库存或同步MySQL库存到Redis
func (s *InventoryStore) SetStock(ctx context.Context, bookID uint, stock int) error {
	key := s.stockKey(bookID)

	if err := s.client.Set(ctx, key, stock, 0).Err(); err != nil {
		return fmt.Errorf("设置库存失败: %w", err)
	}

	return nil
}

// DeductStock 扣减库存（使用Lua脚本）
//
// 教学要点：
//  1. Lua脚本保证原子性
//  2. 幂等性控制（防止重复扣减）
//  3. 返回值含义：
//     0: 库存不足
//     1: 扣减成功
//     2: 重复扣减
func (s *InventoryStore) DeductStock(ctx context.Context, bookID uint, quantity int, orderID uint) (int, error) {
	key := s.stockKey(bookID)

	// 执行Lua脚本
	result, err := s.client.EvalSha(ctx, s.deductScriptSHA, []string{key}, quantity, orderID).Result()
	if err != nil {
		return 0, fmt.Errorf("执行扣减脚本失败: %w", err)
	}

	// 转换结果
	code, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("脚本返回值类型错误: %T", result)
	}

	return int(code), nil
}

// ReleaseStock 释放库存（使用Lua脚本）
func (s *InventoryStore) ReleaseStock(ctx context.Context, bookID uint, quantity int, orderID uint) (int, error) {
	key := s.stockKey(bookID)

	result, err := s.client.EvalSha(ctx, s.releaseScriptSHA, []string{key}, quantity, orderID).Result()
	if err != nil {
		return 0, fmt.Errorf("执行释放脚本失败: %w", err)
	}

	code, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("脚本返回值类型错误: %T", result)
	}

	return int(code), nil
}

// RestockInventory 补充库存（使用Lua脚本）
func (s *InventoryStore) RestockInventory(ctx context.Context, bookID uint, quantity int) (int, error) {
	key := s.stockKey(bookID)

	result, err := s.client.EvalSha(ctx, s.restockScriptSHA, []string{key}, quantity).Result()
	if err != nil {
		return 0, fmt.Errorf("执行补货脚本失败: %w", err)
	}

	newStock, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("脚本返回值类型错误: %T", result)
	}

	return int(newStock), nil
}

// BatchGetStock 批量获取库存
//
// 教学要点：
// 1. 使用Pipeline批量查询（减少网络往返）
// 2. 一次性查询多个库存
func (s *InventoryStore) BatchGetStock(ctx context.Context, bookIDs []uint) (map[uint]int, error) {
	if len(bookIDs) == 0 {
		return make(map[uint]int), nil
	}

	// 使用Pipeline批量查询
	pipe := s.client.Pipeline()
	cmds := make(map[uint]*redis.StringCmd, len(bookIDs))

	for _, bookID := range bookIDs {
		key := s.stockKey(bookID)
		cmds[bookID] = pipe.Get(ctx, key)
	}

	// 执行Pipeline
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("批量查询库存失败: %w", err)
	}

	// 解析结果
	result := make(map[uint]int, len(bookIDs))
	for bookID, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				result[bookID] = 0 // 库存不存在，返回0
				continue
			}
			// 单个查询失败不影响整体
			result[bookID] = 0
			continue
		}

		stock, err := strconv.Atoi(val)
		if err != nil {
			result[bookID] = 0
			continue
		}

		result[bookID] = stock
	}

	return result, nil
}

// stockKey 生成库存键
// 格式：stock:{book_id}
func (s *InventoryStore) stockKey(bookID uint) string {
	return fmt.Sprintf("stock:%d", bookID)
}

// DeleteStock 删除库存（用于测试）
func (s *InventoryStore) DeleteStock(ctx context.Context, bookID uint) error {
	key := s.stockKey(bookID)
	return s.client.Del(ctx, key).Err()
}
