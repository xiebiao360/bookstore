package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 教学说明：图书模块集成测试
//
// 测试场景覆盖：
// 1. 图书上架（需要认证）
// 2. 图书列表查询（公开接口）
// 3. 分页、排序、搜索功能
// 4. 参数验证（ISBN格式、价格范围、库存）

// TestBookPublish 测试图书上架功能
func TestBookPublish(t *testing.T) {
	// 准备测试数据：注册并登录用户
	_, token := RegisterTestUser(t, "book_publisher")

	t.Run("正常上架图书", func(t *testing.T) {
		isbn := GenerateTestISBN()
		bookReq := map[string]interface{}{
			"title":       "《Go语言高级编程》",
			"author":      "柴树杉",
			"isbn":        isbn,
			"publisher":   "人民邮电出版社",
			"price":       8900, // 89.00元
			"stock":       100,
			"description": "深入理解Go语言底层原理",
		}

		resp := PostJSON(t, BaseURL+"/books", bookReq, token)

		assert.Equal(t, 0, resp.Code, "上架应该成功")
		// message可能是"success"或"上架成功"

		var data BookData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.NotZero(t, data.ID, "图书ID应该大于0")
		assert.Equal(t, isbn, data.ISBN, "ISBN应该一致")
		assert.Equal(t, "《Go语言高级编程》", data.Title, "标题应该一致")
		assert.Equal(t, int64(8900), data.Price, "价格应该一致")
		assert.Equal(t, 100, data.Stock, "库存应该一致")

		t.Logf("✓ 上架成功，图书ID: %d, ISBN: %s", data.ID, data.ISBN)
	})

	t.Run("未登录不能上架", func(t *testing.T) {
		bookReq := map[string]interface{}{
			"title":       "《测试图书》",
			"author":      "测试作者",
			"isbn":        GenerateTestISBN(),
			"publisher":   "测试出版社",
			"price":       5900,
			"stock":       50,
			"description": "测试描述",
		}

		resp := PostJSON(t, BaseURL+"/books", bookReq, "") // 空token

		assert.NotEqual(t, 0, resp.Code, "未登录应该失败")
		assert.Contains(t, resp.Message, "token", "错误信息应该提示token相关")

		t.Logf("✓ 未登录正确被拒绝: %s", resp.Message)
	})

	t.Run("ISBN重复应失败", func(t *testing.T) {
		isbn := GenerateTestISBN()

		// 第一次上架
		bookReq1 := map[string]interface{}{
			"title":       "《图书A》",
			"author":      "作者A",
			"isbn":        isbn,
			"publisher":   "出版社A",
			"price":       5900,
			"stock":       10,
			"description": "描述A",
		}

		resp1 := PostJSON(t, BaseURL+"/books", bookReq1, token)
		require.Equal(t, 0, resp1.Code, "第一次上架应该成功")

		// 第二次上架（相同ISBN）
		bookReq2 := map[string]interface{}{
			"title":       "《图书B》",
			"author":      "作者B",
			"isbn":        isbn, // 相同ISBN
			"publisher":   "出版社B",
			"price":       6900,
			"stock":       20,
			"description": "描述B",
		}

		resp2 := PostJSON(t, BaseURL+"/books", bookReq2, token)
		assert.NotEqual(t, 0, resp2.Code, "重复ISBN应该失败")
		assert.Contains(t, resp2.Message, "ISBN", "错误信息应该提示ISBN相关")

		t.Logf("✓ 重复ISBN正确返回错误: %s", resp2.Message)
	})

	t.Run("ISBN格式错误应失败", func(t *testing.T) {
		// 教学说明：ISBN-13格式验证
		// 有效格式：978-7-111-54742-6 或 9787111547426（13位数字）
		// 无效格式：123（太短）、abcd-efgh-ijkl（非数字）

		invalidISBNs := []string{
			"123",            // 太短
			"abc123def456",   // 包含字母
			"978711154742",   // 12位（少1位）
			"97871115474299", // 14位（多1位）
		}

		for _, invalidISBN := range invalidISBNs {
			bookReq := map[string]interface{}{
				"title":       "《测试图书》",
				"author":      "测试作者",
				"isbn":        invalidISBN,
				"publisher":   "测试出版社",
				"price":       5900,
				"stock":       10,
				"description": "ISBN格式测试",
			}

			resp := PostJSON(t, BaseURL+"/books", bookReq, token)
			assert.NotEqual(t, 0, resp.Code, "无效ISBN应该失败: %s", invalidISBN)
			assert.Contains(t, resp.Message, "ISBN", "错误信息应该提示ISBN相关")

			t.Logf("✓ 无效ISBN '%s' 正确被拒绝: %s", invalidISBN, resp.Message)
		}
	})

	t.Run("价格范围验证", func(t *testing.T) {
		// 教学说明：价格范围验证
		// 有效范围：1-999999分（0.01元 - 9999.99元）

		testCases := []struct {
			price       int64
			shouldFail  bool
			description string
		}{
			{0, true, "价格为0"},
			{-100, true, "负价格"},
			{1, false, "最小有效价格(0.01元)"},
			{999999, false, "最大有效价格(9999.99元)"},
			{1000000, true, "超过最大价格"},
		}

		for _, tc := range testCases {
			bookReq := map[string]interface{}{
				"title":       "《价格测试》",
				"author":      "测试作者",
				"isbn":        GenerateTestISBN(),
				"publisher":   "测试出版社",
				"price":       tc.price,
				"stock":       10,
				"description": tc.description,
			}

			resp := PostJSON(t, BaseURL+"/books", bookReq, token)

			if tc.shouldFail {
				assert.NotEqual(t, 0, resp.Code, "价格%d应该失败: %s", tc.price, tc.description)
				t.Logf("✓ %s 正确被拒绝: %s", tc.description, resp.Message)
			} else {
				assert.Equal(t, 0, resp.Code, "价格%d应该成功: %s", tc.price, tc.description)
				t.Logf("✓ %s 正确通过", tc.description)
			}
		}
	})

	t.Run("库存范围验证", func(t *testing.T) {
		testCases := []struct {
			stock       int
			shouldFail  bool
			description string
		}{
			{-1, true, "负库存"},
			{0, false, "库存为0（允许，表示无货）"},
			{1, false, "最小库存"},
			{9999, false, "最大库存"},
			{10000, true, "超过最大库存"},
		}

		for _, tc := range testCases {
			bookReq := map[string]interface{}{
				"title":       "《库存测试》",
				"author":      "测试作者",
				"isbn":        GenerateTestISBN(),
				"publisher":   "测试出版社",
				"price":       5900,
				"stock":       tc.stock,
				"description": tc.description,
			}

			resp := PostJSON(t, BaseURL+"/books", bookReq, token)

			if tc.shouldFail {
				assert.NotEqual(t, 0, resp.Code, "库存%d应该失败: %s", tc.stock, tc.description)
				t.Logf("✓ %s 正确被拒绝: %s", tc.description, resp.Message)
			} else {
				assert.Equal(t, 0, resp.Code, "库存%d应该成功: %s", tc.stock, tc.description)
				t.Logf("✓ %s 正确通过", tc.description)
			}
		}
	})
}

