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
	Analysis AnalysisConfig
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
	URL           string
	WorksEndpoint string
	FilesEndpoint string
	HashEndpoint  string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
}

type ServicesConfig struct {
	Work ServiceConfig
	File ServiceConfig
}

type RabbitMQConfig struct {
	URL           string
	Exchange      string
	RoutingKey    string
	QueueName     string
	ConsumerTag   string
	PrefetchCount int
}

type AnalysisConfig struct {
	HashAlgorithm         string
	SimilarityThreshold   int
	EnableContentAnalysis bool
	MaxWorkers            int
	BatchSize             int
	Timeout               time.Duration
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
	viper.SetDefault("server.address", ":8083")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "analysis_user")
	viper.SetDefault("database.password", "analysis_password")
	viper.SetDefault("database.name", "analysis_db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Work service defaults
	viper.SetDefault("services.work.url", "http://work-service:8081")
	viper.SetDefault("services.work.works_endpoint", "/api/v1/works")
	viper.SetDefault("services.work.timeout", "10s")
	viper.SetDefault("services.work.retry_count", 3)
	viper.SetDefault("services.work.retry_delay", "100ms")

	// File service defaults
	viper.SetDefault("services.file.url", "http://file-service:8082")
	viper.SetDefault("services.file.files_endpoint", "/api/v1/files")
	viper.SetDefault("services.file.hash_endpoint", "/api/v1/files/by-hash")
	viper.SetDefault("services.file.timeout", "30s")
	viper.SetDefault("services.file.retry_count", 3)
	viper.SetDefault("services.file.retry_delay", "100ms")

	// RabbitMQ defaults
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.exchange", "plagiarism_exchange")
	viper.SetDefault("rabbitmq.routing_key", "work.created")
	viper.SetDefault("rabbitmq.queue_name", "work_created_queue")
	viper.SetDefault("rabbitmq.consumer_tag", "analysis-consumer")
	viper.SetDefault("rabbitmq.prefetch_count", 5)

	// Analysis defaults
	viper.SetDefault("analysis.hash_algorithm", "sha256")
	viper.SetDefault("analysis.similarity_threshold", 100)
	viper.SetDefault("analysis.enable_content_analysis", false)
	viper.SetDefault("analysis.max_workers", 5)
	viper.SetDefault("analysis.batch_size", 10)
	viper.SetDefault("analysis.timeout", "300s")

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
