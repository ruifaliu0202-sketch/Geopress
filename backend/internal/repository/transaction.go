package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type TxRunner struct {
	db *sql.DB
}

func NewTxRunner(db *sql.DB) TxRunner {
	return TxRunner{db: db}
}

func (runner TxRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, q Queryer) error) error {
	if runner.db == nil {
		return errors.New("repository transaction runner requires database")
	}

	tx, err := runner.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 多表写入和审计/权益变更必须在同一事务内完成，调用方只暴露 Queryer 防止越过事务。
	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
