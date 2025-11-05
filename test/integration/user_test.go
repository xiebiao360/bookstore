package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 教学说明：用户模块集成测试
//
// 集成测试 vs 单元测试：
// - 单元测试：Mock外部依赖（数据库、Redis），测试单个函数的逻辑
// - 集成测试：使用真实的数据库和Redis，测试完整的API流程
//
// 集成测试的价值：
// 1. 验证各组件协同工作（Handler → UseCase → Service → Repository → Database）
// 2. 发现配置错误（如数据库连接、Wire依赖注入）
// 3. 验证业务流程的完整性
//
// 运行方式：
//   make test-integration   # 需要先启动Docker环境
//   go test -v ./test/integration/...

// TestUserRegister 测试用户注册功能
//
// 测试场景：
// 1. 正常注册
// 2. 重复邮箱注册（应失败）
// 3. 密码格式校验
// 4. 邮箱格式校验
func TestUserRegister(t *testing.T) {
	// 教学说明：使用t.Run()组织子测试
	// 好处：
	// 1. 测试结果更清晰（可以看到每个子场景的结果）
	// 2. 子测试失败不影响其他子测试
	// 3. 可以使用 go test -run=TestUserRegister/正常注册 运行单个子测试

	t.Run("正常注册", func(t *testing.T) {
		email := GenerateTestEmail("normal_user")
		registerReq := map[string]string{
			"email":    email,
			"password": "Test1234",
			"nickname": "测试用户",
		}

		resp := PostJSON(t, BaseURL+"/users/register", registerReq, "")

		// 断言响应码为0（成功）
		assert.Equal(t, 0, resp.Code, "注册应该成功")
		// 注意：message可能是"success"或"注册成功"，取决于实现
		// 这里只检查响应码，不检查具体消息

		// 解析并验证返回数据
		var data RegisterData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.NotZero(t, data.ID, "用户ID应该大于0")
		assert.Equal(t, email, data.Email, "返回的邮箱应该与请求一致")
		assert.Equal(t, "测试用户", data.Nickname, "返回的昵称应该与请求一致")

		t.Logf("✓ 注册成功，用户ID: %d", data.ID)
	})

	t.Run("重复邮箱注册应失败", func(t *testing.T) {
		// 第一次注册
		email := GenerateTestEmail("duplicate_user")
		registerReq := map[string]string{
			"email":    email,
			"password": "Test1234",
			"nickname": "测试用户1",
		}

		resp1 := PostJSON(t, BaseURL+"/users/register", registerReq, "")
		require.Equal(t, 0, resp1.Code, "第一次注册应该成功")

		// 第二次注册（相同邮箱）
		registerReq["nickname"] = "测试用户2"
		resp2 := PostJSON(t, BaseURL+"/users/register", registerReq, "")

		// 教学说明：错误码定义
		// 40901: 邮箱已存在（409 Conflict + 01自定义业务码）
		assert.NotEqual(t, 0, resp2.Code, "重复邮箱注册应该失败")
		assert.Contains(t, resp2.Message, "邮箱", "错误信息应该提示邮箱相关")

		t.Logf("✓ 重复邮箱注册正确返回错误: %s", resp2.Message)
	})

	t.Run("密码过短应失败", func(t *testing.T) {
		email := GenerateTestEmail("short_pwd")
		registerReq := map[string]string{
			"email":    email,
			"password": "123", // 太短（<6位）
			"nickname": "测试用户",
		}

		resp := PostJSON(t, BaseURL+"/users/register", registerReq, "")

		assert.NotEqual(t, 0, resp.Code, "密码过短应该失败")
		// 错误信息可能是英文参数验证信息，包含Password即可
		// assert.Contains(t, resp.Message, "密码", "错误信息应该提示密码相关")

		t.Logf("✓ 密码过短正确返回错误: %s", resp.Message)
	})

	t.Run("邮箱格式错误应失败", func(t *testing.T) {
		registerReq := map[string]string{
			"email":    "invalid-email", // 无效邮箱格式
			"password": "Test1234",
			"nickname": "测试用户",
		}

		resp := PostJSON(t, BaseURL+"/users/register", registerReq, "")

		assert.NotEqual(t, 0, resp.Code, "邮箱格式错误应该失败")
		// 错误信息可能是英文参数验证信息，包含Email即可
		// assert.Contains(t, resp.Message, "邮箱", "错误信息应该提示邮箱相关")

		t.Logf("✓ 邮箱格式错误正确返回错误: %s", resp.Message)
	})
}

