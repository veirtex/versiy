package database

import (
	"context"
	"time"
	"versiy/internal/util"

	"github.com/jackc/pgx/v5"
)

type URLStore struct {
	dbConn *pgx.Conn
}

type URLInsert struct {
	OriginalURL string
	ExpiresAt   *time.Time
}

func (us *URLStore) Store(ctx context.Context, params URLInsert, secret string) (string, error) {
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

func (us *URLStore) LastTimeAccessed(ctx context.Context, shortCode string) error {
	_, err := us.dbConn.Exec(ctx,
		"UPDATE links SET last_time_accessed = $1 WHERE short_code = $2",
		time.Now(),
		shortCode,
	)
	return err
}
