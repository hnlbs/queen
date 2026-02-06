package ydb_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/ydb-platform/ydb-go-sdk/v3" // YDB driver for database/sql

	"github.com/honeynil/queen"
	"github.com/honeynil/queen/drivers/ydb"
)

// Example demonstrates basic usage of the YDB driver.
func Example() {
	// Connect to YDB
	// Connection string format: grpc://host:port/database
	// For secure connection use grpcs://
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Create YDB driver
	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create Queen instance
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	// Register migrations
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users_table",
		UpSQL: `
			CREATE TABLE users (
				id         Utf8,
				email      Utf8 NOT NULL,
				name       Utf8,
				created_at Timestamp,
				PRIMARY KEY (id)
			)
		`,
		DownSQL: `DROP TABLE users`,
	})

	q.MustAdd(queen.M{
		Version: "002",
		Name:    "add_users_bio",
		UpSQL:   `ALTER TABLE users ADD COLUMN bio Utf8`,
		DownSQL: `ALTER TABLE users DROP COLUMN bio`,
	})

	// Apply all pending migrations
	ctx := context.Background()
	if err := q.Up(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Migrations applied successfully!")
}

// Example_customTableName demonstrates using a custom table name for migrations.
func Example_customTableName() {
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Use custom table name
	driver, err := ydb.NewWithTableName(db, "my_custom_migrations")
	if err != nil {
		log.Fatal(err)
	}
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	// The migrations will be tracked in "my_custom_migrations" table
	// instead of the default "queen_migrations"
}

// Example_goFunctionMigration demonstrates using Go functions for complex migrations.
func Example_goFunctionMigration() {
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	// SQL migration to insert initial data
	q.MustAdd(queen.M{
		Version: "003",
		Name:    "add_users",
		UpSQL: `
			UPSERT INTO users (id, email, name)
			VALUES
				('1', 'alice@example.com', 'Alice Smith'),
				('2', 'bob@example.com', 'Bob Johnson'),
				('3', 'carol@example.com', 'Carol Williams'),
				('4', 'david@example.com', 'David Brown'),
				('5', 'eve@example.com', 'Eve Davis')
		`,
		DownSQL: `DELETE FROM users WHERE id IN ('1', '2', '3', '4', '5')`,
	})

	// Migration using Go function for complex logic
	q.MustAdd(queen.M{
		Version:        "004",
		Name:           "normalize_names",
		ManualChecksum: "v1", // Important: track function changes!
		UpFunc: func(ctx context.Context, tx *sql.Tx) error {
			// Fetch all users
			rows, err := tx.QueryContext(ctx, "SELECT id, name FROM users")
			if err != nil {
				return err
			}
			defer func() { _ = rows.Close() }()

			// Normalize each name
			for rows.Next() {
				var id string
				var name string
				if err := rows.Scan(&id, &name); err != nil {
					return err
				}

				// Convert to uppercase
				normalized := normalizeNames(name)

				// Update using UPSERT (YDB-specific)
				_, err = tx.ExecContext(ctx,
					"UPSERT INTO users (id, name) VALUES ($1, $2)",
					id, normalized)
				if err != nil {
					return err
				}
			}

			return rows.Err()
		},
		DownFunc: func(ctx context.Context, tx *sql.Tx) error {
			// Rollback is not possible for this migration
			return nil
		},
	})

	ctx := context.Background()
	if err := q.Up(ctx); err != nil {
		log.Fatal(err)
	}
}

// Example_withConfig demonstrates using custom configuration.
func Example_withConfig() {
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create Queen with custom config
	config := &queen.Config{
		TableName:   "custom_migrations",
		LockTimeout: 10 * time.Minute, // 10 minutes
	}
	q := queen.NewWithConfig(driver, config)
	defer func() { _ = q.Close() }()

	// Your migrations here
}

// Example_status demonstrates checking migration status.
//
// Note: This example requires a running YDB server.
// It will be skipped in CI if YDB is not available.
func Example_status() {
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		fmt.Println("YDB not available")
		return
	}
	defer func() { _ = db.Close() }()

	// Check if YDB is actually available
	if err := db.Ping(); err != nil {
		fmt.Println("YDB not available")
		return
	}

	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}
	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	// Register migrations
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE users (
				id   Utf8,
				name Utf8,
				PRIMARY KEY (id)
			)
		`,
		DownSQL: `DROP TABLE users`,
	})

	q.MustAdd(queen.M{
		Version: "002",
		Name:    "add_users_bio",
		UpSQL:   `ALTER TABLE users ADD COLUMN bio Utf8`,
		DownSQL: `ALTER TABLE users DROP COLUMN bio`,
	})

	ctx := context.Background()

	// Apply first migration only
	if err := q.UpSteps(ctx, 1); err != nil {
		log.Fatal(err)
	}

	// Check status
	statuses, err := q.Status(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range statuses {
		fmt.Printf("%s: %s (%s)\n", s.Version, s.Name, s.Status)
	}

	// Example output (when YDB is available):
	// 001: create_users (applied)
	// 002: add_users_bio (pending)
}

// Example_withAuthentication demonstrates connecting to YDB with authentication.
func Example_withAuthentication() {
	// For YDB with authentication, use connection string with credentials:
	// grpcs://user:password@host:port/database
	db, err := sql.Open("ydb", "grpcs://root:password@localhost:2135/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}

	q := queen.New(driver)
	defer func() { _ = q.Close() }()

	// Your migrations here
}

// Example_transactionIsolation demonstrates using custom transaction isolation levels.
func Example_transactionIsolation() {
	db, err := sql.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	driver, err := ydb.New(db)
	if err != nil {
		log.Fatal(err)
	}

	// Create Queen with custom isolation level
	config := &queen.Config{
		TableName:      "migrations",
		IsolationLevel: sql.LevelSerializable, // YDB default is Serializable
	}
	q := queen.NewWithConfig(driver, config)
	defer func() { _ = q.Close() }()

	// You can also set isolation level per migration
	q.MustAdd(queen.M{
		Version:        "001",
		Name:           "create_users",
		IsolationLevel: sql.LevelSerializable,
		UpSQL: `
			CREATE TABLE users (
				id   Utf8,
				name Utf8,
				PRIMARY KEY (id)
			)
		`,
		DownSQL: `DROP TABLE users`,
	})

	ctx := context.Background()
	if err := q.Up(ctx); err != nil {
		log.Fatal(err)
	}
}

// Helper function for name normalization
func normalizeNames(name string) string {
	// Simple normalization (uppercase)
	// In real code, you might want more sophisticated logic
	return strings.ToUpper(name)
}
