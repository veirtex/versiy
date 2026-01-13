package database

import (
	"context"

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

func NewRedisConn(ctx context.Context, addr string, psw string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: psw,
		DB:       0,
	})
	return rdb
}
