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

---

## Features

- **Code-as-migrations** — migrations are Go structs, not SQL files
- **SQL and Go functions** — plain SQL, Go functions, or both in one migration
- **Transactions** — every migration runs inside a transaction with configurable isolation level
- **Advisory locks** — prevents concurrent migration runs on the same database
- **Auto-versioning** — automatic version numbering based on naming patterns (sequential, padded, semver)
- **Checksums** — detects modified migrations after apply
- **Gap detection** — finds and fixes out-of-order or missing migrations
- **Health checks** — `doctor` command with deep SQL analysis
- **Built-in CLI** — `up`, `down`, `status`, `plan`, `create`, `doctor`, and more
- **Terminal UI** — interactive TUI for managing migrations
- **Import from goose** — one command to convert goose migrations
- **6 databases** — PostgreSQL, MySQL, SQLite, ClickHouse, CockroachDB, MSSQL

## Install

```bash
go get github.com/honeynil/queen
```

## Quick Start

### 1. Define migrations in Go

```go
// migrations/001_create_users.go
package migrations

import "github.com/honeynil/queen"

func Register001CreateUsers(q *queen.Queen) {
	q.MustAdd(queen.M{
		Version: "001",
		Name:    "create_users",
		UpSQL: `
			CREATE TABLE users (
				id SERIAL PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				name VARCHAR(255),
				created_at TIMESTAMP DEFAULT NOW()
			)`,
		DownSQL: `DROP TABLE users`,
	})
}
```

### 2. Register and run

```go
// cmd/migrate/main.go
package main

import (
	"github.com/honeynil/queen/cli"
	"yourmodule/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cli.Run(migrations.Register)
}
```

### 3. Use the CLI

```bash
go build -o migrate cmd/migrate/main.go

./migrate up                    # apply all pending
./migrate down 1                # rollback last
./migrate status                # show state
```

## Go Function Migrations

For complex logic that SQL can't express:

```go
q.MustAdd(queen.M{
	Version:        "003",
	Name:           "seed_admin",
	ManualChecksum: "v1",
	UpFunc: func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO users (email, name) VALUES ($1, $2)`,
			"admin@example.com", "Admin")
		return err
	},
	DownFunc: func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM users WHERE email = $1`,
			"admin@example.com")
		return err
	},
})
```

## CLI

Initialize a project:

```bash
queen init --driver postgres          # scaffold project structure
queen init --interactive              # TUI setup wizard
```

Create migrations:

```bash
queen create add_posts --type sql     # generate SQL migration file
queen create seed_data --type go      # generate Go function migration
```

Manage migrations:

```bash
queen up                              # apply all pending
queen up --steps 1                    # apply one
queen down 1                          # rollback last
queen status                          # show migration state
queen plan                            # preview what will run
queen explain 001                     # inspect specific migration
```

Diagnostics:

```bash
queen doctor                          # run health checks
queen doctor --deep                   # deep SQL analysis
queen doctor --fix                    # show fix recommendations
queen gap detect                      # find migration gaps
queen tui                             # interactive terminal UI
```

Import from goose:

```bash
queen import ./sql_migrations --from goose --output migrations
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

## License

Apache License 2.0
