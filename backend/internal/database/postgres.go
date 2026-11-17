package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func DBConn(ConnString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), ConnString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = pool.Ping(pingCtx)

	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	return pool, nil
}
