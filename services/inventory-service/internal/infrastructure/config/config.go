package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Inventory InventoryConfig `mapstructure:"inventory"`
	Log       LogConfig       `mapstructure:"log"`
}

type ServerConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	LogMode         bool   `mapstructure:"log_mode"`
}

type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// InventoryConfig 库存配置
// 教学要点：业务配置与技术配置分离
type InventoryConfig struct {
	EnableCache      bool `mapstructure:"enable_cache"`
	WarningThreshold int  `mapstructure:"warning_threshold"`
	SyncInterval     int  `mapstructure:"sync_interval"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetEnvPrefix("INVENTORY")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("无效的服务端口: %d", c.Server.Port)
	}

	if c.Database.DSN == "" {
		return fmt.Errorf("数据库DSN不能为空")
	}

	if c.Redis.Addr == "" {
		return fmt.Errorf("Redis地址不能为空")
	}

	return nil
}

func (c *DatabaseConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}

func (c *InventoryConfig) GetSyncInterval() time.Duration {
	return time.Duration(c.SyncInterval) * time.Second
}
