package database

import (
	"context"
	"time"
	"versiy/internal/util"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type URLStore struct {
	dbConn      *pgxpool.Pool
	redisClient *redis.Client
}

type URLInsert struct {
	OriginalURL string
	ExpiresAt   *time.Time
}

func (us *URLStore) Store(ctx context.Context, params URLInsert, secret string) (string, error) {
	var existingShortCode string
	var existingID int64

	err := us.dbConn.QueryRow(ctx,
		`SELECT short_code, id FROM links
		 WHERE original_url = $1 AND expires_at > $2
		 ORDER BY created_at DESC
		 LIMIT 1`,
		params.OriginalURL,
		time.Now(),
	).Scan(&existingShortCode, &existingID)

	if err == nil {
		return existingShortCode, nil
	}

	if err != pgx.ErrNoRows {
		return "", err
	}

	tx, err := us.dbConn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO links (original_url, expires_at)
		 VALUES ($1, $2)
		 RETURNING id`,
		params.OriginalURL,
		params.ExpiresAt,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	shortCode := util.GenerateShortCode(secret, id)

	_, err = tx.Exec(ctx,
		`UPDATE links
		 SET short_code = $1
		 WHERE id = $2`,
		shortCode,
		id,
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return shortCode, nil
}

func (us *URLStore) Get(ctx context.Context, shortCode string) (string, error) {
	var url string
	err := us.dbConn.QueryRow(ctx,
		"SELECT original_url FROM links WHERE short_code = $1 AND expires_at >= $2",
		shortCode,
		time.Now(),
	).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (us *URLStore) CacheResult(ctx context.Context, shortCode, url string, TTL time.Duration) error {
	status := us.redisClient.Set(ctx, shortCode, url, TTL)
	if status.Err() != nil {
		return status.Err()
	}
	return nil
}

func (us *URLStore) CheckCached(ctx context.Context, shortCode string) (string, error) {
	value, err := us.redisClient.Get(ctx, shortCode).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

func (us *URLStore) LastTimeAccessed(ctx context.Context, shortCode string) error {
	_, err := us.dbConn.Exec(ctx,
		"UPDATE links SET last_time_accessed = $1 WHERE short_code = $2",
		time.Now(),
		shortCode,
	)
	return err
}

func (us *URLStore) getLinkID(ctx context.Context, shortCode string) (int, error) {
	var id int
	query := "SELECT id FROM links WHERE short_code = $1"
	if err := us.dbConn.QueryRow(ctx, query, shortCode).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (us *URLStore) UpdateClicks(ctx context.Context, shortCode string) error {
	id, err := us.getLinkID(ctx, shortCode)
	if err != nil {
		return err
	}
	query := "INSERT INTO links_clicks(link_id, clicks) VALUES ($1, $2) ON CONFLICT (link_id) DO UPDATE SET clicks = links_clicks.clicks + 1"
	if _, err := us.dbConn.Exec(ctx, query, id, 1); err != nil {
		return err
	}
	return nil
}
