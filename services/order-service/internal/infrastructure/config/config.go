package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
//
// 教学要点：
// 1. 配置结构化：使用struct而非全局变量
//   - 便于测试（可以创建测试配置）
//   - 类型安全（编译期检查）
//   - IDE友好（自动补全）
//
// 2. 配置分组：Server、Database、Redis等
//   - 清晰的配置边界
//   - 便于扩展
type Config struct {
	Server   ServerConfig             `mapstructure:"server"`
	Database DatabaseConfig           `mapstructure:"database"`
	Redis    RedisConfig              `mapstructure:"redis"`
	Order    OrderConfig              `mapstructure:"order"`
	Services map[string]ServiceConfig `mapstructure:"services"` // 下游服务配置
	Log      LogConfig                `mapstructure:"log"`
}

// ServerConfig gRPC服务配置
type ServerConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	LogMode         bool   `mapstructure:"log_mode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// OrderConfig 订单业务配置
type OrderConfig struct {
	PaymentTimeout     int    `mapstructure:"payment_timeout"`       // 支付超时（分钟）
	OrderNoPrefix      string `mapstructure:"order_no_prefix"`       // 订单号前缀
	MaxItemsPerOrder   int    `mapstructure:"max_items_per_order"`   // 单个订单最多商品种类
	MaxQuantityPerItem int    `mapstructure:"max_quantity_per_item"` // 单个商品最大数量
}

// ServiceConfig 下游服务配置
//
// 教学要点：
// 为什么需要配置超时时间？
// - 防雪崩：下游服务hang住时，不会无限等待
// - 快速失败：超时后立即返回错误，释放资源
// - 用户体验：避免长时间等待
type ServiceConfig struct {
	Addr    string `mapstructure:"addr"`
	Timeout int    `mapstructure:"timeout"` // 超时时间（秒）
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load 加载配置文件
//
// 教学要点：
// 1. Viper配置加载流程：
//   - SetConfigFile：指定配置文件路径
//   - ReadInConfig：读取文件
//   - Unmarshal：解析到struct
//
// 2. 为什么使用Viper而非encoding/json？
//   - 支持多格式（YAML、JSON、TOML）
//   - 支持环境变量覆盖（12-Factor App）
//   - 支持配置热更新（Watch）
//
// 3. 错误处理：
//   - 配置加载失败应该panic（无法启动服务）
//   - 使用log.Fatalf而非panic（打印错误信息）
func Load(configPath string) *Config {
	v := viper.New()

	// 设置配置文件
	v.SetConfigFile(configPath)

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	// 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	// 设置默认值（可选）
	setDefaults(&cfg)

	log.Printf("✅ 配置加载成功: %s", configPath)
	return &cfg
}

// setDefaults 设置默认值
//
// 教学要点：
// 防御性编程：配置文件缺失某些字段时，使用合理的默认值
func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9005
	}

	if cfg.Order.PaymentTimeout == 0 {
		cfg.Order.PaymentTimeout = 15 // 默认15分钟
	}

	if cfg.Order.MaxItemsPerOrder == 0 {
		cfg.Order.MaxItemsPerOrder = 20
	}

	if cfg.Order.MaxQuantityPerItem == 0 {
		cfg.Order.MaxQuantityPerItem = 99
	}
}

// GetServiceAddr 获取下游服务地址
func (c *Config) GetServiceAddr(serviceName string) string {
	if svc, ok := c.Services[serviceName]; ok {
		return svc.Addr
	}
	return ""
}

// GetServiceTimeout 获取下游服务超时时间
func (c *Config) GetServiceTimeout(serviceName string) time.Duration {
	if svc, ok := c.Services[serviceName]; ok {
		return time.Duration(svc.Timeout) * time.Second
	}
	return 5 * time.Second // 默认5秒
}

// Validate 验证配置合法性
//
// 教学要点：
// 配置验证的最佳实践：
// 1. 启动时验证，而非运行时（Fail Fast原则）
// 2. 验证关键配置：数据库DSN、必需的服务地址
// 3. 给出清晰的错误提示
func (c *Config) Validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn 不能为空")
	}

	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr 不能为空")
	}

	// 验证必需的下游服务
	requiredServices := []string{"inventory", "catalog"}
	for _, svc := range requiredServices {
		if addr := c.GetServiceAddr(svc); addr == "" {
			return fmt.Errorf("services.%s.addr 不能为空", svc)
		}
	}

	return nil
}
