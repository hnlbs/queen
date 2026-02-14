<p align="center">
  <img src="assets/queen_logo.png" alt="Queen" width="200">
</p>

<h1 align="center">Queen</h1>

<p align="center">
  Lightweight database migration library for Go.<br>
  Migrations are defined as code, not files — type safety, IDE support, and easy testing included.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/honeynil/queen"><img src="https://pkg.go.dev/badge/github.com/honeynil/queen.svg" alt="Go Reference"></a>
  <a href="https://github.com/honeynil/queen/actions/workflows/test.yml"><img src="https://github.com/honeynil/queen/actions/workflows/test.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/honeynil/queen/actions/workflows/integration-tests.yml"><img src="https://github.com/honeynil/queen/actions/workflows/integration-tests.yml/badge.svg" alt="Integration Tests"></a>
  <a href="https://goreportcard.com/report/github.com/honeynil/queen"><img src="https://goreportcard.com/badge/github.com/honeynil/queen" alt="Go Report Card"></a>
  <a href="https://github.com/honeynil/queen/releases"><img src="https://img.shields.io/github/v/release/honeynil/queen" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License"></a>
</p>


## Features

**Migrations as Code** - Type-safe Go structs with compile-time validation. No external files to manage.

**SQL and Go Functions** - Pure SQL for schema changes, Go functions for complex data transformations. Both run inside transactions.

**Gap Detection** - Automatically detect and resolve missing, skipped, or unregistered migrations.

**6 Databases** - PostgreSQL, MySQL, SQLite, ClickHouse, CockroachDB, MS SQL Server.

**Checksum Validation** - Automatic SHA-256 checksums detect when applied migrations are modified.

**Rich Metadata** - Track who applied each migration, when, on which host, and how long it took.

## Migration Example
```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/honeynil/queen"
    "github.com/honeynil/queen/drivers/postgres"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    db, err := sql.Open("pgx", "postgres://localhost/myapp?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    driver := postgres.New(db)
    q := queen.New(driver)
    defer q.Close()

    q.MustAdd(queen.M{
        Version: "001",
        Name:    "create_users_table",
        UpSQL: `
            CREATE TABLE users (
                id SERIAL PRIMARY KEY,
                email VARCHAR(255) NOT NULL UNIQUE,
                name VARCHAR(255),
                created_at TIMESTAMP DEFAULT NOW()
            )
        `,
        DownSQL: `DROP TABLE users`,
    })

    ctx := context.Background()
    if err := q.Up(ctx); err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("All migrations applied")

    statuses, err := q.Status(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, s := range statuses {
        log.Printf("Migration %s (%s): %s", s.Version, s.Name, s.Status)
    }
}
```

## Supported Databases

| Database | Driver | SQL Driver |
|---|---|---|
| PostgreSQL | `queen/drivers/postgres` | `github.com/jackc/pgx/v5/stdlib` |
| MySQL | `queen/drivers/mysql` | `github.com/go-sql-driver/mysql` |
| SQLite | `queen/drivers/sqlite` | `github.com/mattn/go-sqlite3` |
| ClickHouse | `queen/drivers/clickhouse` | `github.com/ClickHouse/clickhouse-go/v2` |
| CockroachDB | `queen/drivers/cockroachdb` | `github.com/jackc/pgx/v5/stdlib` |
| MSSQL | `queen/drivers/mssql` | `github.com/microsoft/go-mssqldb` |

## Documentation

Full documentation: [queen-wiki.honeynil.tech](https://queen-wiki.honeynil.tech)

Apache License 2.0
