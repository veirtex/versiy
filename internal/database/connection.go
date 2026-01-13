package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func NewDBConn(ctx context.Context, addr string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewRedisConn(ctx context.Context, addr string, psw string, dialTimeout, readTimeout, writeTimeout, poolTimeout time.Duration) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     psw,
		DB:           0,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolTimeout:  poolTimeout,
	})
	return rdb
}
