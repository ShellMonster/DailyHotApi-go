package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用全局配置结构体
// 这个结构体就像一个"配置清单",存储了所有应用需要的设置
type Config struct {
	Server ServerConfig `mapstructure:"server"` // 服务器配置
	Cache  CacheConfig  `mapstructure:"cache"`  // 缓存配置
	Redis  RedisConfig  `mapstructure:"redis"`  // Redis 配置
	Log    LogConfig    `mapstructure:"log"`    // 日志配置
}

// ServerConfig 服务器配置
// 定义了 HTTP 服务器的基本参数
type ServerConfig struct {
	Port         int           `mapstructure:"port"`          // 监听端口,比如 6688
	Host         string        `mapstructure:"host"`          // 监听地址,比如 "0.0.0.0"
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`  // 读取超时时间
	WriteTimeout time.Duration `mapstructure:"write_timeout"` // 写入超时时间
	Prefork      bool          `mapstructure:"prefork"`       // 是否启用多进程模式(提高并发性能)
}

// CacheConfig 内存缓存配置 (BigCache)
// 就像一个超快的"货架",可以存放最常用的数据
type CacheConfig struct {
	Enabled          bool          `mapstructure:"enabled"`             // 是否启用缓存
	DefaultExpire    time.Duration `mapstructure:"default_expire"`      // 默认过期时间,比如 5 分钟
	CleanupInterval  time.Duration `mapstructure:"cleanup_interval"`    // 清理过期数据的间隔
	MaxEntries       int           `mapstructure:"max_entries"`         // 最大条目数
	MaxEntrySize     int           `mapstructure:"max_entry_size"`      // 单个条目最大大小(字节)
	HardMaxCacheSize int           `mapstructure:"hard_max_cache_size"` // 缓存总大小上限(MB)
}

// RedisConfig Redis 配置
// Redis 是一个分布式缓存,就像"共享仓库",多台服务器可以共用
type RedisConfig struct {
	Enabled  bool          `mapstructure:"enabled"`   // 是否启用 Redis
	Host     string        `mapstructure:"host"`      // Redis 服务器地址
	Port     int           `mapstructure:"port"`      // Redis 端口
	Password string        `mapstructure:"password"`  // 密码(如果有)
	DB       int           `mapstructure:"db"`        // 数据库编号(0-15)
	PoolSize int           `mapstructure:"pool_size"` // 连接池大小
	Timeout  time.Duration `mapstructure:"timeout"`   // 连接超时时间
}

// LogConfig 日志配置
// 控制日志如何输出、存储
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别: debug, info, warn, error
	Format     string `mapstructure:"format"`      // 输出格式: json 或 console
	OutputPath string `mapstructure:"output_path"` // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 日志文件保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志
}

var globalConfig *Config

// Load 加载配置文件
// 这个函数负责从配置文件或环境变量中读取所有配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径和名称
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// 默认查找当前目录下的 config.yaml
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
	}

	// 支持从环境变量读取配置,环境变量优先级更高
	// 比如环境变量 DAILYHOT_SERVER_PORT=8080 可以覆盖配置文件中的 server.port
	v.SetEnvPrefix("DAILYHOT")
	v.AutomaticEnv()

	// 设置默认值,如果配置文件和环境变量都没设置,就用这些默认值
	setDefaults(v)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在,使用默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 将配置解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// setDefaults 设置默认配置
// 这些是合理的默认值,即使没有配置文件也能正常运行
func setDefaults(v *viper.Viper) {
	// 服务器默认配置
	v.SetDefault("server.port", 6688)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 10*time.Second)
	v.SetDefault("server.prefork", false)

	// 内存缓存默认配置
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.default_expire", 5*time.Minute)
	v.SetDefault("cache.cleanup_interval", 10*time.Minute)
	v.SetDefault("cache.max_entries", 10000)
	v.SetDefault("cache.max_entry_size", 500)      // 500 字节
	v.SetDefault("cache.hard_max_cache_size", 256) // 256 MB

	// Redis 默认配置
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.timeout", 5*time.Second)

	// 日志默认配置
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output_path", "logs/app.log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 5)
	v.SetDefault("log.max_age", 30)
	v.SetDefault("log.compress", true)
}

// Get 获取全局配置实例
// 在其他地方可以通过这个函数获取配置
func Get() *Config {
	return globalConfig
}
