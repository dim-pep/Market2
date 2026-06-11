package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dim-pep/Market2/spotservice/config"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/prometheus/client_golang/prometheus"
)

type postgresMarketsRepo struct {
	db *sql.DB
}

func NewPostgresMarketsRepo(cfg *config.Config) (*postgresMarketsRepo, error) {
	var db *sql.DB
	operation := func() error {
		var err error
		db, err = openDB(cfg)
		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 5 * time.Second
	bo.MaxElapsedTime = 15 * time.Second
	bo.RandomizationFactor = 0.5

	err := backoff.Retry(operation, bo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres after retries: %w", err)
	}

	return &postgresMarketsRepo{db: db}, nil
}

func openDB(cfg *config.Config) (*sql.DB, error) {
	conn, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.Db.User, cfg.Db.Password, cfg.Db.Host, cfg.Db.Port, cfg.Db.DBName, cfg.Db.SSLMode))

	if err != nil {
		return nil, fmt.Errorf("failed to create a connection to db: %w", err)
	}

	conn.SetMaxOpenConns(cfg.Db.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.Db.MaxIdleConns)

	maxLifetime, err := time.ParseDuration(cfg.Db.MaxLifetime)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to parse max_lifetime [%s] from cfg: %w", cfg.Db.MaxLifetime, err)
	}

	conn.SetConnMaxLifetime(maxLifetime)

	maxIdleTime, err := time.ParseDuration(cfg.Db.MaxIdleTime)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to parse max_idle_time [%s] from cfg: %w", cfg.Db.MaxIdleTime, err)
	}

	conn.SetConnMaxIdleTime(maxIdleTime)

	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	err = conn.PingContext(ctx)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	collector := sqlstats.NewStatsCollector(cfg.Db.DBName, conn)
	prometheus.MustRegister(collector)

	return conn, nil
}

func (pr *postgresMarketsRepo) Close() error {
	if pr == nil || pr.db == nil {
		return nil
	}

	return pr.db.Close()
}

func (pr *postgresMarketsRepo) CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return pr.db.PingContext(ctx)
}

func InitTables(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return InitTablesWithContext(ctx, db)
}

func InitTablesWithContext(ctx context.Context, db *sql.DB) error {
	if err := createTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := createIndexes(ctx, db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}

func createTable(ctx context.Context, db *sql.DB) error {
	marketsTable := `
		CREATE TABLE IF NOT EXISTS markets (
			id              BIGINT      PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
			symbol          TEXT        NOT NULL UNIQUE,
			enabled         BOOLEAN     NOT NULL DEFAULT TRUE,
			allowed_roles   TEXT[],
			deleted_at      TIMESTAMPTZ DEFAULT NULL
		);
	`

	if _, err := db.ExecContext(ctx, marketsTable); err != nil {
		return fmt.Errorf("failed to create markets table: %w", err)
	}

	return nil
}

func createIndexes(ctx context.Context, db *sql.DB) error {
	activeMarketsIndex := `
		CREATE INDEX IF NOT EXISTS idx_markets_active 
		ON markets (id)
		WHERE enabled = true AND deleted_at IS NULL;
	`

	if _, err := db.ExecContext(ctx, activeMarketsIndex); err != nil {
		return fmt.Errorf("failed to create idx_markets_active index: %w", err)
	}

	return nil
}
