# Week 3 Day 17: Swagger API文档完成报告

## 📋 任务概述

本阶段为项目集成了**Swagger API文档**，提供交互式的API测试界面，大幅提升了开发效率和API可维护性。

## ✅ 完成内容

### 1. Swag工具安装

```bash
# 安装Swag CLI
go install github.com/swaggo/swag/cmd/swag@latest

# 添加依赖
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

**验证安装**:
```bash
$ swag --version
swag version v1.16.4
```

---

### 2. Swagger基础配置 (`cmd/api/main.go`)

在main.go顶部添加了API的全局元信息：

```go
// @title           图书商城API文档
// @version         1.0
// @description     这是一个教学导向的Go微服务实战项目的API文档
// @description     本项目演示了DDD分层架构、Wire依赖注入、防超卖等核心技术
//
// @contact.name    项目维护者
// @contact.url     https://github.com/xiebiao/bookstore
// @contact.email   xiebiao@example.com
//
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /api/v1
//
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 输入"Bearer {token}"进行身份验证
```

**教学价值**:
- `@title`: API文档的标题
- `@host` + `@BasePath`: 定义API的完整路径（http://localhost:8080/api/v1）
- `@securityDefinitions`: 定义JWT认证方式
- 这些注释会被Swag解析生成OpenAPI规范的文档

---

### 3. API接口Swagger注释

为所有Handler方法添加了详细的Swagger注释。

#### 3.1 用户注册接口

```go
// Register 用户注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户模块
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "注册信息"
// @Success      200 {object} response.Response{data=dto.UserResponse} "注册成功"
// @Failure      400 {object} response.Response "参数错误"
// @Failure      409 {object} response.Response "邮箱已存在"
// @Router       /users/register [post]
```

**注释详解**:
- `@Summary`: 接口简短描述（显示在列表中）
- `@Description`: 接口详细描述
- `@Tags`: 接口分组（Swagger UI中按Tag分类）
- `@Accept` / `@Produce`: 请求/响应的Content-Type
- `@Param`: 参数定义
  - 格式: `name in type required comment`
  - `request body dto.RegisterRequest true "注册信息"`
    - name: request（参数名）
    - in: body（请求体）
    - type: dto.RegisterRequest（数据类型）
    - required: true（必填）
- `@Success` / `@Failure`: 响应定义
  - 格式: `httpCode {dataType} comment`
  - `{object} response.Response{data=dto.UserResponse}`表示响应体是Response，其中data字段类型为UserResponse
- `@Router`: 路由定义（path + httpMethod）
  - 路径相对于`@BasePath`（/api/v1）

#### 3.2 用户登录接口

```go
// Login 用户登录
// @Summary      用户登录
// @Description  验证邮箱密码，返回JWT Token
// @Tags         用户模块
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "登录信息"
// @Success      200 {object} response.Response{data=dto.LoginResponse} "登录成功，返回access_token和refresh_token"
// @Failure      400 {object} response.Response "参数错误"
// @Failure      401 {object} response.Response "邮箱或密码错误"
// @Failure      404 {object} response.Response "用户不存在"
// @Router       /users/login [post]
//
// 教学说明：JWT认证流程
// 1. 客户端发送邮箱+密码
// 2. 服务端验证密码（bcrypt对比哈希值）
// 3. 验证成功后生成JWT Token：
//    - Access Token: 有效期2小时，用于API认证
//    - Refresh Token: 有效期7天，用于刷新Access Token
// 4. 将Session信息存储到Redis（用于登出功能）
// 5. 返回Token给客户端
// 6. 客户端后续请求携带Token: Authorization: Bearer <token>
```

#### 3.3 图书上架接口（需要认证）

```go
// PublishBook 发布图书(上架)
// @Summary      发布图书
// @Description  会员发布图书商品上架（需要登录）
// @Tags         图书模块
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.PublishBookRequest true "图书信息"
// @Success      200 {object} response.Response{data=dto.BookResponse} "上架成功"
// @Failure      400 {object} response.Response "参数错误（如ISBN格式错误、价格超出范围）"
// @Failure      401 {object} response.Response "未登录"
// @Failure      409 {object} response.Response "ISBN已存在"
// @Router       /books [post]
//
// 教学说明：@Security注释
// - @Security BearerAuth: 表示此接口需要JWT认证
// - BearerAuth在main.go中定义为securityDefinitions
// - Swagger UI会显示🔒图标，并提供Token输入框
// - 测试时需先调用/users/login获取token，然后点击Authorize按钮输入
```

**@Security的作用**:
- Swagger UI会在接口右侧显示🔒图标
- 点击"Authorize"按钮可以输入JWT Token
- 输入后，所有带`@Security`的接口请求会自动携带Token

#### 3.4 图书列表接口（Query参数）

```go
// ListBooks 查询图书列表
// @Summary      图书列表
// @Description  分页查询图书列表，支持关键词搜索和排序（公开接口，无需登录）
// @Tags         图书模块
// @Accept       json
// @Produce      json
// @Param        page      query    int    false "页码（默认1）" default(1) minimum(1)
// @Param        page_size query    int    false "每页数量（默认20，最大100）" default(20) minimum(1) maximum(100)
// @Param        keyword   query    string false "搜索关键词（匹配标题/作者/出版社）"
// @Param        sort_by   query    string false "排序方式" Enums(price_asc, price_desc, created_at_desc) default(created_at_desc)
// @Success      200 {object} response.Response{data=dto.ListBooksResponse} "查询成功"
// @Failure      400 {object} response.Response "参数错误（如page_size超过100）"
// @Router       /books [get]
//
// 教学说明：Query参数注释
// - @Param的格式: name in type required comment [attributes]
// - in类型: query（URL参数）| path（路径参数）| body（请求体）| header（请求头）
// - attributes（可选）:
//   - default(value): 默认值
//   - minimum(value): 最小值
//   - maximum(value): 最大值
//   - Enums(v1,v2,v3): 枚举值
// - Swagger UI会根据这些属性生成友好的输入控件（如下拉框、数字输入框）
```

**Query参数的attributes**:
- `default(1)`: 默认值为1
- `minimum(1)`: 最小值为1
- `maximum(100)`: 最大值为100
- `Enums(...)`: 枚举值，Swagger UI会渲染为下拉框

#### 3.5 创建订单接口（核心功能）

```go
// CreateOrder 创建订单
// @Summary      创建订单
// @Description  用户下单购买图书（需要登录），使用悲观锁防止超卖
// @Tags         订单模块
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateOrderRequest true "订单信息"
// @Success      200 {object} response.Response{data=dto.CreateOrderResponse} "下单成功"
// @Failure      400 {object} response.Response "参数错误（如商品数量超过999）"
// @Failure      401 {object} response.Response "未登录"
// @Failure      404 {object} response.Response "图书不存在"
// @Failure      50001 {object} response.Response "库存不足"
// @Router       /orders [post]
//
// 教学说明：防超卖的核心逻辑
// 本接口是整个项目的核心功能之一，演示了如何在高并发场景下防止库存超卖。
//
// 实现方案：悲观锁（SELECT FOR UPDATE）
// 1. 开启数据库事务
// 2. 使用SELECT FOR UPDATE锁定库存行
// 3. 检查库存是否充足
// 4. 创建订单
// 5. 扣减库存
// 6. 提交事务
```

---

### 4. 生成Swagger文档

运行Swag工具生成OpenAPI规范的文档：

```bash
cd /home/xiebiao/Workspace/bookstore
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

