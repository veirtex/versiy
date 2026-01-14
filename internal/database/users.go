package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type UsersStore struct {
	RedisClient *redis.Client
}

func (u *UsersStore) IncrUser(ctx context.Context, id string, duration time.Duration) (int, error) {
	_ = u.RedisClient.SetNX(ctx, id, 0, duration)
	if val, err := u.RedisClient.Incr(ctx, id).Result(); err != nil {
		return 0, err
	} else {
		return int(val), nil
	}
}