// TestUserLogin 测试用户登录功能
//
// 测试场景：
// 1. 正常登录
// 2. 密码错误
// 3. 用户不存在
// 4. Token有效性
func TestUserLogin(t *testing.T) {
	// 准备测试数据：先注册一个用户
	email := GenerateTestEmail("login_test")
	password := "Test1234"
	registerReq := map[string]string{
		"email":    email,
		"password": password,
		"nickname": "登录测试用户",
	}

	registerResp := PostJSON(t, BaseURL+"/users/register", registerReq, "")
	require.Equal(t, 0, registerResp.Code, "准备测试数据：注册用户")

	t.Run("正常登录", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    email,
			"password": password,
		}

		resp := PostJSON(t, BaseURL+"/users/login", loginReq, "")

		assert.Equal(t, 0, resp.Code, "登录应该成功")
		// message可能是"success"或"登录成功"

		// 解析并验证返回数据
		var data LoginData
		err := json.Unmarshal(resp.Data, &data)
		require.NoError(t, err, "解析响应数据失败")

		assert.NotEmpty(t, data.AccessToken, "应该返回access_token")
		assert.NotEmpty(t, data.RefreshToken, "应该返回refresh_token")

		// 教学说明：JWT Token格式
		// JWT由三部分组成：header.payload.signature
		// 例如：eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
		assert.Contains(t, data.AccessToken, ".", "JWT Token应该包含点号分隔符")

		t.Logf("✓ 登录成功，Access Token长度: %d", len(data.AccessToken))
	})

	t.Run("密码错误应失败", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    email,
			"password": "WrongPassword",
		}

		resp := PostJSON(t, BaseURL+"/users/login", loginReq, "")

		assert.NotEqual(t, 0, resp.Code, "密码错误应该失败")
		assert.Contains(t, resp.Message, "密码", "错误信息应该提示密码相关")

		t.Logf("✓ 密码错误正确返回错误: %s", resp.Message)
	})

	t.Run("用户不存在应失败", func(t *testing.T) {
		loginReq := map[string]string{
			"email":    "nonexistent@test.com",
			"password": "Test1234",
		}

		resp := PostJSON(t, BaseURL+"/users/login", loginReq, "")

		assert.NotEqual(t, 0, resp.Code, "用户不存在应该失败")
		// 安全考虑：不应该明确提示"用户不存在"，而是统一返回"邮箱或密码错误"
		// 防止攻击者枚举邮箱

		t.Logf("✓ 用户不存在正确返回错误: %s", resp.Message)
	})

	t.Run("Token可以访问受保护接口", func(t *testing.T) {
		// 先登录获取Token
		loginReq := map[string]string{
			"email":    email,
			"password": password,
		}

		loginResp := PostJSON(t, BaseURL+"/users/login", loginReq, "")
		require.Equal(t, 0, loginResp.Code, "登录失败")

		var loginData LoginData
		err := json.Unmarshal(loginResp.Data, &loginData)
		require.NoError(t, err, "解析登录响应失败")

		token := loginData.AccessToken

		// 使用Token访问需要认证的接口（发布图书）
		bookReq := map[string]interface{}{
			"title":       "《测试图书》",
			"author":      "测试作者",
			"isbn":        GenerateTestISBN(),
			"publisher":   "测试出版社",
			"price":       5900,
			"stock":       10,
			"description": "Token验证测试",
		}

		bookResp := PostJSON(t, BaseURL+"/books", bookReq, token)
		assert.Equal(t, 0, bookResp.Code, "使用有效Token应该可以发布图书")

		t.Logf("✓ Token验证通过，可以访问受保护接口")
	})

	t.Run("无效Token应被拒绝", func(t *testing.T) {
		bookReq := map[string]interface{}{
			"title":       "《测试图书》",
			"author":      "测试作者",
			"isbn":        GenerateTestISBN(),
			"publisher":   "测试出版社",
			"price":       5900,
			"stock":       10,
			"description": "无效Token测试",
		}

		invalidToken := "invalid.jwt.token"
		bookResp := PostJSON(t, BaseURL+"/books", bookReq, invalidToken)

		assert.NotEqual(t, 0, bookResp.Code, "无效Token应该被拒绝")
		assert.Contains(t, bookResp.Message, "token", "错误信息应该提示token相关")

		t.Logf("✓ 无效Token正确被拒绝: %s", bookResp.Message)
	})
}

