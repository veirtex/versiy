package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type UsersStore struct {
	redisClient *redis.Client
}

func (u *UsersStore) IncrUser(ctx context.Context, id string, duration time.Duration) (int, error) {
	if _, err := u.redisClient.SetNX(ctx, id, 0, duration).Result(); err != nil {
		return 0, err
	}
	if val, err := u.redisClient.Incr(ctx, id).Result(); err != nil {
		return 0, err
	} else {
		return int(val), nil
	}
}
