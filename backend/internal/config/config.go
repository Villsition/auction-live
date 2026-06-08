package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
	Redis  RedisConfig  `mapstructure:"redis"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Database, d.Charset)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
	// Optional read replica. Falls back to Host:Port if empty.
	ReadHost string `mapstructure:"read_host"`
	ReadPort int    `mapstructure:"read_port"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// ReadAddr returns the read replica address, or the write address if not configured.
func (r RedisConfig) ReadAddr() string {
	if r.ReadHost == "" {
		return r.Addr()
	}
	port := r.ReadPort
	if port == 0 {
		port = r.Port
	}
	return fmt.Sprintf("%s:%d", r.ReadHost, port)
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// 1. Load YAML as defaults
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		// Config file is optional if env vars are set
		fmt.Printf("warning: config file not found: %v, using env vars\n", err)
	}

	// 2. Override with environment variables
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 3. Bind specific env vars to config keys
	v.BindEnv("db.user", "DB_USER")
	v.BindEnv("db.password", "DB_PASSWORD")
	v.BindEnv("db.host", "DB_HOST")
	v.BindEnv("db.port", "DB_PORT")
	v.BindEnv("db.database", "DB_NAME")

	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")

	// Also support REDIS_ADDR format (host:port)
	v.BindEnv("redis.addr", "REDIS_ADDR")
	if addr := v.GetString("redis.addr"); addr != "" {
		parts := strings.Split(addr, ":")
		if len(parts) >= 1 {
			v.Set("redis.host", parts[0])
		}
		if len(parts) >= 2 {
			var port int
			fmt.Sscanf(parts[1], "%d", &port)
			v.Set("redis.port", port)
		}
	}

	v.BindEnv("jwt.secret", "JWT_SECRET")

	v.BindEnv("server.port", "SERVER_PORT")

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("db.charset", "utf8mb4")
	v.SetDefault("db.max_idle_conns", 10)
	v.SetDefault("db.max_open_conns", 100)
	v.SetDefault("redis.host", "127.0.0.1")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 100)
	v.SetDefault("jwt.expire_hours", 24)
	v.SetDefault("log.level", "debug")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
