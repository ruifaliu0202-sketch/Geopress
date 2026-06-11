package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	conn *sql.DB
}

func Open(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, nil
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.conn == nil {
		return errors.New("database is not configured")
	}
	return db.conn.PingContext(ctx)
}

func (db *DB) SQL() *sql.DB {
	if db == nil {
		return nil
	}
	return db.conn
}
