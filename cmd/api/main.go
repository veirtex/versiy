package main

import (
	"context"
	"time"
	"versiy/env"
	"versiy/internal/database"
)

func main() {
	ctx := context.Background()
	cfg := config{
		addr: env.GetString("ADDR", ":3000"),
		postgresConfig: postgreSQLConfig{
			addr:           env.GetString("POSTGRES_ADDR", ""),
			max_idle_conns: 10, // not used yet
		},
		redisConfig: redisConfig{
			addr:       env.GetString("REDIS_ADDR", ""),
			pswd:       env.GetString("REDIS_PASSWORD", ""),
			defualtTTL: time.Duration(time.Hour * 24),
		},
		secret:      env.GetString("SECRET", "so secret"),
		defaultLink: env.GetString("DEFAULT_DOMAIN", ""),
	}

	conn, err := database.NewDBConn(ctx, cfg.postgresConfig.addr)
	if err != nil {
		panic(err)
	}

	redisClient := database.NewRedisConn(ctx, cfg.redisConfig.addr, cfg.redisConfig.pswd)

	defer conn.Close(ctx)

	app := application{
		cfg:   cfg,
		store: database.NewStorage(conn, redisClient),
		env:   env.GetString("ENVIRONMENT", "development"),
	}

	r := app.mount()
	app.run(r)
}
