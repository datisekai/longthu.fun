// Package db provides shared database plumbing. Right now it only exposes
// WithTx — the canonical way services run a transactional unit. Story 1.5+
// audit-in-same-tx invariant + Stories 1.7 + 1.8's repeated BeginTx pattern
// converge here.
package db

import (
	"context"
	"database/sql"
	"fmt"
)

// WithTx runs fn inside a transaction. Commits if fn returns nil; rolls back
// on any error (including the panic case). Callers do NOT call Begin /
// Commit / Rollback themselves — that's the whole point.
//
// Example:
//
//	err := db.WithTx(ctx, s.db, func(tx *sql.Tx) error {
//	    q := dbgen.New(tx)
//	    if _, err := q.InsertGroup(...); err != nil {
//	        return err
//	    }
//	    return audit.Record(ctx, tx, audit.Event{...})
//	})
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.WithTx: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db.WithTx: commit: %w", err)
	}
	committed = true
	return nil
}