**生成的文件**:
```
docs/
├── docs.go          # Go代码，可以被import
├── swagger.json     # OpenAPI JSON格式
└── swagger.yaml     # OpenAPI YAML格式
```

**参数说明**:
- `-g cmd/api/main.go`: 指定包含`@title`等全局注释的入口文件
- `-o docs`: 输出目录
- `--parseDependency`: 解析依赖包中的注释
- `--parseInternal`: 解析internal包中的注释

---

### 5. 集成Swagger UI

在`wire.go`中添加Swagger路由：

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
)

func provideGinEngine(...) *gin.Engine {
    r := gin.Default()
    
    // Swagger文档路由
    // 访问 http://localhost:8080/swagger/index.html 查看API文档
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    
    // ... 其他路由
    return r
}
```

**教学说明**:
- `ginSwagger.WrapHandler`: Gin的Swagger UI中间件
- `swaggerFiles.Handler`: 提供swagger.json等静态文件
- 路由使用通配符`/*any`来匹配所有Swagger UI资源

在`main.go`中导入生成的docs包：

```go
import (
    _ "github.com/xiebiao/bookstore/docs" // Swagger文档导入
)
```

**为什么使用空导入（`_`）？**
- docs包的init()函数会自动注册Swagger文档到全局变量
- 我们不直接调用docs包的函数，只需要触发init()
- 使用`_`告诉Go编译器：虽然没用这个包，但不要移除这个导入

---

## 🎓 教学要点总结

### 1. Swagger vs 手动文档

| 特性 | Swagger | 手动维护的Markdown |
|------|---------|-------------------|
| 文档位置 | 代码注释中 | 独立的.md文件 |
| 维护成本 | 低（修改代码时同步修改注释） | 高（需单独维护文档）|
| 文档准确性 | 高（与代码在一起） | 低（容易过时） |
| 交互测试 | 支持（Swagger UI） | 不支持 |
| 客户端生成 | 支持（swagger-codegen） | 不支持 |

### 2. Swagger注释的核心概念

#### 全局配置（main.go）
```go
// @title       API标题
// @version     版本号
// @description API描述
// @host        服务地址
// @BasePath    基础路径
// @securityDefinitions.apikey 认证定义
```

#### 接口配置（handler.go）
```go
// @Summary      简短描述
// @Description  详细描述
// @Tags         分组标签
// @Accept       请求格式
// @Produce      响应格式
// @Param        参数定义
// @Success      成功响应
// @Failure      失败响应
// @Security     认证要求
// @Router       路由路径 [方法]
```

### 3. 参数类型详解

#### Body参数（JSON请求体）
```go
// @Param request body dto.RegisterRequest true "注册信息"
```

#### Query参数（URL参数）
```go
// @Param page query int false "页码" default(1) minimum(1)
```

#### Path参数（路径参数）
```go
// @Param id path int true "用户ID"
// 对应路由: /users/:id
```

#### Header参数
```go
// @Param Authorization header string true "Bearer Token"
```

### 4. 响应类型的泛型写法

```go
// 基础响应
// @Success 200 {object} response.Response

// 响应带数据（data字段类型为UserResponse）
// @Success 200 {object} response.Response{data=dto.UserResponse}

// 响应带数组数据
// @Success 200 {object} response.Response{data=[]dto.BookListItem}
```

### 5. 认证流程

**Step 1**: 在main.go中定义认证方式
```go
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
```

**Step 2**: 在需要认证的接口添加`@Security`
```go
// @Security BearerAuth
```

**Step 3**: 在Swagger UI中使用
1. 调用`/users/login`接口获取token
2. 点击Swagger UI右上角的"Authorize"按钮
3. 输入`Bearer <你的token>`
4. 后续所有带🔒的接口会自动携带Token

---

## 📊 实现效果

### Swagger UI界面

访问 http://localhost:8080/swagger/index.html

**功能特性**:
1. **接口分组**: 按Tags分组（用户模块、图书模块、订单模块）
2. **交互测试**: 点击"Try it out"可直接测试接口
3. **参数说明**: 每个参数都有类型、是否必填、示例值
4. **响应示例**: 显示Success和Failure的响应结构
5. **认证支持**: 点击Authorize输入JWT Token
6. **请求示例**: 自动生成curl命令和各语言的SDK调用代码

### API列表

**用户模块**:
- POST /api/v1/users/register - 用户注册
- POST /api/v1/users/login - 用户登录

**图书模块**:
- GET /api/v1/books - 图书列表（支持分页、搜索、排序）
- POST /api/v1/books - 发布图书（需登录）

**订单模块**:
- POST /api/v1/orders - 创建订单（需登录，防超卖）

---

## 📁 新增/修改文件清单

### 新增文件（4个）
```
docs/
├── docs.go          # Swagger文档的Go代码
├── swagger.json     # OpenAPI JSON格式
└── swagger.yaml     # OpenAPI YAML格式

（wire_gen.go自动更新）
```

### 修改文件（5个）
```
cmd/api/main.go
  - 添加Swagger全局注释（@title, @host等）
  - 导入docs包
  - 更新启动信息（添加Swagger URL）

cmd/api/wire.go
  - 导入ginSwagger和swaggerFiles
  - 注册Swagger路由

internal/interface/http/handler/user.go
  - Register接口添加Swagger注释
  - Login接口添加Swagger注释

internal/interface/http/handler/book.go
  - PublishBook接口添加Swagger注释
  - ListBooks接口添加Swagger注释

internal/interface/http/handler/order.go
  - CreateOrder接口添加Swagger注释

go.mod
  - 新增github.com/swaggo/gin-swagger
  - 新增github.com/swaggo/files
  - 新增github.com/swaggo/swag
```

---

## 🧪 测试验证

### 1. 构建测试
```bash
$ cd /home/xiebiao/Workspace/bookstore
$ go build -o bin/api ./cmd/api
# 构建成功，无错误
```

### 2. 启动测试
```bash
$ ./bin/api
🚀 服务启动成功（使用Wire依赖注入 + Swagger文档）
   访问地址: http://localhost:8080
   健康检查: http://localhost:8080/ping
   API文档: http://localhost:8080/swagger/index.html

   教学要点：
   - Wire自动生成了所有依赖注入代码（见wire_gen.go）
   - Swagger自动生成了API文档（见docs/swagger.json）
   - main.go从100+行精简到30行
   - 依赖管理集中在wire.go，职责清晰
```

### 3. Swagger UI测试
```bash
$ curl -s http://localhost:8080/swagger/index.html | head -5
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
```

### 4. API文档JSON测试
```bash
$ curl -s http://localhost:8080/swagger/doc.json | jq '.paths | keys'
[
  "/books",
  "/orders",
  "/users/login",
  "/users/register"
]
```

---

## 💡 最佳实践

### 1. 注释编写规范

**DO（推荐）**:
```go
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 用户模块
```

**DON'T（不推荐）**:
```go
// @Summary 注册  // 太简短，不清楚
// @Tags Users    // 使用英文，不统一
```

### 2. 参数描述要详细

**DO**:
```go
// @Param page_size query int false "每页数量（默认20，最大100）" default(20) minimum(1) maximum(100)
```

**DON'T**:
```go
// @Param page_size query int false "每页数量"  // 缺少约束说明
```

### 3. 响应类型要精确

**DO**:
```go
// @Success 200 {object} response.Response{data=dto.UserResponse} "注册成功"
```

**DON'T**:
```go
// @Success 200 {object} response.Response  // 缺少data字段类型
```

### 4. 生产环境考虑

**安全措施**:
```go
// 方法1: 通过环境变量控制
if os.Getenv("ENABLE_SWAGGER") == "true" {
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// 方法2: 添加Basic Auth
authorized := r.Group("/swagger", gin.BasicAuth(gin.Accounts{
    "admin": "password",
}))
authorized.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

---

## 🚀 下一步计划

根据ROADMAP.md，接下来是：
- **Day 18**: Makefile + README完善

---

## 📚 参考资料

- [Swag官方文档](https://github.com/swaggo/swag)
- [OpenAPI规范](https://swagger.io/specification/)
- [Swagger UI文档](https://swagger.io/tools/swagger-ui/)
- 项目内部文档: TEACHING.md, ROADMAP.md

---

**报告生成时间**: 2025-11-05  
**实现周期**: Week 3 Day 17  
**新增代码**: Swagger注释约200行，docs/自动生成约1000行  
**测试结果**: ✅ 全部通过  
**功能特性**: 5个API接口 + 交互式测试 + JWT认证
