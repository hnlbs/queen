// Package postgres provides a PostgreSQL driver for Queen migrations.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/honeynil/queen"
	"github.com/honeynil/queen/drivers/base"
)

// Driver implements the queen.Driver interface for PostgreSQL.
type Driver struct {
	base.Driver
	lockID   int64
	lockConn *sql.Conn
}

// New creates a new PostgreSQL driver.
func New(db *sql.DB) *Driver {
	return NewWithTableName(db, "queen_migrations")
}

// NewWithTableName creates a new PostgreSQL driver with a custom table name.
func NewWithTableName(db *sql.DB, tableName string) *Driver {
	return &Driver{
		Driver: base.Driver{
			DB:        db,
			TableName: tableName,
			Config: base.Config{
				Placeholder:     base.PlaceholderDollar,
				QuoteIdentifier: base.QuoteDoubleQuotes,
				ParseTime:       nil,
			},
		},
		lockID: hashTableName(tableName),
	}
}

// Init creates the migrations tracking table if it doesn't exist.
func (d *Driver) Init(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64) NOT NULL,
			applied_by VARCHAR(255),
			duration_ms BIGINT,
			hostname VARCHAR(255),
			environment VARCHAR(50),
			action VARCHAR(20) DEFAULT 'apply',
			status VARCHAR(20) DEFAULT 'success',
			error_message TEXT
		)
	`, d.Config.QuoteIdentifier(d.TableName))

	if _, err := d.DB.ExecContext(ctx, query); err != nil {
		return err
	}

	migrations := []string{
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS applied_by VARCHAR(255)`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS duration_ms BIGINT`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS hostname VARCHAR(255)`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS environment VARCHAR(50)`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS action VARCHAR(20) DEFAULT 'apply'`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'success'`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS error_message TEXT`, d.Config.QuoteIdentifier(d.TableName)),
	}

	for _, migration := range migrations {
		if _, err := d.DB.ExecContext(ctx, migration); err != nil {
			continue
		}
	}

	return nil
}

// Lock acquires an advisory lock to prevent concurrent migrations.
func (d *Driver) Lock(ctx context.Context, timeout time.Duration) error {
	conn, err := d.DB.Conn(ctx)
	if err != nil {
		return err
	}

	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = conn.ExecContext(lockCtx, "SELECT pg_advisory_lock($1)", d.lockID)
	if err != nil {
		_ = conn.Close()
		if lockCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: failed to acquire advisory lock '%d' for table '%s'",
				queen.ErrLockTimeout, d.lockID, d.TableName)
		}
		return err
	}

	d.lockConn = conn
	return nil
}

// Unlock releases the advisory lock.
func (d *Driver) Unlock(ctx context.Context) error {
	if d.lockConn == nil {
		return nil
	}
	defer func() {
		_ = d.lockConn.Close()
		d.lockConn = nil
	}()

	_, err := d.lockConn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", d.lockID)
	if err != nil {
		return fmt.Errorf("failed to release advisory lock '%d' for table '%s': %w",
			d.lockID, d.TableName, err)
	}
	return nil
}

// hashTableName creates a unique int64 hash from the table name for advisory locks.
func hashTableName(name string) int64 {
	var hash int64
	for i, c := range name {
		hash = hash*31 + int64(c) + int64(i)
	}
	return hash
}
