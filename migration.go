package queen

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/honeynil/queen/internal/checksum"
)

// MigrationFunc is a function that executes a migration using a transaction.
// It receives a context and a transaction, and should return an error if the migration fails.
type MigrationFunc func(ctx context.Context, tx *sql.Tx) error

// Migration represents a single database migration.
//
// A migration can be defined using SQL strings (UpSQL/DownSQL) or
// Go functions (UpFunc/DownFunc). You can also mix both approaches
// in the same migration.
//
// # Version and Name
//
// Version must be unique across all migrations. Queen uses natural sorting,
// so "1", "2", "10" sort correctly. You can use prefixes for organization:
// "users_001", "posts_001".
//
// Name should be a human-readable description like "create_users" or
// "add_email_index".
//
// # SQL Migrations
//
// For simple schema changes, use UpSQL and DownSQL:
//
//	queen.M{
//	    Version: "001",
//	    Name:    "create_users",
//	    UpSQL:   "CREATE TABLE users (id INT)",
//	    DownSQL: "DROP TABLE users",
//	}
//
// # Go Function Migrations
//
// For complex logic that can't be expressed in SQL, use UpFunc and DownFunc:
//
//	queen.M{
//	    Version:        "002",
//	    Name:           "migrate_data",
//	    ManualChecksum: "v1",
//	    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
//	        // Your migration logic here
//	        return nil
//	    },
//	}
//
// IMPORTANT: When using UpFunc/DownFunc, always set ManualChecksum to track
// changes. Update it whenever you modify the function (e.g., "v1" -> "v2").
//
// # Checksums
//
// Queen automatically calculates checksums for SQL migrations. For Go function
// migrations, you must provide ManualChecksum. This detects when applied
// migrations have been modified, which can indicate a problem.
type Migration struct {
	// Version uniquely identifies this migration.
	// Examples: "001", "002", "user_001", "v1.0.0"
	Version string

	// Name describes what this migration does.
	// Examples: "create_users", "add_email_index"
	Name string

	// UpSQL applies the migration using SQL.
	// Leave empty when using UpFunc.
	UpSQL string

	// DownSQL rolls back the migration using SQL.
	// Optional but recommended for safe rollbacks.
	DownSQL string

	// UpFunc applies the migration using Go code.
	// Use for complex logic that can't be expressed in SQL.
	UpFunc MigrationFunc

	// DownFunc rolls back the migration using Go code.
	// Optional but recommended for safe rollbacks.
	DownFunc MigrationFunc

	// ManualChecksum tracks changes to function migrations.
	// Required when using UpFunc/DownFunc for validation.
	// Examples: "v1", "v2", "normalize-emails-v1"
	// Update this whenever you modify the function.
	ManualChecksum string

	// IsolationLevel sets the transaction isolation level for this migration.
	// Default: sql.LevelDefault (uses Config.IsolationLevel or database default)
	//
	// This overrides the global Config.IsolationLevel for this specific migration.
	//
	// Use cases:
	//   - Critical migrations requiring SERIALIZABLE isolation
	//   - Bulk data migrations that can use READ COMMITTED for better performance
	//   - Preventing race conditions during schema changes
	//
	// Example:
	//   queen.M{
	//       Version: "003",
	//       Name:    "critical_update",
	//       IsolationLevel: sql.LevelSerializable,
	//       UpSQL:   "UPDATE users SET ...",
	//   }
	IsolationLevel sql.IsolationLevel

	// Lazy-loaded checksum cache. sync.Once pointer prevents copylocks warning
	// when Migration is passed by value.
	checksumOnce *sync.Once
	checksum     string
}

// M is a convenient alias for Migration, used in registration:
//
//	q.Add(queen.M{
//	    Version: "001",
//	    Name: "create_users",
//	    UpSQL: "CREATE TABLE users...",
//	    DownSQL: "DROP TABLE users",
//	})
type M = Migration