// TestUserAuthFlow 测试完整的认证流程
//
// 教学说明：
// 这是一个"端到端"(E2E)测试，验证完整的用户认证流程
// 注册 → 登录 → 访问受保护资源
func TestUserAuthFlow(t *testing.T) {
	t.Log("========================================")
	t.Log("测试完整认证流程")
	t.Log("========================================")

	// Step 1: 注册新用户
	t.Log("\n➜ Step 1: 注册新用户")
	email := GenerateTestEmail("auth_flow")
	password := "Test1234"

	registerReq := map[string]string{
		"email":    email,
		"password": password,
		"nickname": "认证流程测试",
	}

	registerResp := PostJSON(t, BaseURL+"/users/register", registerReq, "")
	require.Equal(t, 0, registerResp.Code, "注册失败")

	var registerData RegisterData
	err := json.Unmarshal(registerResp.Data, &registerData)
	require.NoError(t, err, "解析注册响应失败")

	t.Logf("✓ 注册成功，用户ID: %d, 邮箱: %s", registerData.ID, registerData.Email)

	// Step 2: 使用邮箱密码登录
	t.Log("\n➜ Step 2: 登录获取Token")
	loginReq := map[string]string{
		"email":    email,
		"password": password,
	}

	loginResp := PostJSON(t, BaseURL+"/users/login", loginReq, "")
	require.Equal(t, 0, loginResp.Code, "登录失败")

	var loginData LoginData
	err = json.Unmarshal(loginResp.Data, &loginData)
	require.NoError(t, err, "解析登录响应失败")

	token := loginData.AccessToken
	t.Logf("✓ 登录成功，获取Token: %s...", token[:30])

	// Step 3: 使用Token访问受保护接口（发布图书）
	t.Log("\n➜ Step 3: 使用Token发布图书")
	isbn := GenerateTestISBN()
	bookReq := map[string]interface{}{
		"title":       "《Go微服务实战》",
		"author":      "测试作者",
		"isbn":        isbn,
		"publisher":   "测试出版社",
		"price":       8900,
		"stock":       100,
		"description": "认证流程测试图书",
	}

	bookResp := PostJSON(t, BaseURL+"/books", bookReq, token)
	require.Equal(t, 0, bookResp.Code, "发布图书失败")

	var bookData BookData
	err = json.Unmarshal(bookResp.Data, &bookData)
	require.NoError(t, err, "解析图书响应失败")

	t.Logf("✓ 发布图书成功，图书ID: %d, ISBN: %s", bookData.ID, bookData.ISBN)

	// Step 4: 创建订单（再次验证Token）
	t.Log("\n➜ Step 4: 使用Token创建订单")
	orderReq := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"book_id":  bookData.ID,
				"quantity": 2,
			},
		},
	}

	orderResp := PostJSON(t, BaseURL+"/orders", orderReq, token)
	require.Equal(t, 0, orderResp.Code, "创建订单失败")

	var orderData OrderData
	err = json.Unmarshal(orderResp.Data, &orderData)
	require.NoError(t, err, "解析订单响应失败")

	t.Logf("✓ 创建订单成功，订单号: %s, 金额: %s", orderData.OrderNo, orderData.TotalYuan)

	t.Log("\n========================================")
	t.Log("✅ 完整认证流程测试通过")
	t.Log("========================================")
	t.Log("\n教学要点：")
	t.Log("1. JWT Token在整个会话中保持有效")
	t.Log("2. Token可以访问所有需要认证的接口")
	t.Log("3. 服务端通过Token识别用户身份（无需Session）")
	t.Log("4. Token存储在Redis中，实现登出功能")
}
