package main

import (
	"fmt"
	"log"

	_ "github.com/xiebiao/bookstore/docs" // Swagger文档导入
)

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
//
// @externalDocs.description   项目文档
// @externalDocs.url           https://github.com/xiebiao/bookstore/blob/main/README.md
//
// 教学说明：Swagger注释格式
// - @title: API文档标题
// - @version: API版本号
// - @description: API描述（支持多行）
// - @host: API服务地址
// - @BasePath: API基础路径
// - @securityDefinitions: 定义认证方式（JWT Bearer Token）
// - @contact: 联系信息
// - @license: 许可证信息
//
// Swagger的价值：
// 1. 自动生成API文档，减少手动维护成本
// 2. 提供交互式测试界面，方便前端调试
// 3. 文档与代码同步，避免文档过时
// 4. 支持多语言客户端SDK生成

// main 主程序入口
// 当前版本：Phase 1 - Week 3 Day 15-16 - Wire依赖注入
//
// 教学说明：
// 对比重构前后的main.go：
//
// 重构前（手动依赖注入）：
// - 需要手动创建所有依赖（60+行代码）
// - 依赖顺序容易出错
// - 新增依赖需要手动调整多处代码
// - 代码冗长，可读性差
//
// 重构后（Wire自动生成）：
// - 只需调用InitializeApp()（1行代码）
// - Wire自动分析依赖链，保证顺序正确
// - 新增依赖只需在wire.go中添加Provider
// - main.go专注于启动逻辑，职责清晰
//
// Wire的价值：
// 1. 编译期生成代码，零运行时开销
// 2. 类型安全，编译期检测依赖错误
// 3. 自动检测循环依赖
// 4. 代码可读性高，便于维护
func main() {
	// 使用Wire初始化整个应用
	// Wire会自动：
	// 1. 加载配置
	// 2. 初始化数据库和Redis连接
	// 3. 创建所有Repository、Service、UseCase、Handler
	// 4. 注册所有路由
	// 5. 返回配置好的Gin引擎
	engine, err := InitializeApp()
	if err != nil {
		log.Fatalf("应用初始化失败: %v", err)
	}

	// 启动服务（保留原有的启动信息打印）
	// 注意：这里无法访问cfg对象（Wire的返回值限制）
	// 解决方案：将端口号硬编码或从环境变量读取
	// 生产环境建议：使用环境变量或配置文件
	port := 8080 // 默认端口，与config.yaml保持一致

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("\n🚀 服务启动成功（使用Wire依赖注入 + Swagger文档）\n")
	fmt.Printf("   访问地址: http://localhost%s\n", addr)
	fmt.Printf("   健康检查: http://localhost%s/ping\n", addr)
	fmt.Printf("   API文档: http://localhost%s/swagger/index.html\n", addr)
	fmt.Printf("\n   教学要点：\n")
	fmt.Printf("   - Wire自动生成了所有依赖注入代码（见wire_gen.go）\n")
	fmt.Printf("   - Swagger自动生成了API文档（见docs/swagger.json）\n")
	fmt.Printf("   - main.go从100+行精简到30行\n")
	fmt.Printf("   - 依赖管理集中在wire.go，职责清晰\n")
	fmt.Printf("\n按Ctrl+C停止服务\n\n")

	if err := engine.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// 教学总结：Wire重构带来的好处
//
// 1. 代码简洁性
//    - 重构前：100+行（包含所有依赖创建代码）
//    - 重构后：30行（只保留启动逻辑）
//    - 减少代码量：70%+
//
// 2. 可维护性
//    - 依赖管理集中在wire.go
//    - main.go只关注启动流程
//    - 新增功能只需修改wire.go
//
// 3. 类型安全
//    - 编译期检查依赖关系
//    - 自动检测循环依赖
//    - 参数类型不匹配时编译失败
//
// 4. 开发效率
//    - 无需手动管理依赖顺序
//    - 重构时自动更新依赖链
//    - 减少人为错误
//
// 5. 性能
//    - 编译期生成，零运行时反射
//    - 与手写代码性能完全相同
//    - 无运行时开销
//
// 对比Spring的@Autowired（运行时反射注入）：
// - Spring：运行时通过反射扫描@Autowired注解，动态注入依赖
// - Wire：编译期生成代码，生成的是普通Go函数调用
// - Spring的灵活性更高（可以热加载），但有运行时开销
// - Wire牺牲了一些灵活性，但性能更好，更符合Go的设计哲学