// Validate ensures Version, Name, and at least one Up method are defined.
func (m *Migration) Validate() error {
	if m.Version == "" || strings.Contains(m.Version, " ") || !IsValidMigrationName(m.Version) {
		return ErrInvalidMigration
	}

	if len(m.Name) > 63 {
		return ErrNameTooLong
	}

	if m.Name == "" || !IsValidMigrationName(m.Name) {
		return ErrInvalidMigrationName
	}

	// Must have at least one Up method
	if m.UpSQL == "" && m.UpFunc == nil {
		return ErrInvalidMigration
	}

	return nil
}

// noChecksumMarker is used when checksum validation cannot be performed for
// Go function migrations without an explicit ManualChecksum.
//
// # Why Go Functions Can't Be Hashed
//
// Unlike SQL strings, Go functions are compiled runtime values. The function's
// bytecode is not accessible for inspection or hashing. This means we cannot
// automatically detect if a Go migration function has been modified after
// being applied to the database.
//
// # Requirement: Set ManualChecksum
//
// For Go function migrations (UpFunc/DownFunc), you MUST provide a ManualChecksum
// to enable integrity validation. Update the checksum whenever you modify the
// function logic to detect unintended changes.
//
// # Example Usage
//
//	queen.M{
//	    Version:        "002",
//	    Name:           "migrate_user_data",
//	    ManualChecksum: "v1",  // Required for Go functions
//	    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
//	        // Your migration logic here
//	        return nil
//	    },
//	}
//
// If you modify the function, update the checksum:
//
//	ManualChecksum: "v2"  // Updated after changing the function
//
// Without ManualChecksum, the migration will use this marker and skip
// checksum validation, which may hide accidental modifications.
const noChecksumMarker = "no-checksum-go-func"

// Checksum returns a hash for validation.
// Uses ManualChecksum if set, calculates from SQL otherwise, or returns a marker for Go functions.
func (m *Migration) Checksum() string {
	if m.checksumOnce == nil {
		m.checksumOnce = &sync.Once{}
	}

	m.checksumOnce.Do(func() {
		// If manual checksum is provided, use it
		if m.ManualChecksum != "" {
			m.checksum = m.ManualChecksum
			return
		}

		// For SQL migrations, calculate checksum
		if m.UpSQL != "" || m.DownSQL != "" {
			m.checksum = checksum.Calculate(m.UpSQL, m.DownSQL)
			return
		}

		// For Go functions without manual checksum, use special marker
		m.checksum = noChecksumMarker
	})

	return m.checksum
}

// HasRollback checks if DownSQL or DownFunc is defined.
func (m *Migration) HasRollback() bool {
	return m.DownSQL != "" || m.DownFunc != nil
}

// IsDestructive checks DownSQL for destructive keywords: DROP TABLE, DROP DATABASE, TRUNCATE, etc.
// Up migrations are assumed constructive and not checked.
func (m *Migration) IsDestructive() bool {
	if m.DownSQL == "" {
		return false
	}

	sql := strings.ToUpper(m.DownSQL)

	destructiveKeywords := []string{
		"DROP TABLE",
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
	}

	for _, keyword := range destructiveKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}

	return false
}

// executeUp runs UpFunc or UpSQL within the transaction.
func (m *Migration) executeUp(ctx context.Context, tx *sql.Tx) error {
	if m.UpFunc != nil {
		return m.UpFunc(ctx, tx)
	}

	if m.UpSQL != "" {
		_, err := tx.ExecContext(ctx, m.UpSQL)
		return err
	}

	return ErrInvalidMigration
}

// executeDown runs DownFunc or DownSQL within the transaction.
func (m *Migration) executeDown(ctx context.Context, tx *sql.Tx) error {
	if m.DownFunc != nil {
		return m.DownFunc(ctx, tx)
	}

	if m.DownSQL != "" {
		_, err := tx.ExecContext(ctx, m.DownSQL)
		return err
	}

	return ErrInvalidMigration
}
