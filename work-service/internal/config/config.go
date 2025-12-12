package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Services ServicesConfig
	RabbitMQ RabbitMQConfig
	Logging  LoggingConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type ServiceConfig struct {
	URL             string
	UploadEndpoint  string
	ReportsEndpoint string
	Timeout         time.Duration
	RetryCount      int
	RetryDelay      time.Duration
}

type ServicesConfig struct {
	File     ServiceConfig
	Analysis ServiceConfig
}

type RabbitMQConfig struct {
	URL        string
	Exchange   string
	RoutingKey string
	QueueName  string
}

type LoggingConfig struct {
	Level   string
	Pretty  bool
	NoColor bool
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
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
	viper.SetDefault("server.address", ":8081")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "work_user")
	viper.SetDefault("database.password", "work_password")
	viper.SetDefault("database.name", "work_db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// File service defaults
	viper.SetDefault("services.file.url", "http://file-service:8082")
	viper.SetDefault("services.file.upload_endpoint", "/api/v1/files/upload")
	viper.SetDefault("services.file.timeout", "30s")
	viper.SetDefault("services.file.retry_count", 3)
	viper.SetDefault("services.file.retry_delay", "100ms")

	// Analysis service defaults
	viper.SetDefault("services.analysis.url", "http://analysis-service:8083")
	viper.SetDefault("services.analysis.reports_endpoint", "/api/v1/reports/work")
	viper.SetDefault("services.analysis.timeout", "10s")
	viper.SetDefault("services.analysis.retry_count", 3)
	viper.SetDefault("services.analysis.retry_delay", "100ms")

	// RabbitMQ defaults
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.exchange", "plagiarism_exchange")
	viper.SetDefault("rabbitmq.routing_key", "work.created")
	viper.SetDefault("rabbitmq.queue_name", "work_created_queue")

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
