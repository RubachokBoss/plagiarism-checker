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
	URL             string        `mapstructure:"url"`
	UploadEndpoint  string        `mapstructure:"upload_endpoint"`
	ReportsEndpoint string        `mapstructure:"reports_endpoint"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RetryCount      int           `mapstructure:"retry_count"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
}

type ServicesConfig struct {
	File     ServiceConfig `mapstructure:"file"`
	Analysis ServiceConfig `mapstructure:"analysis"`
}

type RabbitMQConfig struct {
	URL        string `mapstructure:"url"`
	Exchange   string `mapstructure:"exchange"`
	RoutingKey string `mapstructure:"routing_key"`
	QueueName  string `mapstructure:"queue_name"`
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
	viper.SetDefault("server.address", ":8081")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "10s")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "work_user")
	viper.SetDefault("database.password", "work_password")
	viper.SetDefault("database.name", "work_db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	viper.SetDefault("services.file.url", "http://file-service:8082")
	viper.SetDefault("services.file.upload_endpoint", "/api/v1/files/upload")
	viper.SetDefault("services.file.timeout", "30s")
	viper.SetDefault("services.file.retry_count", 3)
	viper.SetDefault("services.file.retry_delay", "100ms")

	viper.SetDefault("services.analysis.url", "http://analysis-service:8083")
	viper.SetDefault("services.analysis.reports_endpoint", "/api/v1/reports/work")
	viper.SetDefault("services.analysis.timeout", "10s")
	viper.SetDefault("services.analysis.retry_count", 3)
	viper.SetDefault("services.analysis.retry_delay", "100ms")

	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.exchange", "plagiarism_exchange")
	viper.SetDefault("rabbitmq.routing_key", "work.created")
	viper.SetDefault("rabbitmq.queue_name", "work_created_queue")

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
