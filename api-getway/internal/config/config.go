package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Proxy    ProxyConfig    `mapstructure:"proxy"`
	Services ServicesConfig `mapstructure:"services"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type ServerConfig struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type ProxyConfig struct {
	Timeout         time.Duration `mapstructure:"timeout"`
	MaxIdleConns    int           `mapstructure:"max_idle_connections"`
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`
}

type ServiceConfig struct {
	URL            string        `mapstructure:"url"`
	HealthEndpoint string        `mapstructure:"health_endpoint"`
	Timeout        time.Duration `mapstructure:"timeout"`
	RetryCount     int           `mapstructure:"retry_count"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
}

type ServicesConfig struct {
	Work     ServiceConfig `mapstructure:"work"`
	File     ServiceConfig `mapstructure:"file"`
	Analysis ServiceConfig `mapstructure:"analysis"`
}

type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	Pretty  bool   `mapstructure:"pretty"`
	NoColor bool   `mapstructure:"no_color"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Установка значений по умолчанию
	setDefaults()

	// Чтение конфигурации
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Привязка переменных окружения
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Загрузка конфигурации в структуру
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	// Proxy defaults
	viper.SetDefault("proxy.timeout", "30s")
	viper.SetDefault("proxy.max_idle_connections", 100)
	viper.SetDefault("proxy.idle_conn_timeout", "90s")

	// Work service defaults
	viper.SetDefault("services.work.url", "http://work-service:8081")
	viper.SetDefault("services.work.health_endpoint", "/health")
	viper.SetDefault("services.work.timeout", "10s")
	viper.SetDefault("services.work.retry_count", 3)
	viper.SetDefault("services.work.retry_delay", "100ms")

	// File service defaults
	viper.SetDefault("services.file.url", "http://file-service:8082")
	viper.SetDefault("services.file.health_endpoint", "/health")
	viper.SetDefault("services.file.timeout", "15s")
	viper.SetDefault("services.file.retry_count", 3)
	viper.SetDefault("services.file.retry_delay", "100ms")

	// Analysis service defaults
	viper.SetDefault("services.analysis.url", "http://analysis-service:8083")
	viper.SetDefault("services.analysis.health_endpoint", "/health")
	viper.SetDefault("services.analysis.timeout", "10s")
	viper.SetDefault("services.analysis.retry_count", 3)
	viper.SetDefault("services.analysis.retry_delay", "100ms")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.pretty", false)
	viper.SetDefault("logging.no_color", false)

	// CORS defaults
	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	viper.SetDefault("cors.exposed_headers", []string{"Link"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 300)
}
