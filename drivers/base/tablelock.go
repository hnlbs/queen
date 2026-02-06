// Package base provides common functionality for database drivers.
package base

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/honeynil/queen"
)

// TableLockConfig configures table-based distributed locking.
type TableLockConfig struct {
	CleanupQuery string
	CheckQuery   string
	InsertQuery  string
	ScanFunc     func(*sql.Row) (bool, error)
}

// AcquireTableLock implements distributed locking using a lock table.
// Retries with exponential backoff until lock is acquired or timeout is reached.
func AcquireTableLock(ctx context.Context, db *sql.DB, config TableLockConfig, lockKey, ownerID string, timeout time.Duration) error {
	start := time.Now()
	expiresAt := time.Now().Add(timeout)

	backoff := 50 * time.Millisecond
	maxBackoff := 1 * time.Second

	for {
		_, _ = db.ExecContext(ctx, config.CleanupQuery, lockKey, expiresAt)

		row := db.QueryRowContext(ctx, config.CheckQuery, lockKey)
		hasLock, err := config.ScanFunc(row)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			goto retry
		}

		if !hasLock {
			_, err := db.ExecContext(ctx, config.InsertQuery, lockKey, expiresAt, ownerID)
			if err == nil {
				return nil
			}
		}

	retry:
		if time.Since(start) >= timeout {
			return queen.ErrLockTimeout
		}

		select {
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
