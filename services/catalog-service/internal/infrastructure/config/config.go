package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
// 教学要点：
// 1. 使用结构体承载配置，类型安全
// 2. 支持嵌套配置（server、database、redis等）
// 3. 配置验证（确保必填项不为空）
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig 服务器配置
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

// CacheConfig 缓存配置
type CacheConfig struct {
	ListTTL   int `mapstructure:"list_ttl"`
	DetailTTL int `mapstructure:"detail_ttl"`
	SearchTTL int `mapstructure:"search_ttl"`
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
// 1. 使用Viper加载YAML配置
// 2. 支持环境变量覆盖（格式：CATALOG_SERVER_PORT）
// 3. 配置验证（必填项检查）
//
// DO（正确做法）：
// - 配置文件提供默认值
// - 环境变量覆盖敏感信息（如数据库密码）
// - 启动时验证配置完整性
//
// DON'T（错误做法）：
// - 硬编码敏感信息到配置文件
// - 不验证配置就直接使用（可能导致运行时错误）
func Load(configPath string) (*Config, error) {
	// 步骤1：设置配置文件路径
	viper.SetConfigFile(configPath)

	// 步骤2：设置环境变量前缀和自动绑定
	// 环境变量格式：CATALOG_SERVER_PORT、CATALOG_DATABASE_DSN
	viper.SetEnvPrefix("CATALOG")
	viper.AutomaticEnv()

	// 步骤3：读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 步骤4：解析到结构体
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 步骤5：验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证服务端口
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的服务端口: %d", c.Server.Port)
	}

	// 验证数据库配置
	if c.Database.DSN == "" {
		return fmt.Errorf("数据库DSN不能为空")
	}

	// 验证Redis配置
	if c.Redis.Addr == "" {
		return fmt.Errorf("Redis地址不能为空")
	}

	return nil
}

// GetConnMaxLifetime 获取连接最大生命周期（time.Duration）
func (c *DatabaseConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

// GetListTTL 获取列表缓存时间（time.Duration）
func (c *CacheConfig) GetListTTL() time.Duration {
	return time.Duration(c.ListTTL) * time.Second
}

// GetDetailTTL 获取详情缓存时间（time.Duration）
func (c *CacheConfig) GetDetailTTL() time.Duration {
	return time.Duration(c.DetailTTL) * time.Second
}

// GetSearchTTL 获取搜索缓存时间（time.Duration）
func (c *CacheConfig) GetSearchTTL() time.Duration {
	return time.Duration(c.SearchTTL) * time.Second
}
