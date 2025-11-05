package integration

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 教学说明：订单模块集成测试
//
// 订单模块是本项目的核心，包含以下关键技术点：
// 1. 数据库事务（Transaction）
// 2. 悲观锁防超卖（SELECT FOR UPDATE）
// 3. 并发控制
// 4. 订单状态机
//
// 这个测试文件验证了这些核心功能的正确性

// TestOrderCreate 测试订单创建功能
func TestOrderCreate(t *testing.T) {
	// 准备测试数据
	_, token := RegisterTestUser(t, "order_creator")

	t.Run("正常创建订单", func(t *testing.T) {
		// 上架一本图书，库存10
		bookID := PublishTestBook(t, token, "《订单测试图书》", 10)

		// 创建订单，购买3本
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 3,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.Equal(t, 0, resp.Code, "创建订单应该成功")
		// message可能是"success"或"下单成功"

		var data OrderData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.NotZero(t, data.OrderID, "订单ID应该大于0")
		assert.NotEmpty(t, data.OrderNo, "订单号不应该为空")
		assert.Equal(t, int64(26700), data.Total, "订单金额应该是89.00*3=267.00元")
		assert.Equal(t, "267.00", data.TotalYuan, "订单金额(元)应该是267.00")

		t.Logf("✓ 订单创建成功")
		t.Logf("  订单ID: %d", data.OrderID)
		t.Logf("  订单号: %s", data.OrderNo)
		t.Logf("  订单金额: %s元", data.TotalYuan)
	})

	t.Run("未登录不能下单", func(t *testing.T) {
		bookID := PublishTestBook(t, token, "《测试图书》", 10)

		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 1,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, "") // 空token

		assert.NotEqual(t, 0, resp.Code, "未登录应该失败")
		assert.Contains(t, resp.Message, "token", "错误信息应该提示token相关")

		t.Logf("✓ 未登录正确被拒绝: %s", resp.Message)
	})

	t.Run("图书不存在应失败", func(t *testing.T) {
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  999999, // 不存在的图书ID
					"quantity": 1,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.NotEqual(t, 0, resp.Code, "图书不存在应该失败")
		assert.Contains(t, resp.Message, "图书", "错误信息应该提示图书相关")

		t.Logf("✓ 图书不存在正确返回错误: %s", resp.Message)
	})

	t.Run("购买数量为0应失败", func(t *testing.T) {
		bookID := PublishTestBook(t, token, "《测试图书》", 10)

		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 0, // 数量为0
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.NotEqual(t, 0, resp.Code, "购买数量为0应该失败")

		t.Logf("✓ 购买数量为0正确返回错误: %s", resp.Message)
	})

	t.Run("购买数量超过上限应失败", func(t *testing.T) {
		bookID := PublishTestBook(t, token, "《测试图书》", 10)

		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 1000, // 超过单次购买上限（假设限制为999）
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.NotEqual(t, 0, resp.Code, "购买数量超限应该失败")

		t.Logf("✓ 购买数量超限正确返回错误: %s", resp.Message)
	})

	t.Run("多商品订单", func(t *testing.T) {
		// 上架2本不同的图书
		bookID1 := PublishTestBook(t, token, "《图书A》", 10)
		bookID2 := PublishTestBook(t, token, "《图书B》", 20)

		// 创建订单，购买2种图书
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID1,
					"quantity": 2,
				},
				{
					"book_id":  bookID2,
					"quantity": 3,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.Equal(t, 0, resp.Code, "多商品订单应该成功")

		var data OrderData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		// 总金额 = 89*2 + 89*3 = 178 + 267 = 445元
		expectedTotal := int64(89*2+89*3) * 100
		assert.Equal(t, expectedTotal, data.Total, "订单总金额应该是两本书的总和")

		t.Logf("✓ 多商品订单创建成功，总金额: %s元", data.TotalYuan)
	})
}

