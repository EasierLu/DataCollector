package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Initialized bool            `yaml:"initialized"`
	Server      ServerConfig    `yaml:"server"`
	TLS         TLSConfig       `yaml:"tls"`
	Database    DatabaseConfig  `yaml:"database"`
	JWT         JWTConfig       `yaml:"jwt"`
	Collector   CollectorConfig `yaml:"collector"`
	Log         LogConfig       `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug, release
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string         `yaml:"driver"`
	SQLite   SQLiteConfig   `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// SQLiteConfig SQLite配置
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	Expiration time.Duration `yaml:"expiration"`
}

// CollectorConfig 数据采集配置
type CollectorConfig struct {
	MaxBodySize       int64    `yaml:"max_body_size"`
	RateLimitPerToken int      `yaml:"rate_limit_per_token"`
	RateLimitPerIP    int      `yaml:"rate_limit_per_ip"`
	AllowedOrigins    []string `yaml:"allowed_origins"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
	MaxSize  int    `yaml:"max_size"` // MB
	MaxAge   int    `yaml:"max_age"`  // days
}

// Load 从YAML文件加载配置
// 如果文件不存在返回 nil, nil（区分"文件不存在"和"解析失败"）
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用环境变量覆盖
	cfg.applyEnvOverrides()

	return &cfg, nil
}

// Save 将当前配置序列化写回 YAML 文件
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Initialized: false,
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "debug",
		},
		TLS: TLSConfig{
			Enabled:  false,
			CertFile: "",
			KeyFile:  "",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			SQLite: SQLiteConfig{
				Path: "./data/datacollector.db",
			},
			Postgres: PostgresConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "datacollector",
				Password: "",
				DBName:   "datacollector",
				SSLMode:  "disable",
			},
		},
		JWT: JWTConfig{
			Secret:     "change-me-to-a-secure-random-string",
			Expiration: 24 * time.Hour,
		},
		Collector: CollectorConfig{
			MaxBodySize:       1048576,
			RateLimitPerToken: 100,
			RateLimitPerIP:    200,
			AllowedOrigins:    []string{"*"},
		},
		Log: LogConfig{
			Level:    "info",
			Format:   "json",
			Output:   "stdout",
			FilePath: "./logs/datacollector.log",
			MaxSize:  100,
			MaxAge:   30,
		},
	}
}

// applyEnvOverrides 应用环境变量覆盖配置
func (c *Config) applyEnvOverrides() {
	// 数据库驱动
	if v := os.Getenv("DB_DRIVER"); v != "" {
		c.Database.Driver = v
	}

	// SQLite 路径
	if v := os.Getenv("DB_SQLITE_PATH"); v != "" {
		c.Database.SQLite.Path = v
	}

	// PostgreSQL 配置
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Postgres.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Postgres.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.Postgres.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Postgres.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.Postgres.DBName = v
	}

	// 服务器配置
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("SERVER_MODE"); v != "" {
		c.Server.Mode = v
	}

	// TLS 配置
	if v := os.Getenv("TLS_ENABLED"); v != "" {
		c.TLS.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("TLS_CERT_FILE"); v != "" {
		c.TLS.CertFile = v
	}
	if v := os.Getenv("TLS_KEY_FILE"); v != "" {
		c.TLS.KeyFile = v
	}

	// JWT 密钥
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWT.Secret = v
	}

	// CORS 允许的源（逗号分隔）
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		c.Collector.AllowedOrigins = origins
	}

	// 日志配置
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("LOG_OUTPUT"); v != "" {
		c.Log.Output = v
	}
	if v := os.Getenv("LOG_FILE_PATH"); v != "" {
		c.Log.FilePath = v
	}
}

// DSN 返回数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case "sqlite":
		return c.SQLite.Path
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Postgres.Host,
			c.Postgres.Port,
			c.Postgres.User,
			c.Postgres.Password,
			c.Postgres.DBName,
			c.Postgres.SSLMode,
		)
	default:
		return ""
	}
}
