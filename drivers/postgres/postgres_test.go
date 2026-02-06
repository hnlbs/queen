package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/honeynil/queen"
	"github.com/honeynil/queen/drivers/base"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestQuoteIdentifier tests the identifier quoting function.
func TestQuoteIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple table name",
			input:    "users",
			expected: `"users"`,
		},
		{
			name:     "table name with double quote",
			input:    `my"table`,
			expected: `"my""table"`,
		},
		{
			name:     "table name with multiple quotes",
			input:    `my"ta"ble`,
			expected: `"my""ta""ble"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "table name with spaces",
			input:    "my table",
			expected: `"my table"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base.QuoteDoubleQuotes(tt.input)
			if result != tt.expected {
				t.Errorf("quoteIdentifier(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDriverCreation tests driver creation functions.
func TestDriverCreation(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a mock database connection", err)
	}
	_ = db.Close()

	t.Run("New creates driver with default table name", func(t *testing.T) {
		t.Parallel()
		driver := New(db)
		if driver.DB != db {
			t.Error("driver.db should be set")
		}
		if driver.TableName != "queen_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "queen_migrations")
		}
	})

	t.Run("NewWithTableName creates driver with custom table name", func(t *testing.T) {
		t.Parallel()
		driver := NewWithTableName(db, "custom_migrations")
		if driver.DB != db {
			t.Error("driver.db should be set")
		}
		if driver.TableName != "custom_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "custom_migrations")
		}
	})
}

