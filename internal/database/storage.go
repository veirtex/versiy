package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Storage struct {
	URL interface {
		Store(ctx context.Context, params URLInsert, secret string) (string, error)
		Get(ctx context.Context, shortCode string) (string, error)
		LastTimeAccessed(ctx context.Context, shortCode string) error
	}
}

func NewStorage(conn *pgx.Conn) Storage {
	return Storage{
		URL: &URLStore{dbConn: conn},
	}
}