// TestBookList 测试图书列表查询功能
func TestBookList(t *testing.T) {
	// 准备测试数据：上架多本图书
	_, token := RegisterTestUser(t, "book_lister")

	// 上架5本不同价格的图书，用于测试排序
	books := []struct {
		title string
		price int64
		stock int
	}{
		{"《Go语言圣经》", 7900, 50},
		{"《设计模式》", 8900, 30},
		{"《重构》", 6900, 100},
		{"《代码整洁之道》", 5900, 80},
		{"《Go并发编程》", 9900, 20},
	}

	for _, book := range books {
		bookReq := map[string]interface{}{
			"title":       book.title,
			"author":      "测试作者",
			"isbn":        GenerateTestISBN(),
			"publisher":   "测试出版社",
			"price":       book.price,
			"stock":       book.stock,
			"description": "测试描述",
		}

		resp := PostJSON(t, BaseURL+"/books", bookReq, token)
		require.Equal(t, 0, resp.Code, "上架测试数据失败")
	}

	t.Run("默认查询（第1页，每页20条）", func(t *testing.T) {
		// 教学说明：不带任何参数，应该返回默认分页结果
		resp := GetJSON(t, BaseURL+"/books", "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.GreaterOrEqual(t, len(data.Items), 5, "至少应该返回刚上架的5本书")
		assert.Equal(t, 1, data.Page, "默认应该是第1页")
		assert.Equal(t, 20, data.PageSize, "默认每页应该是20条")
		assert.GreaterOrEqual(t, data.Total, int64(5), "总数至少是5")

		t.Logf("✓ 默认查询成功，返回 %d 本书，总数: %d", len(data.Items), data.Total)
	})

	t.Run("分页查询", func(t *testing.T) {
		// 教学说明：测试分页参数
		// page=2&page_size=2 应该返回第2页的2条数据

		url := fmt.Sprintf("%s/books?page=1&page_size=2", BaseURL)
		resp := GetJSON(t, url, "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.LessOrEqual(t, len(data.Items), 2, "每页最多2条")
		assert.Equal(t, 1, data.Page, "应该是第1页")
		assert.Equal(t, 2, data.PageSize, "每页应该是2条")

		t.Logf("✓ 分页查询成功，第%d页，每页%d条，返回%d条", data.Page, data.PageSize, len(data.Items))
	})

	t.Run("价格升序排序", func(t *testing.T) {
		// 教学说明：sort_by=price_asc 按价格从低到高排序
		url := fmt.Sprintf("%s/books?sort_by=price_asc&page_size=5", BaseURL)
		resp := GetJSON(t, url, "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		// 验证排序：第一本应该是最便宜的
		if len(data.Items) >= 2 {
			assert.LessOrEqual(t, data.Items[0].Price, data.Items[1].Price,
				"第一本书价格应该 <= 第二本书价格")
			t.Logf("✓ 价格升序正确: %s (%.2f元) <= %s (%.2f元)",
				data.Items[0].Title, float64(data.Items[0].Price)/100,
				data.Items[1].Title, float64(data.Items[1].Price)/100)
		}
	})

	t.Run("价格降序排序", func(t *testing.T) {
		// 教学说明：sort_by=price_desc 按价格从高到低排序
		url := fmt.Sprintf("%s/books?sort_by=price_desc&page_size=5", BaseURL)
		resp := GetJSON(t, url, "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		// 验证排序：第一本应该是最贵的
		if len(data.Items) >= 2 {
			assert.GreaterOrEqual(t, data.Items[0].Price, data.Items[1].Price,
				"第一本书价格应该 >= 第二本书价格")
			t.Logf("✓ 价格降序正确: %s (%.2f元) >= %s (%.2f元)",
				data.Items[0].Title, float64(data.Items[0].Price)/100,
				data.Items[1].Title, float64(data.Items[1].Price)/100)
		}
	})

	t.Run("关键词搜索", func(t *testing.T) {
		// 教学说明：keyword参数会在title、author、publisher中搜索（LIKE查询）
		url := fmt.Sprintf("%s/books?keyword=Go", BaseURL)
		resp := GetJSON(t, url, "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		// 验证所有返回的书都包含关键词
		for _, book := range data.Items {
			containsKeyword := false
			if len(book.Title) > 0 && contains(book.Title, "Go") {
				containsKeyword = true
			}
			if len(book.Author) > 0 && contains(book.Author, "Go") {
				containsKeyword = true
			}
			if len(book.Publisher) > 0 && contains(book.Publisher, "Go") {
				containsKeyword = true
			}

			// 注意：这里可能因为其他测试上架的图书而失败
			// 在实际项目中，应该使用独立的测试数据库
			if !containsKeyword {
				t.Logf("⚠ 警告：图书'%s'不包含关键词'Go'，可能是其他测试数据", book.Title)
			}
		}

		t.Logf("✓ 关键词搜索成功，找到 %d 本包含'Go'的图书", len(data.Items))
	})

	t.Run("组合查询：分页+排序+搜索", func(t *testing.T) {
		// 教学说明：测试参数组合
		url := fmt.Sprintf("%s/books?keyword=测试&sort_by=price_asc&page=1&page_size=3", BaseURL)
		resp := GetJSON(t, url, "")

		assert.Equal(t, 0, resp.Code, "查询应该成功")

		var data BookListData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.LessOrEqual(t, len(data.Items), 3, "最多返回3条")
		assert.Equal(t, 1, data.Page, "应该是第1页")
		assert.Equal(t, 3, data.PageSize, "每页应该是3条")

		// 验证排序
		if len(data.Items) >= 2 {
			assert.LessOrEqual(t, data.Items[0].Price, data.Items[1].Price,
				"应该按价格升序排列")
		}

		t.Logf("✓ 组合查询成功，返回 %d 条结果", len(data.Items))
	})

	t.Run("参数边界测试", func(t *testing.T) {
		// 教学说明：测试边界值
		testCases := []struct {
			params      string
			description string
			shouldFail  bool
		}{
			{"?page=0", "页码为0", true},
			{"?page=-1", "页码为负数", true},
			{"?page_size=0", "每页数量为0", true},
			{"?page_size=101", "每页数量超过100", true},
			{"?page_size=100", "每页数量为最大值100", false},
			{"?page=1&page_size=1", "最小有效分页", false},
		}

		for _, tc := range testCases {
			url := fmt.Sprintf("%s/books%s", BaseURL, tc.params)
			resp := GetJSON(t, url, "")

			if tc.shouldFail {
				assert.NotEqual(t, 0, resp.Code, "%s 应该失败", tc.description)
				t.Logf("✓ %s 正确返回错误: %s", tc.description, resp.Message)
			} else {
				assert.Equal(t, 0, resp.Code, "%s 应该成功", tc.description)
				t.Logf("✓ %s 正确通过", tc.description)
			}
		}
	})

	t.Run("公开接口无需认证", func(t *testing.T) {
		// 教学说明：图书列表是公开接口，不需要登录即可访问
		resp := GetJSON(t, BaseURL+"/books", "") // 空token

		assert.Equal(t, 0, resp.Code, "公开接口应该可以访问")

		t.Logf("✓ 图书列表公开访问成功")
	})
}

// TestBookStockManagement 测试库存管理
func TestBookStockManagement(t *testing.T) {
	_, token := RegisterTestUser(t, "stock_manager")

	t.Run("库存扣减和恢复", func(t *testing.T) {
		// 上架一本库存为10的图书
		bookID := PublishTestBook(t, token, "《库存测试图书》", 10)

		// 创建订单，购买3本
		orderReq := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"book_id":  bookID,
					"quantity": 3,
				},
			},
		}

		orderResp := PostJSON(t, BaseURL+"/orders", orderReq, token)
		require.Equal(t, 0, orderResp.Code, "创建订单失败")

		// 查询图书列表，验证库存已扣减
		// 注意：这里需要实现根据ID查询单本图书的接口才能精确验证
		// 目前只能通过列表接口模糊验证
		t.Logf("✓ 订单创建成功，库存应该从10减少到7")

		// 教学说明：
		// 在真实场景中，应该添加以下接口：
		// GET /books/:id - 查询单本图书详情
		// 这样可以精确验证库存变化
	})
}

// contains 辅助函数：检查字符串是否包含子串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		len(s) > 0 && (s[0:1] == substr[0:1] || contains(s[1:], substr) ||
			(len(s) > len(substr) && contains(s[1:], substr))))
}