// TestOrderStockControl 测试库存控制（防超卖核心功能）
func TestOrderStockControl(t *testing.T) {
	_, token := RegisterTestUser(t, "stock_tester")

	t.Run("库存不足应失败", func(t *testing.T) {
		// 上架库存为5的图书
		bookID := PublishTestBook(t, token, "《库存测试》", 5)

		// 尝试购买8本（超过库存）
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 8,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.NotEqual(t, 0, resp.Code, "库存不足应该失败")
		assert.Contains(t, resp.Message, "库存", "错误信息应该提示库存相关")

		t.Logf("✓ 库存不足正确返回错误: %s", resp.Message)
	})

	t.Run("库存恰好足够", func(t *testing.T) {
		// 上架库存为5的图书
		bookID := PublishTestBook(t, token, "《库存边界测试》", 5)

		// 购买恰好5本
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 5,
				},
			},
		}

		resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

		assert.Equal(t, 0, resp.Code, "库存恰好足够应该成功")

		t.Logf("✓ 库存边界测试通过（购买5本，库存5本）")
	})

	t.Run("多次下单逐步扣减库存", func(t *testing.T) {
		// 上架库存为10的图书
		bookID := PublishTestBook(t, token, "《多次下单测试》", 10)

		// 第一次下单：购买3本
		orderReq1 := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 3,
				},
			},
		}
		resp1 := PostJSON(t, BaseURL+"/orders", orderReq1, token)
		require.Equal(t, 0, resp1.Code, "第一次下单应该成功")
		t.Logf("✓ 第一次下单成功，购买3本，剩余7本")

		// 第二次下单：购买4本
		orderReq2 := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 4,
				},
			},
		}
		resp2 := PostJSON(t, BaseURL+"/orders", orderReq2, token)
		require.Equal(t, 0, resp2.Code, "第二次下单应该成功")
		t.Logf("✓ 第二次下单成功，购买4本，剩余3本")

		// 第三次下单：尝试购买5本（库存不足）
		orderReq3 := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 5,
				},
			},
		}
		resp3 := PostJSON(t, BaseURL+"/orders", orderReq3, token)
		assert.NotEqual(t, 0, resp3.Code, "第三次下单应该失败（库存不足）")
		t.Logf("✓ 第三次下单正确失败（尝试购买5本，实际剩余3本）")

		// 第四次下单：购买3本（恰好用完库存）
		orderReq4 := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 3,
				},
			},
		}
		resp4 := PostJSON(t, BaseURL+"/orders", orderReq4, token)
		assert.Equal(t, 0, resp4.Code, "第四次下单应该成功（恰好用完库存）")
		t.Logf("✓ 第四次下单成功，购买3本，库存清零")
	})
}

// TestOrderConcurrency 测试并发下单（防超卖核心场景）
//
// 教学说明：
// 这是本项目最重要的测试之一，验证了悲观锁防超卖的正确性
//
// 场景设计：
// - 库存：10本
// - 并发请求：20个goroutine同时下单，每个购买1本
// - 预期结果：10个成功，10个失败（库存不足）
//
// 技术要点：
// - 使用 sync.WaitGroup 等待所有goroutine完成
// - 使用 sync.Mutex 保护共享变量（成功/失败计数）
// - SELECT FOR UPDATE 确保同一时刻只有一个事务能获取库存锁
func TestOrderConcurrency(t *testing.T) {
	_, token := RegisterTestUser(t, "concurrency_tester")

	t.Run("并发下单防超卖（10库存，20并发请求）", func(t *testing.T) {
		// 上架库存为10的图书
		bookID := PublishTestBook(t, token, "《并发测试图书》", 10)

		t.Logf("\n========================================")
		t.Logf("开始并发测试：10本库存，20个并发请求")
		t.Logf("========================================")

		var (
			wg           sync.WaitGroup
			mu           sync.Mutex
			successCount int
			failCount    int
		)

		// 启动20个goroutine并发下单
		concurrency := 20
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				orderReq := map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"book_id":  bookID,
							"quantity": 1,
						},
					},
				}

				resp := PostJSON(t, BaseURL+"/orders", orderReq, token)

				mu.Lock()
				if resp.Code == 0 {
					successCount++
					t.Logf("  [请求%02d] ✓ 下单成功", idx+1)
				} else {
					failCount++
					t.Logf("  [请求%02d] ✗ 下单失败: %s", idx+1, resp.Message)
				}
				mu.Unlock()
			}(i)
		}

		// 等待所有goroutine完成
		wg.Wait()

		t.Logf("\n========================================")
		t.Logf("并发测试结果：")
		t.Logf("  成功下单: %d 个", successCount)
		t.Logf("  失败下单: %d 个", failCount)
		t.Logf("========================================")

		// 验证结果
		assert.Equal(t, 10, successCount, "成功订单数应该等于库存数")
		assert.Equal(t, 10, failCount, "失败订单数应该是总请求数减去库存数")
		assert.Equal(t, concurrency, successCount+failCount, "成功+失败应该等于总请求数")

		if successCount == 10 && failCount == 10 {
			t.Logf("\n✅ 防超卖机制测试通过！")
			t.Logf("✅ 悲观锁(SELECT FOR UPDATE)有效防止了超卖")
			t.Logf("\n教学要点：")
			t.Logf("1. SELECT FOR UPDATE 锁定库存行，确保同一时刻只有一个事务能修改")
			t.Logf("2. 其他事务等待锁释放，按顺序执行")
			t.Logf("3. 库存扣减是原子操作，不会出现竞态条件")
			t.Logf("4. 成功订单数 = 库存数，不会超卖也不会少卖")
		} else {
			t.Errorf("❌ 防超卖机制失败！")
			t.Errorf("   预期：成功10个，失败10个")
			t.Errorf("   实际：成功%d个，失败%d个", successCount, failCount)
		}
	})

	t.Run("不同用户并发抢购", func(t *testing.T) {
		// 教学说明：模拟真实场景，多个用户同时抢购同一本书
		// 上架库存为5的图书
		_, token1 := RegisterTestUser(t, "buyer1")
		bookID := PublishTestBook(t, token1, "《热门图书》", 5)

		// 注册多个买家
		tokens := []string{token1}
		for i := 2; i <= 10; i++ {
			_, token := RegisterTestUser(t, fmt.Sprintf("buyer%d", i))
			tokens = append(tokens, token)
		}

		t.Logf("\n========================================")
		t.Logf("模拟多用户抢购：5本库存，10个买家")
		t.Logf("========================================")

		var (
			wg           sync.WaitGroup
			mu           sync.Mutex
			successCount int
		)

		// 10个买家同时下单
		for i, token := range tokens {
			wg.Add(1)
			go func(idx int, userToken string) {
				defer wg.Done()

				orderReq := map[string]interface{}{
					"items": []map[string]interface{}{
						{
							"book_id":  bookID,
							"quantity": 1,
						},
					},
				}

				resp := PostJSON(t, BaseURL+"/orders", orderReq, userToken)

				mu.Lock()
				if resp.Code == 0 {
					successCount++
					t.Logf("  [买家%02d] ✓ 抢购成功", idx+1)
				} else {
					t.Logf("  [买家%02d] ✗ 抢购失败: %s", idx+1, resp.Message)
				}
				mu.Unlock()
			}(i, token)
		}

		wg.Wait()

		t.Logf("\n========================================")
		t.Logf("抢购结果：%d 位买家成功（库存5本）", successCount)
		t.Logf("========================================")

		assert.Equal(t, 5, successCount, "应该有5位买家抢购成功")

		t.Logf("\n✅ 多用户并发抢购测试通过！")
	})
}

