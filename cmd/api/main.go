package main

import (
	"context"
	"sync"
	"time"
	"versiy/env"
	"versiy/internal/database"
)

func main() {
	ctx := context.Background()
	cfg := config{
		addr: env.GetString("ADDR", ":3000"),
		postgresConfig: postgreSQLConfig{
			addr: env.GetString("POSTGRES_ADDR", ""),
		},
		redisConfig: redisConfig{
			addr:         env.GetString("REDIS_ADDR", ""),
			pswd:         env.GetString("REDIS_PASSWORD", ""),
			defualtTTL:   time.Duration(time.Hour * 24),
			dialTimeout:  10 * time.Second,
			readTimeout:  5 * time.Second,
			writeTimeout: 5 * time.Second,
			poolTimeout:  5 * time.Second,
		},
		secret:      env.GetString("SECRET", "so secret"),
		defaultLink: env.GetString("DEFAULT_DOMAIN", ""),
		rateLimiting: rateLimitConfig{
			size:     10,
			duration: time.Duration(time.Second * 15),
		},
	}

	pool, err := database.NewDBConn(ctx, cfg.postgresConfig.addr)
	if err != nil {
		panic(err)
	}

	redisClient := database.NewRedisConn(ctx, cfg.redisConfig.addr, cfg.redisConfig.pswd, cfg.redisConfig.dialTimeout, cfg.redisConfig.readTimeout, cfg.redisConfig.writeTimeout, cfg.redisConfig.poolTimeout)

	defer pool.Close()
	defer redisClient.Close()

	app := application{
		cfg:   cfg,
		store: database.NewStorage(pool, redisClient),
		env:   env.GetString("ENVIRONMENT", "development"),
		mut:   &sync.Mutex{},
	}

	r := app.mount()
	app.run(r)
}
