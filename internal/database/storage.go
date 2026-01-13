package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Storage struct {
	URL interface {
		Store(ctx context.Context, params URLInsert, secret string) (string, error)
		Get(ctx context.Context, shortCode string) (string, error)
		LastTimeAccessed(ctx context.Context, shortCode string) error
		UpdateClicks(ctx context.Context, shortCode string) error
		CacheResult(ctx context.Context, shortCode, url string, TTL time.Duration) error
		CheckCached(ctx context.Context, shortCode string) (string, error)
	}
	Users interface {
		IncrUser(ctx context.Context, id string, duration time.Duration) (int, error)
	}
}

func NewStorage(conn *pgx.Conn, redis *redis.Client) Storage {
	return Storage{
		URL:   &URLStore{dbConn: conn, redisClient: redis},
		Users: &UsersStore{redisClient: redis},
	}
}
