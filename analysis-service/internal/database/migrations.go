package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Migrator struct {
	migrate *migrate.Migrate
}

func NewMigrator(cfg config.DatabaseConfig) *Migrator {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(fmt.Sprintf("failed to open database: %v", err))
	}

	// Проверка соединения с контекстом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		panic(fmt.Sprintf("failed to ping database: %v", err))
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to create migration driver: %v", err))
	}

	// Получаем путь к миграциям
	migrationPath := "file://migrations"
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		migrationPath = "file://./migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres", driver,
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create migrator: %v", err))
	}

	return &Migrator{migrate: m}
}

func (m *Migrator) Up() error {
	defer func() { _, _ = m.migrate.Close() }()
	if err := m.migrate.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func (m *Migrator) Down() error {
	defer func() { _, _ = m.migrate.Close() }()
	if err := m.migrate.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

func (m *Migrator) Force(version int) error {
	defer func() { _, _ = m.migrate.Close() }()
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version to %d: %w", version, err)
	}
	return nil
}
