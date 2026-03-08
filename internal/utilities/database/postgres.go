package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/flexer2006/case-person-enrichment-go/internal/utilities"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresProvider interface {
	Pool() *pgxpool.Pool
	Close(ctx context.Context)
	Ping(ctx context.Context) error
	GetDSN() string
}

var ErrInvalidConfiguration = errors.New("invalid database configuration: required fields missing")

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MinConns int
	MaxConns int
}

type PostgresDB struct {
	pool *pgxpool.Pool
	PostgresConfig
	dsn string
}

func (c PostgresConfig) Validate() error {
	if c.Host == "" || c.Port == 0 || c.User == "" || c.Database == "" {
		return ErrInvalidConfiguration
	}
	return nil
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

func NewPostgres(ctx context.Context, config PostgresConfig) (*PostgresDB, error) {
	if err := config.Validate(); err != nil {
		utilities.Error(ctx, "invalid database configuration", zap.Error(err))
		return nil, err
	}

	dsn := config.DSN()
	pool, err := connect(ctx, dsn, config.MinConns, config.MaxConns)
	if err != nil {
		return nil, err
	}

	return &PostgresDB{
		pool:           pool,
		dsn:            dsn,
		PostgresConfig: config,
	}, nil
}

func connect(ctx context.Context, dsn string, minConn, maxConn int) (*pgxpool.Pool, error) {
	utilities.Info(ctx, "connecting to postgres database")

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		utilities.Error(ctx, "failed to parse database configuration", zap.Error(err))
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	applyLimits(cfg, minConn, maxConn)

	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		utilities.Error(ctx, "failed to create connection pool", zap.Error(err))
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		utilities.Error(ctx, "failed to ping database", zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	utilities.Info(ctx, "connected to postgres database")
	return pool, nil
}

func applyLimits(cfg *pgxpool.Config, minConn, maxConn int) {
	to32 := func(n int) int32 {
		switch {
		case n <= 0:
			return 0
		case n > math.MaxInt32:
			return math.MaxInt32
		default:
			return int32(n)
		}
	}

	if minConn > 0 {
		cfg.MinConns = to32(minConn)
	}

	if maxConn > 0 {
		cfg.MaxConns = to32(maxConn)
	}
}

func (db *PostgresDB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *PostgresDB) Close(ctx context.Context) {
	if db.pool != nil {
		utilities.Info(ctx, "closing postgres database connection")
		db.pool.Close()
	}
}

func (db *PostgresDB) Ping(ctx context.Context) error {
	if db.pool == nil {
		return fmt.Errorf("failed to ping database: connection pool is nil")
	}
	return db.pool.Ping(ctx)
}

func parseDSN(dsn string) PostgresConfig {
	if dsn == "" {
		return PostgresConfig{}
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return PostgresConfig{}
	}

	port := 0
	if u.Port() != "" {
		if p, err := strconv.Atoi(u.Port()); err == nil {
			port = p
		}
	}

	user := ""
	password := ""
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
	}

	dbname := strings.TrimPrefix(u.Path, "/")
	sslmode := u.Query().Get("sslmode")

	return PostgresConfig{
		Host:     u.Hostname(),
		Port:     port,
		User:     user,
		Password: password,
		Database: dbname,
		SSLMode:  sslmode,
	}
}

func NewPostgresWithDSN(ctx context.Context, dsn string, minConn, maxConn int) (*PostgresDB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("dsn must not be empty")
	}

	pool, err := connect(ctx, dsn, minConn, maxConn)
	if err != nil {
		return nil, err
	}

	config := parseDSN(dsn)
	config.MinConns = minConn
	config.MaxConns = maxConn

	return &PostgresDB{
		pool:           pool,
		dsn:            dsn,
		PostgresConfig: config,
	}, nil
}

func (db *PostgresDB) GetDSN() string {
	return db.dsn
}

func (db *PostgresDB) Config() PostgresConfig {
	return db.PostgresConfig
}
