package clickhouse

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/honeynil/queen"
	"github.com/honeynil/queen/drivers/base"
)

// TestQuoteIdentifier tests the identifier quoting function.
func TestQuoteIdentifier(t *testing.T) {
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
				t.Errorf("base.QuoteDoubleQuotes(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDriverCreation tests driver creation functions.
func TestDriverCreation(t *testing.T) {
	db := &sql.DB{} // Mock DB for testing

	t.Run("New creates driver with default table name", func(t *testing.T) {
		driver, err := New(db)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		if driver.DB != db {
			t.Error("driver.DB should be set")
		}
		if driver.TableName != "queen_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "queen_migrations")
		}
		if driver.ownerID == "" {
			t.Error("driver.ownerID should be set")
		}
	})

	t.Run("NewWithTableName creates driver with custom table name", func(t *testing.T) {
		driver, err := NewWithTableName(db, "custom_migrations")
		if err != nil {
			t.Fatalf("NewWithTableName() failed: %v", err)
		}
		if driver.DB != db {
			t.Error("driver.DB should be set")
		}
		if driver.TableName != "custom_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "custom_migrations")
		}
		if driver.ownerID == "" {
			t.Error("driver.ownerID should be set")
		}
	})

	t.Run("New generates unique owner IDs", func(t *testing.T) {
		driver1, err := New(db)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		driver2, err := New(db)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		if driver1.ownerID == driver2.ownerID {
			t.Error("expected different owner IDs for different driver instances")
		}
	})
}

// setupTestDB creates a test database connection.
// This requires ClickHouse to be running. Tests will be skipped if ClickHouse is not available.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Try to get DSN from environment variable first
	dsn := os.Getenv("CLICKHOUSE_TEST_DSN")
	if dsn == "" {
		// Default DSN for local testing
		dsn = "clickhouse://default@localhost:9000/default?dial_timeout=5s"
	}

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		t.Skip("ClickHouse not available:", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skip("ClickHouse not available:", err)
	}

	// Cleanup function
	cleanup := func() {
		// Drop all test tables
		_, _ = db.Exec("DROP TABLE IF EXISTS queen_migrations")
		_, _ = db.Exec("DROP TABLE IF EXISTS queen_migrations_lock")
		_, _ = db.Exec("DROP TABLE IF EXISTS test_users")
		_, _ = db.Exec("DROP TABLE IF EXISTS test_posts")
		_ = db.Close()
	}

	return db, cleanup
}

func TestInit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	// Init should create the table
	err = driver.Init(ctx)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.QueryRowContext(ctx,
		"SELECT name FROM system.tables WHERE database = 'default' AND name = 'queen_migrations'").Scan(&tableName)
	if err != nil {
		t.Fatalf("migrations table was not created: %v", err)
	}
	if tableName != "queen_migrations" {
		t.Errorf("table name = %q; want %q", tableName, "queen_migrations")
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

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	// Init
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
			id UUID DEFAULT generateUUIDv4()
		) ENGINE = MergeTree() ORDER BY id`,
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
			id UUID DEFAULT generateUUIDv4()
		) ENGINE = MergeTree() ORDER BY id`,
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

func TestRemove(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	// Init and record a migration
	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	m := &queen.Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL: `CREATE TABLE test_users (
			id UUID DEFAULT generateUUIDv4()
		) ENGINE = MergeTree() ORDER BY id`,
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
	applied, err = driver.GetApplied(ctx)
	if err != nil {
		t.Fatalf("GetApplied() failed: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("expected 0 migrations after removal, got %d", len(applied))
	}
}

func TestLocking(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Acquire lock
	err = driver.Lock(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("Lock() failed: %v", err)
	}

	// Try to acquire the same lock from the same driver instance (should fail)
	err = driver.Lock(ctx, 100*time.Millisecond)
	if !errors.Is(err, queen.ErrLockTimeout) {
		t.Errorf("expected ErrLockTimeout, got %v", err)
	}

	// Release lock
	if err := driver.Unlock(ctx); err != nil {
		t.Fatalf("Unlock() failed: %v", err)
	}

	// Now should be able to acquire the lock again
	err = driver.Lock(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("second Lock() failed after unlock: %v", err)
	}

	// Clean up
	_ = driver.Unlock(ctx)
}

func TestExec(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test successful transaction
	err = driver.Exec(ctx, sql.LevelDefault, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			CREATE TABLE test_users (
				id UUID DEFAULT generateUUIDv4(),
				name String
			) ENGINE = MergeTree() ORDER BY id
		`)
		return err
	})
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	// Verify table was created
	var tableName string
	err = db.QueryRowContext(ctx,
		"SELECT name FROM system.tables WHERE database = 'default' AND name = 'test_users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("table was not created: %v", err)
	}

	// Test failed transaction (should rollback)
	// Note: ClickHouse doesn't support full ACID transactions like PostgreSQL/MySQL,
	// so rollback behavior may be limited. This test verifies the error handling.
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

func TestFullMigrationCycle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	ctx := context.Background()

	// Add migrations
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE test_users (
				id UUID DEFAULT generateUUIDv4(),
				email String NOT NULL
			) ENGINE = ReplacingMergeTree() ORDER BY id
		`,
		DownSQL: `DROP TABLE test_users`,
	})

	q.MustAdd(queen.M{
		Version: "002",
		Name:    "create_posts",
		UpSQL: `
			CREATE TABLE test_posts (
				id UUID DEFAULT generateUUIDv4(),
				user_id UUID NOT NULL,
				title String
			) ENGINE = ReplacingMergeTree() ORDER BY id
		`,
		DownSQL: `DROP TABLE test_posts`,
	})

	// Apply all migrations
	if err := q.Up(ctx); err != nil {
		t.Fatalf("Up() failed: %v", err)
	}

	// Verify tables exist
	var tableCount uint64
	err = db.QueryRowContext(ctx,
		"SELECT count() FROM system.tables WHERE database = 'default' AND name IN ('test_users', 'test_posts')").Scan(&tableCount)
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
		"SELECT count() FROM system.tables WHERE database = 'default' AND name IN ('test_users', 'test_posts')").Scan(&tableCount)
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

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Record a migration
	m := &queen.Migration{
		Version: "001",
		Name:    "test_migration",
		UpSQL: `CREATE TABLE test (
			id UUID DEFAULT generateUUIDv4()
		) ENGINE = MergeTree() ORDER BY id`,
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

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	ctx := context.Background()

	if err := driver.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Try to unlock when not locked - should be graceful and not return error
	err = driver.Unlock(ctx)
	if err != nil {
		t.Errorf("expected nil when unlocking without lock (graceful), got: %v", err)
	}
}

