// Package ydb provides a YandexDB (YDB) driver for Queen migrations.
package ydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/honeynil/queen"
	"github.com/honeynil/queen/drivers/base"
	"github.com/ydb-platform/ydb-go-sdk/v3"
)

// Driver implements the queen.Driver interface for YDB.
type Driver struct {
	base.Driver
	lockTableName string
	lockKey       string
	ownerID       string
}

// New creates a new YDB driver.
func New(db *sql.DB) (*Driver, error) {
	return NewWithTableName(db, "queen_migrations")
}

// NewWithTableName creates a new YDB driver with a custom table name.
func NewWithTableName(db *sql.DB, tableName string) (*Driver, error) {
	ownerID, err := base.GenerateOwnerID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate lock owner ID: %w", err)
	}

	return &Driver{
		Driver: base.Driver{
			DB:        db,
			TableName: tableName,
			Config: base.Config{
				Placeholder:     base.PlaceholderDollar,
				QuoteIdentifier: base.QuoteBackticks,
				ParseTime:       nil,
			},
		},
		lockTableName: tableName + "_lock",
		lockKey:       "migration_lock",
		ownerID:       ownerID,
	}, nil
}

// Init creates the migrations tracking table and lock table if they don't exist.
func (d *Driver) Init(ctx context.Context) error {
	ctx = ydb.WithQueryMode(ctx, ydb.SchemeQueryMode)

	migrationsQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version       Utf8,
			name          Utf8,
			applied_at    Timestamp,
			checksum      Utf8,
			applied_by    Utf8,
			duration_ms   Int64,
			hostname      Utf8,
			environment   Utf8,
			action        Utf8,
			status        Utf8,
			error_message Utf8,
			PRIMARY KEY (version)
		)
	`, d.Config.QuoteIdentifier(d.TableName))

	if _, err := d.DB.ExecContext(ctx, migrationsQuery); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations := []string{
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN applied_by Utf8`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN duration_ms Int64`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN hostname Utf8`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN environment Utf8`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN action Utf8`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN status Utf8`, d.Config.QuoteIdentifier(d.TableName)),
		fmt.Sprintf(`ALTER TABLE %s ADD COLUMN error_message Utf8`, d.Config.QuoteIdentifier(d.TableName)),
	}

	for _, migration := range migrations {
		_, _ = d.DB.ExecContext(ctx, migration)
	}

	lockQuery := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			lock_key    Utf8,
			acquired_at Timestamp,
			expires_at  Timestamp NOT NULL,
			owner_id    Utf8 NOT NULL,
			PRIMARY KEY (lock_key)
		)
		WITH (
			TTL = Interval("PT10S") ON expires_at
		)
	`, d.Config.QuoteIdentifier(d.lockTableName))

	if _, err := d.DB.ExecContext(ctx, lockQuery); err != nil {
		return fmt.Errorf("failed to create lock table: %w", err)
	}

	return nil
}

// Lock acquires a distributed lock to prevent concurrent migrations.
func (d *Driver) Lock(ctx context.Context, timeout time.Duration) error {
	cfg := base.TableLockConfig{
		CleanupQuery: fmt.Sprintf(
			"DELETE FROM %s WHERE lock_key = $1 AND expires_at < CurrentUtcTimestamp()",
			d.Config.QuoteIdentifier(d.lockTableName),
		),
		CheckQuery: fmt.Sprintf(
			"SELECT 1 FROM %s WHERE lock_key = $1 AND expires_at >= CurrentUtcTimestamp() LIMIT 1",
			d.Config.QuoteIdentifier(d.lockTableName),
		),
		InsertQuery: fmt.Sprintf(
			"INSERT INTO %s (lock_key, acquired_at, expires_at, owner_id) VALUES ($1, CurrentUtcTimestamp(), $2, $3)",
			d.Config.QuoteIdentifier(d.lockTableName),
		),
		ScanFunc: func(row *sql.Row) (bool, error) {
			var exists int
			err := row.Scan(&exists)
			if err != nil && err != sql.ErrNoRows {
				return false, err
			}
			return err != sql.ErrNoRows, nil
		},
	}

	err := base.AcquireTableLock(ctx, d.DB, cfg, d.lockKey, d.ownerID, timeout)
	if errors.Is(err, queen.ErrLockTimeout) {
		return fmt.Errorf("%w: failed to acquire lock '%s' for table '%s'",
			queen.ErrLockTimeout, d.lockKey, d.lockTableName)
	}
	return err
}

// Unlock releases the migration lock.
func (d *Driver) Unlock(ctx context.Context) error {
	unlockQuery := fmt.Sprintf(
		"DELETE FROM %s WHERE lock_key = $1 AND owner_id = $2",
		d.Config.QuoteIdentifier(d.lockTableName),
	)

	_, err := d.DB.ExecContext(ctx, unlockQuery, d.lockKey, d.ownerID)
	if err != nil {
		return fmt.Errorf("failed to release lock '%s' for table '%s': %w",
			d.lockKey, d.TableName, err)
	}

	return nil
}

// Record marks a migration as applied in the database.
func (d *Driver) Record(ctx context.Context, m *queen.Migration, meta *queen.MigrationMetadata) error {
	var appliedBy, hostname, environment, action, status, errorMessage interface{}
	var durationMS interface{}

	if meta != nil {
		if meta.AppliedBy != "" {
			appliedBy = meta.AppliedBy
		}
		if meta.DurationMS > 0 {
			durationMS = meta.DurationMS
		}
		if meta.Hostname != "" {
			hostname = meta.Hostname
		}
		if meta.Environment != "" {
			environment = meta.Environment
		}
		if meta.Action != "" {
			action = meta.Action
		}
		if meta.Status != "" {
			status = meta.Status
		}
		if meta.ErrorMessage != "" {
			errorMessage = meta.ErrorMessage
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (version, name, applied_at, checksum, applied_by, duration_ms, hostname, environment, action, status, error_message)
		VALUES ($1, $2, CurrentUtcTimestamp(), $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		d.Config.QuoteIdentifier(d.TableName),
	)

	_, err := d.DB.ExecContext(ctx, query,
		m.Version, m.Name, m.Checksum(),
		appliedBy, durationMS, hostname, environment, action, status, errorMessage)
	return err
}