func TestInvalidMigrations(t *testing.T) {
	t.Parallel()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a mock database connection", err)
	}
	_ = db.Close()

	driver := New(db)
	q := queen.New(driver)

	// Test empty version
	t.Run("EmptyVersion", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "",
			Name:    "valid_name",
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("Add() with empty version succeeded; want error")
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "002",
			Name:    "",
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("Add() with empty name succeeded; want error")
		}
	})

	// Test version with spaces
	t.Run("VersionWithSpaces", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "003 with spaces",
			Name:    "valid_name",
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("Add() with spaces succeeded; want error")
		}
	})

	// Test migration name > 63 characters
	t.Run("LongMigrationName", func(t *testing.T) {
		longName := strings.Repeat("a", 64) // 64 characters
		err := q.Add(queen.M{
			Version: "004",
			Name:    longName,
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	// Test special characters in version
	t.Run("SpecialCharsInVersion", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "005@№;%!4special",
			Name:    "valid_name",
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("Add() with special characters in version succeeded; want error")
		}
	})

	// Test special characters in name
	t.Run("SpecialCharsInName", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "006",
			Name:    "%",
			UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
			`,
			DownSQL: `DROP TABLE test_users`,
		})
		if err == nil {
			t.Error("Add() with special characters in name succeeded; want error")
		}
	})

	// Test duplicate versions
	t.Run("DuplicateVersions", func(t *testing.T) {
		err := q.Add(queen.M{
			Version: "007",
			Name:    "first",
			UpSQL:   "CREATE TABLE dummy1 ()",
			DownSQL: "DROP TABLE dummy1",
		})
		if err != nil {
			t.Fatalf("first Add() failed: %v", err)
		}

		err = q.Add(queen.M{
			Version: "007",
			Name:    "second",
			UpSQL:   "CREATE TABLE dummy2 ()",
			DownSQL: "DROP TABLE dummy2",
		})
		if err == nil {
			t.Error("Add() with duplicate version succeeded; want error")
		}
	})
}

// Note: Integration tests that require a real PostgreSQL database are in PostgreSQL_integration_test.go
// Run with: go test -tags=integration -v

// setupTestDB creates a test database connection.
// This requires PostgreSQL to be running. Tests will be skipped if PostgreSQL is not available.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Connect to PostgreSQL (using port 5432 to avoid conflicts)
	dsn := os.Getenv("POSTGRESQL_TEST_DSN")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skip("PostgreSQL not available:", err)
	}

	// Cleanup function
	cleanup := func() {
		var errs []error

		tables := []string{"queen_migrations", "test_users", "test_posts"}

		for _, table := range tables {
			if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
				errs = append(errs, fmt.Errorf("failed to drop table %s: %w", table, err))
			}
		}
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close db: %w", err))
		}

		if len(errs) > 0 {
			t.Fatalf("cleanup failed with errors: %v", errs)
		}
	}

	return db, cleanup
}

func TestIntegrationInit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	err := driver.Init(ctx)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.QueryRowContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'queen_migrations';").Scan(&tableName)
	if err != nil {
		t.Fatalf("migrations table was not created: %v", err)
	}

	// Init should be idempotent
	err = driver.Init(ctx)
	if err != nil {
		t.Fatalf("second Init() failed: %v", err)
	}
}

func TestRecordAndGetApplied(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Initially should have no migrations
	applied, err := driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 migrations, got %d", len(applied))
	}

	// Record a migration
	m1 := &queen.Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL: `CREATE TABLE test_users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		)`,
	}
	if err := driver.Record(ctx, m1, nil); err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	// Should now have 1 migration
	applied, err = driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(applied))
	}
	if applied[0].Version != "001" {
		t.Errorf("version = %q; want %q", applied[0].Version, "001")
	}
	if applied[0].Name != "create_users" {
		t.Errorf("name = %q; want %q", applied[0].Name, "create_users")
	}

	// Record another migration
	m2 := &queen.Migration{
		Version: "002",
		Name:    "create_posts",
		UpSQL: `CREATE TABLE test_posts (
			id UUID DEFAULT gen_random_uuid(),
		)`,
	}
	if err := driver.Record(ctx, m2, nil); err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	// Should now have 2 migrations in order
	applied, err = driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}
	if len(applied) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(applied))
	}
	// Should be sorted by applied_at
	if applied[0].Version != "001" {
		t.Errorf("first version = %q; want %q", applied[0].Version, "001")
	}
	if applied[1].Version != "002" {
		t.Errorf("second version = %q; want %q", applied[1].Version, "002")
	}
}

func TestIntegrationRemove(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	// Init and record a migration
	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	m := &queen.Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL: `CREATE TABLE test_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		)`,
	}
	if err := driver.Record(ctx, m, nil); err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	// Verify it was recorded
	applied, _ := driver.GetApplied(ctx)
	if len(applied) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(applied))
	}

	// Remove the migration
	if err := driver.Remove(ctx, "001"); err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Should now be empty
	applied, err := driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 migrations after removal, got %d", len(applied))
	}
}

func TestIntegrationLocking(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Acquire lock
	err := driver.Lock(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}

	// Try to acquire the same lock from another driver instance (should fail)
	db2, err := sql.Open("pgx", os.Getenv("POSTGRESQL_TEST_DSN"))
	if err != nil {
		t.Fatalf("failed to open second connection: %v", err)
	}
	defer func() { _ = db2.Close() }()

	driver2 := New(db2)
	err = driver2.Lock(ctx, 1*time.Second)

	if !errors.Is(err, queen.ErrLockTimeout) {
		t.Errorf("expected ErrLockTimeout, got %v", err)
	}

	// Release lock
	if err := driver.Unlock(ctx); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	// Now the second driver should be able to acquire the lock
	err = driver2.Lock(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("second Lock() failed after unlock: %v", err)
	}

	// Clean up
	_ = driver2.Unlock(ctx)
}

func TestIntegrationExec(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test successful transaction
	err := driver.Exec(ctx, sql.LevelDefault, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				name VARCHAR(255) NOT NULL
			)
		`)
		return err
	})
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	// Verify table was created
	var tableName string
	err = db.QueryRowContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'test_users';").Scan(&tableName)
	if err != nil {
		t.Fatalf("table was not created: %v", err)
	}

	err = driver.Exec(ctx, sql.LevelDefault, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test_users (name) VALUES ('Alice')")
		if err != nil {
			return err
		}
		// Return error to trigger rollback
		return sql.ErrTxDone
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIntegrationFullMigrationCycle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	q := queen.New(driver)

	ctx := context.Background()

	// Add migrations
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE test_users (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				email VARCHAR(255) NOT NULL
			)
		`,
		DownSQL: `DROP TABLE test_users`,
	})

	q.MustAdd(queen.M{
		Version: "002",
		Name:    "create_posts",
		UpSQL: `
			CREATE TABLE test_posts (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id UUID NOT NULL,
				title VARCHAR(255),
				FOREIGN KEY (user_id) REFERENCES test_users(id) ON DELETE CASCADE
			)
		`,
		DownSQL: `DROP TABLE test_posts`,
	})

	// Apply all migrations
	if err := q.Up(ctx); err != nil {
		t.Fatalf("Up() failed: %v", err)
	}

	// Verify tables exist
	var tableCount uint64
	err := db.QueryRowContext(ctx,
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('test_users', 'test_posts');").Scan(&tableCount)
	if err != nil {
		t.Fatalf("failed to check tables: %v", err)
	}
	if tableCount != 2 {
		t.Errorf("expected 2 tables, got %d", tableCount)
	}

	// Check status
	statuses, err := q.Status(ctx)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(statuses))
	}
	for _, s := range statuses {
		if s.Status != queen.StatusApplied {
			t.Errorf("migration %s status = %s; want applied", s.Version, s.Status)
		}
	}

	// Rollback all migrations
	if err := q.Reset(ctx); err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	// Verify tables are gone
	err = db.QueryRowContext(ctx,
		"SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('test_users', 'test_posts');").Scan(&tableCount)
	if err != nil {
		t.Fatalf("failed to check tables: %v", err)
	}
	if tableCount != 0 {
		t.Errorf("expected 0 tables after reset, got %d", tableCount)
	}
}

func TestTimestampParsing(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Record a migration
	m := &queen.Migration{
		Version: "001",
		Name:    "test_migration",
		UpSQL: `CREATE TABLE test (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		)`,
	}
	if err := driver.Record(ctx, m, nil); err != nil {
		t.Fatalf("Record() failed: %v", err)
	}

	// Get applied migrations
	applied, err := driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}

	if len(applied) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(applied))
	}

	// Verify timestamp was parsed correctly
	if applied[0].AppliedAt.IsZero() {
		t.Error("AppliedAt should not be zero")
	}

	// Verify timestamp is recent (within last minute)
	elapsed := time.Since(applied[0].AppliedAt)
	if elapsed > time.Minute {
		t.Errorf("AppliedAt timestamp seems incorrect: %v (elapsed: %v)", applied[0].AppliedAt, elapsed)
	}
}

func TestUnlock_WhenNotLocked(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Try to unlock when not locked - should be graceful and not return error
	err := driver.Unlock(ctx)
	if err != nil {
		t.Errorf("expected nil when unlocking without lock (graceful), got: %v", err)
	}
}

func TestLock_ContextCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)

	if err := driver.Init(context.Background()); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Acquire lock
	if err := driver.Lock(context.Background(), 5*time.Second); err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}
	defer func() { _ = driver.Unlock(context.Background()) }()

	// Try to acquire lock with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := driver.Lock(ctx, 5*time.Second)
	if err == nil {
		t.Error("expected error with canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
