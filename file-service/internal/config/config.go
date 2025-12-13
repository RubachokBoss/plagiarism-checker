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
	Storage  StorageConfig  `mapstructure:"storage"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	Hash     HashConfig     `mapstructure:"hash"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type ServerConfig struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	MaxUploadSize   int64         `mapstructure:"max_upload_size"`
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

type StorageConfig struct {
	Provider   string `mapstructure:"provider"`
	BucketName string `mapstructure:"bucket_name"`
	Region     string `mapstructure:"region"`
}

type MinIOConfig struct {
	Endpoint  string        `mapstructure:"endpoint"`
	AccessKey string        `mapstructure:"access_key"`
	SecretKey string        `mapstructure:"secret_key"`
	UseSSL    bool          `mapstructure:"use_ssl"`
	Timeout   time.Duration `mapstructure:"timeout"`
}

type HashConfig struct {
	Algorithm string `mapstructure:"algorithm"`
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
	viper.SetDefault("server.address", ":8082")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.shutdown_timeout", "10s")
	viper.SetDefault("server.max_upload_size", 104857600) // 100MB

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "file_user")
	viper.SetDefault("database.password", "file_password")
	viper.SetDefault("database.name", "file_db")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	viper.SetDefault("storage.provider", "minio")
	viper.SetDefault("storage.bucket_name", "plagiarism-files")
	viper.SetDefault("storage.region", "us-east-1")

	viper.SetDefault("minio.endpoint", "minio:9000")
	viper.SetDefault("minio.access_key", "minioadmin")
	viper.SetDefault("minio.secret_key", "minioadmin")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.timeout", "30s")

	viper.SetDefault("hash.algorithm", "sha256")

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
