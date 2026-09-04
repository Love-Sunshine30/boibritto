package postgres

import (
	"context"
	"database/sql"
)

// this interface unifies *sql.DB connection pool and *sql.Tx transaction
type Querier interface {
	QueryContext(ctx context.Context, stmt string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, stmt string, args ...any) *sql.Row
	ExecContext(ctx context.Context, stmt string, args ...any) (sql.Result, error)
}

// compile time check if *sql.DB and *sql.Tx satisfies Querier
var (
	_        Querier = (*sql.DB)(nil)
	_Querier         = (*sql.Tx)(nil)
)

// Withtx runs a transaction inside a *sql.Tx, commits on success, otherwise rollbacks
func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err = fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