func TestLockOwnership_PreventsCrossProcessUnlock(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create two driver instances simulating two processes
	driverA, err := New(db)
	if err != nil {
		t.Fatalf("New(driverA) failed: %v", err)
	}
	driverB, err := New(db)
	if err != nil {
		t.Fatalf("New(driverB) failed: %v", err)
	}

	ctx := context.Background()

	if err := driverA.Init(ctx); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Process A acquires lock
	if err := driverA.Lock(ctx, 30*time.Second); err != nil {
		t.Fatalf("driverA.Lock() failed: %v", err)
	}

	// Simulate lock expiration by manually deleting the lock
	// In ClickHouse, we need to use ALTER TABLE DELETE
	_, err = db.ExecContext(ctx, "ALTER TABLE queen_migrations_lock DELETE WHERE lock_key = 'migration_lock'")
	if err != nil {
		t.Fatalf("failed to delete lock: %v", err)
	}

	// Wait for ClickHouse to process the deletion (async operation)
	time.Sleep(2 * time.Second)

	// Verify lock was deleted
	var count int64
	err = db.QueryRowContext(ctx,
		"SELECT count(*) FROM queen_migrations_lock FINAL WHERE lock_key = 'migration_lock'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check lock count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected lock to be deleted, got count=%d", count)
	}

	// Process B acquires new lock (should succeed since A's lock expired)
	if err := driverB.Lock(ctx, 30*time.Second); err != nil {
		t.Fatalf("driverB.Lock() should succeed after A's lock expired: %v", err)
	}

	// Verify Process B's lock exists
	err = db.QueryRowContext(ctx,
		"SELECT count(*) FROM queen_migrations_lock FINAL WHERE lock_key = 'migration_lock'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check lock count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected Process B's lock to exist, got count=%d", count)
	}

	// Process A tries to unlock - should NOT delete Process B's lock
	// This is the critical test: driverA.Unlock() should be graceful and not affect driverB's lock
	if err := driverA.Unlock(ctx); err != nil {
		t.Fatalf("driverA.Unlock() should be graceful: %v", err)
	}

	// Wait for ClickHouse to process any deletions
	time.Sleep(1 * time.Second)

	// Verify Process B's lock STILL EXISTS (critical assertion)
	err = db.QueryRowContext(ctx,
		"SELECT count(*) FROM queen_migrations_lock FINAL WHERE lock_key = 'migration_lock'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check lock count: %v", err)
	}
	if count != 1 {
		t.Errorf("CRITICAL: Process A unlocked Process B's lock! Expected count=1, got count=%d", count)
	}

	// Process B unlocks successfully
	if err := driverB.Unlock(ctx); err != nil {
		t.Fatalf("driverB.Unlock() failed: %v", err)
	}

	// Verify lock is now gone
	time.Sleep(1 * time.Second)
	err = db.QueryRowContext(ctx,
		"SELECT count(*) FROM queen_migrations_lock FINAL WHERE lock_key = 'migration_lock'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check lock count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected lock to be deleted after driverB.Unlock(), got count=%d", count)
	}
}

func TestLock_ContextCancellation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver, err := New(db)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

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

	err = driver.Lock(ctx, 5*time.Second)
	if err == nil {
		t.Error("expected error with canceled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