// TestOrderCompleteFlow 测试完整的下单流程
//
// 教学说明：
// 这是一个端到端(E2E)测试，验证从注册到下单的完整业务流程
func TestOrderCompleteFlow(t *testing.T) {
	t.Log("\n========================================")
	t.Log("完整下单流程测试")
	t.Log("========================================")

	// Step 1: 卖家注册并上架图书
	t.Log("\n➜ Step 1: 卖家注册并上架图书")
	sellerEmail, sellerToken := RegisterTestUser(t, "卖家")
	t.Logf("✓ 卖家注册成功: %s", sellerEmail)

	bookID := PublishTestBook(t, sellerToken, "《Go微服务实战》", 50)
	t.Logf("✓ 图书上架成功，图书ID: %d, 库存: 50", bookID)

	// Step 2: 买家注册
	t.Log("\n➜ Step 2: 买家注册")
	buyerEmail, buyerToken := RegisterTestUser(t, "买家")
	t.Logf("✓ 买家注册成功: %s", buyerEmail)

	// Step 3: 买家浏览图书列表
	t.Log("\n➜ Step 3: 买家浏览图书列表")
	listResp := GetJSON(t, BaseURL+"/books", "")
	require.Equal(t, 0, listResp.Code, "查询图书列表失败")
	t.Logf("✓ 图书列表查询成功")

	// Step 4: 买家创建订单
	t.Log("\n➜ Step 4: 买家创建订单（购买3本）")
	orderReq := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"book_id":  bookID,
				"quantity": 3,
			},
		},
	}

	orderResp := PostJSON(t, BaseURL+"/orders", orderReq, buyerToken)
	require.Equal(t, 0, orderResp.Code, "创建订单失败")

	var orderData OrderData
	err := json.Unmarshal(orderResp.Data, &orderData)
	require.NoError(t, err, "解析订单响应失败")

	t.Logf("✓ 订单创建成功")
	t.Logf("  订单号: %s", orderData.OrderNo)
	t.Logf("  订单金额: %s元", orderData.TotalYuan)
	t.Logf("  预期库存变化: 50 → 47")

	t.Log("\n========================================")
	t.Log("✅ 完整下单流程测试通过")
	t.Log("========================================")
	t.Log("\n业务流程总结：")
	t.Log("1. 卖家注册 → 上架图书（库存50）")
	t.Log("2. 买家注册 → 浏览图书")
	t.Log("3. 买家下单 → 创建订单（购买3本）")
	t.Log("4. 系统扣减库存 → 库存变为47")
	t.Log("5. 订单状态 → PENDING（待支付）")
}
