package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Services ServicesConfig `mapstructure:"services"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Analysis AnalysisConfig `mapstructure:"analysis"`
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

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type ServiceConfig struct {
	URL           string        `mapstructure:"url"`
	WorksEndpoint string        `mapstructure:"works_endpoint"`
	FilesEndpoint string        `mapstructure:"files_endpoint"`
	HashEndpoint  string        `mapstructure:"hash_endpoint"`
	Timeout       time.Duration `mapstructure:"timeout"`
	RetryCount    int           `mapstructure:"retry_count"`
	RetryDelay    time.Duration `mapstructure:"retry_delay"`
}

type ServicesConfig struct {
	Work ServiceConfig `mapstructure:"work"`
	File ServiceConfig `mapstructure:"file"`
}

type RabbitMQConfig struct {
	URL           string `mapstructure:"url"`
	Exchange      string `mapstructure:"exchange"`
	RoutingKey    string `mapstructure:"routing_key"`
	QueueName     string `mapstructure:"queue_name"`
	ConsumerTag   string `mapstructure:"consumer_tag"`
	PrefetchCount int    `mapstructure:"prefetch_count"`
}

type AnalysisConfig struct {
	HashAlgorithm         string        `mapstructure:"hash_algorithm"`
	SimilarityThreshold   int           `mapstructure:"similarity_threshold"`
	EnableContentAnalysis bool          `mapstructure:"enable_content_analysis"`
	MaxWorkers            int           `mapstructure:"max_workers"`
	BatchSize             int           `mapstructure:"batch_size"`
	Timeout               time.Duration `mapstructure:"timeout"`
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

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.address", ":8083")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "analysis_user")
	viper.SetDefault("database.password", "analysis_password")
	viper.SetDefault("database.name", "analysis_db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	viper.SetDefault("services.work.url", "http://work-service:8081")
	viper.SetDefault("services.work.works_endpoint", "/api/v1/works")
	viper.SetDefault("services.work.timeout", "10s")
	viper.SetDefault("services.work.retry_count", 3)
	viper.SetDefault("services.work.retry_delay", "100ms")

	viper.SetDefault("services.file.url", "http://file-service:8082")
	viper.SetDefault("services.file.files_endpoint", "/api/v1/files")
	viper.SetDefault("services.file.hash_endpoint", "/api/v1/files/by-hash")
	viper.SetDefault("services.file.timeout", "30s")
	viper.SetDefault("services.file.retry_count", 3)
	viper.SetDefault("services.file.retry_delay", "100ms")

	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.exchange", "plagiarism_exchange")
	viper.SetDefault("rabbitmq.routing_key", "work.created")
	viper.SetDefault("rabbitmq.queue_name", "work_created_queue")
	viper.SetDefault("rabbitmq.consumer_tag", "analysis-consumer")
	viper.SetDefault("rabbitmq.prefetch_count", 5)

	viper.SetDefault("analysis.hash_algorithm", "sha256")
	viper.SetDefault("analysis.similarity_threshold", 100)
	viper.SetDefault("analysis.enable_content_analysis", false)
	viper.SetDefault("analysis.max_workers", 5)
	viper.SetDefault("analysis.batch_size", 10)
	viper.SetDefault("analysis.timeout", "300s")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.pretty", false)
	viper.SetDefault("logging.no_color", false)

	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allowed_headers", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	viper.SetDefault("cors.exposed_headers", []string{"Link"})
	viper.SetDefault("cors.allow_credentials", true)
	viper.SetDefault("cors.max_age", 300)
}
