package database

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func NewDBConn(ctx context.Context, addr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(addr)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 10
	config.MinConns = 5
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	config.ConnConfig.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewRedisConn(ctx context.Context, addr string, psw string, dialTimeout, readTimeout, writeTimeout, poolTimeout time.Duration) *redis.Client {
	options := &redis.Options{
		Addr:                  addr,
		Password:              psw,
		DB:                    0,
		DialTimeout:           dialTimeout,
		ReadTimeout:           readTimeout,
		WriteTimeout:          writeTimeout,
		PoolTimeout:           poolTimeout,
		PoolSize:              10,
		MinIdleConns:          5,
		MaxIdleConns:          20,
		MaxActiveConns:        30,
		ConnMaxIdleTime:       30 * time.Minute,
		ConnMaxLifetime:       0,
		ContextTimeoutEnabled: true,
	}

	if strings.HasPrefix(addr, "rediss://") {
		options.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(options)
	return rdb
}
