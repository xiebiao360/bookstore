package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/domain/book"
)

// CacheStore Redis缓存存储
//
// 教学要点：
// 1. 缓存的作用
//   - 减少数据库压力（热点数据）
//   - 提升响应速度（内存读取远快于磁盘）
//   - 应对流量峰值（大促、热点事件）
//
// 2. 缓存策略
//   - Cache-Aside（旁路缓存）：先查缓存，未命中再查数据库
//   - Read-Through：缓存层自动加载数据
//   - Write-Through：写缓存同时写数据库
//   - Write-Behind：异步写数据库
//
// 3. 缓存一致性问题
//   - 更新数据库后删除缓存（推荐）
//   - 更新数据库后更新缓存（可能出现并发问题）
type CacheStore struct {
	client    *redis.Client
	listTTL   time.Duration
	detailTTL time.Duration
	searchTTL time.Duration
}

// NewCacheStore 创建缓存存储实例
func NewCacheStore(client *redis.Client, listTTL, detailTTL, searchTTL time.Duration) *CacheStore {
	return &CacheStore{
		client:    client,
		listTTL:   listTTL,
		detailTTL: detailTTL,
		searchTTL: searchTTL,
	}
}

// GetBookDetail 获取图书详情缓存
func (c *CacheStore) GetBookDetail(ctx context.Context, bookID uint) (*book.Book, error) {
	key := c.bookDetailKey(bookID)

	// 从Redis获取JSON字符串
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中，返回nil（调用方需要查询数据库）
			return nil, nil
		}
		return nil, fmt.Errorf("获取缓存失败: %w", err)
	}

	// 反序列化JSON
	var b book.Book
	if err := json.Unmarshal([]byte(val), &b); err != nil {
		return nil, fmt.Errorf("反序列化失败: %w", err)
	}

	return &b, nil
}

// SetBookDetail 设置图书详情缓存
func (c *CacheStore) SetBookDetail(ctx context.Context, b *book.Book) error {
	key := c.bookDetailKey(b.ID)

	// 序列化为JSON
	val, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	// 写入Redis，设置过期时间
	if err := c.client.Set(ctx, key, val, c.detailTTL).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// DeleteBookDetail 删除图书详情缓存
//
// 教学要点：
// 1. 何时删除缓存？
//   - 更新图书信息时
//   - 删除图书时
//
// 2. 为什么不更新缓存？
//   - 更新操作可能并发执行，导致缓存数据不一致
//   - 删除缓存简单可靠，下次查询时重新加载最新数据
func (c *CacheStore) DeleteBookDetail(ctx context.Context, bookID uint) error {
	key := c.bookDetailKey(bookID)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	return nil
}

// GetBookList 获取图书列表缓存
func (c *CacheStore) GetBookList(ctx context.Context, page, pageSize int, sortBy, order string) ([]*book.Book, int64, error) {
	key := c.bookListKey(page, pageSize, sortBy, order)

	// 从Redis获取JSON字符串
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, nil // 缓存未命中
		}
		return nil, 0, fmt.Errorf("获取缓存失败: %w", err)
	}

	// 反序列化
	var result struct {
		Books []*book.Book `json:"books"`
		Total int64        `json:"total"`
	}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, 0, fmt.Errorf("反序列化失败: %w", err)
	}

	return result.Books, result.Total, nil
}

// SetBookList 设置图书列表缓存
func (c *CacheStore) SetBookList(ctx context.Context, books []*book.Book, total int64, page, pageSize int, sortBy, order string) error {
	key := c.bookListKey(page, pageSize, sortBy, order)

	// 序列化
	result := struct {
		Books []*book.Book `json:"books"`
		Total int64        `json:"total"`
	}{
		Books: books,
		Total: total,
	}

	val, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	// 写入Redis
	if err := c.client.Set(ctx, key, val, c.listTTL).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// DeleteBookListCache 删除所有列表缓存
//
// 教学要点：
// 1. 发布新图书时需要删除所有列表缓存
// 2. 使用SCAN命令遍历所有匹配的key
// 3. 批量删除使用UNLINK（异步删除，不阻塞）
func (c *CacheStore) DeleteBookListCache(ctx context.Context) error {
	// 使用SCAN命令查找所有列表缓存key
	pattern := "catalog:list:*"
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描缓存key失败: %w", err)
	}

	// 批量删除
	if len(keys) > 0 {
		if err := c.client.Unlink(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("删除缓存失败: %w", err)
		}
	}

	return nil
}

// GetSearchResult 获取搜索结果缓存
func (c *CacheStore) GetSearchResult(ctx context.Context, keyword string, page, pageSize int) ([]*book.Book, int64, error) {
	key := c.searchResultKey(keyword, page, pageSize)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("获取缓存失败: %w", err)
	}

	var result struct {
		Books []*book.Book `json:"books"`
		Total int64        `json:"total"`
	}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, 0, fmt.Errorf("反序列化失败: %w", err)
	}

	return result.Books, result.Total, nil
}

// SetSearchResult 设置搜索结果缓存
func (c *CacheStore) SetSearchResult(ctx context.Context, books []*book.Book, total int64, keyword string, page, pageSize int) error {
	key := c.searchResultKey(keyword, page, pageSize)

	result := struct {
		Books []*book.Book `json:"books"`
		Total int64        `json:"total"`
	}{
		Books: books,
		Total: total,
	}

	val, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	if err := c.client.Set(ctx, key, val, c.searchTTL).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// bookDetailKey 生成图书详情缓存key
// 格式：catalog:detail:{book_id}
func (c *CacheStore) bookDetailKey(bookID uint) string {
	return fmt.Sprintf("catalog:detail:%d", bookID)
}

// bookListKey 生成图书列表缓存key
// 格式：catalog:list:{page}:{pageSize}:{sortBy}:{order}
//
// 教学要点：
// 1. Key设计原则
//   - 包含所有查询参数（避免脏数据）
//   - 使用冒号分隔（Redis规范）
//   - 有业务前缀（catalog:）便于管理
func (c *CacheStore) bookListKey(page, pageSize int, sortBy, order string) string {
	return fmt.Sprintf("catalog:list:%d:%d:%s:%s", page, pageSize, sortBy, order)
}

// searchResultKey 生成搜索结果缓存key
// 格式：catalog:search:{keyword}:{page}:{pageSize}
func (c *CacheStore) searchResultKey(keyword string, page, pageSize int) string {
	return fmt.Sprintf("catalog:search:%s:%d:%d", keyword, page, pageSize)
}
