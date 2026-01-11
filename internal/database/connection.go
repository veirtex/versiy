package database

import (
	"context"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5"
)

func NewDBConn(ctx context.Context, addr string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
