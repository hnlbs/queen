package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

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
			expected: "`users`",
		},
		{
			name:     "table name with backtick",
			input:    "my`table",
			expected: "`my``table`",
		},
		{
			name:     "table name with multiple backticks",
			input:    "my`ta`ble",
			expected: "`my``ta``ble`",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "``",
		},
		{
			name:     "table name with spaces",
			input:    "my table",
			expected: "`my table`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base.QuoteBackticks(tt.input)
			if result != tt.expected {
				t.Errorf("base.QuoteBackticks(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDriverCreation tests driver creation functions.
func TestDriverCreation(t *testing.T) {
	db := &sql.DB{} // Mock DB for testing

	t.Run("New creates driver with default table name", func(t *testing.T) {
		driver := New(db)
		if driver.DB != db {
			t.Error("driver.DB should be set")
		}
		if driver.TableName != "queen_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "queen_migrations")
		}
		if driver.lockName != "queen_lock_queen_migrations" {
			t.Errorf("driver.lockName = %q; want %q", driver.lockName, "queen_lock_queen_migrations")
		}
	})

	t.Run("NewWithTableName creates driver with custom table name", func(t *testing.T) {
		driver := NewWithTableName(db, "custom_migrations")
		if driver.DB != db {
			t.Error("driver.DB should be set")
		}
		if driver.TableName != "custom_migrations" {
			t.Errorf("driver.TableName = %q; want %q", driver.TableName, "custom_migrations")
		}
		if driver.lockName != "queen_lock_custom_migrations" {
			t.Errorf("driver.lockName = %q; want %q", driver.lockName, "queen_lock_custom_migrations")
		}
	})
}

// Note: Integration tests that require a real MySQL database are in mysql_integration_test.go
// Run with: go test -tags=integration -v

// setupTestDB creates a test database connection.
// This requires MySQL to be running. Tests will be skipped if MySQL is not available.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Connect to MySQL (using port 3307 to avoid conflicts)
	db, err := sql.Open("mysql", "root:test@tcp(localhost:3307)/testdb?parseTime=true&multiStatements=true")
	if err != nil {
		t.Skip("MySQL not available:", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Skip("MySQL not available:", err)
	}

	// Cleanup function
	cleanup := func() {
		// Drop all test tables
		_, _ = db.Exec("DROP TABLE IF EXISTS queen_migrations")
		_, _ = db.Exec("DROP TABLE IF EXISTS test_users")
		_, _ = db.Exec("DROP TABLE IF EXISTS test_posts")
		_ = db.Close()
	}

	return db, cleanup
}

func TestIntegrationInit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	ctx := context.Background()

	// Init should create the table
	err := driver.Init(ctx)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Verify table exists
	var tableName string
	err = db.QueryRowContext(ctx, "SHOW TABLES LIKE 'queen_migrations'").Scan(&tableName)
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

func TestIntegrationRecordAndGetApplied(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
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
		UpSQL:   "CREATE TABLE users (id INT)",
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
		UpSQL:   "CREATE TABLE posts (id INT)",
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
		UpSQL:   "CREATE TABLE users (id INT)",
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
	db2, err := sql.Open("mysql", "root:test@tcp(localhost:3307)/testdb?parseTime=true")
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
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255)
			) ENGINE=InnoDB
		`)
		return err
	})
	if err != nil {
		t.Fatalf("Exec() failed: %v", err)
	}

	// Verify table was created
	var tableName string
	err = db.QueryRowContext(ctx, "SHOW TABLES LIKE 'test_users'").Scan(&tableName)
	if err != nil {
		t.Fatalf("table was not created: %v", err)
	}

	// Test failed transaction (should rollback)
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

	// Verify rollback (table should be empty)
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}

func TestIntegrationFullMigrationCycle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	driver := New(db)
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	ctx := context.Background()

	// Add migrations
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE test_users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				email VARCHAR(255) NOT NULL UNIQUE
			) ENGINE=InnoDB
		`,
		DownSQL: `DROP TABLE test_users`,
	})

	q.MustAdd(queen.M{
		Version: "002",
		Name:    "create_posts",
		UpSQL: `
			CREATE TABLE test_posts (
				id INT AUTO_INCREMENT PRIMARY KEY,
				user_id INT NOT NULL,
				title VARCHAR(255),
				FOREIGN KEY (user_id) REFERENCES test_users(id) ON DELETE CASCADE
			) ENGINE=InnoDB
		`,
		DownSQL: `DROP TABLE test_posts`,
	})

	// Apply all migrations
	if err := q.Up(ctx); err != nil {
		t.Fatalf("Up() failed: %v", err)
	}

	// Verify tables exist
	var tableCount int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'testdb' AND table_name IN ('test_users', 'test_posts')").Scan(&tableCount)
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
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'testdb' AND table_name IN ('test_users', 'test_posts')").Scan(&tableCount)
	if err != nil {
		t.Fatalf("failed to check tables: %v", err)
	}
	if tableCount != 0 {
		t.Errorf("expected 0 tables after reset, got %d", tableCount)
	}
}
