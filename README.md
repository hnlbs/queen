<p align="center">
  <img src="assets/queen_logo.png" alt="Queen" width="200">
</p>

<h1 align="center">Queen</h1>

<p align="center">
  Lightweight database migration library for Go.<br>
  Define migrations in Go, not SQL files.
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

- **Migrations as Go code** — type-safe structs validated at compile time, no separate `.sql` files to ship.
- **SQL or Go functions** — pure SQL for schema, Go functions for data transformations. Both run inside a transaction.
- **6 databases** — PostgreSQL, MySQL, SQLite, ClickHouse, CockroachDB, MS SQL Server.
- **Embeddable CLI + TUI** — wire the CLI into your own binary alongside your migrations (see [CLI](#cli)).
- **Distributed locking** — safe to run from multiple instances, with configurable lock timeout.
- **Checksum validation** — SHA-256 over SQL detects modifications to already-applied migrations.
- **Gap detection** — finds missing, skipped, or unregistered migrations.
- **Dry run / plan / explain** — preview what will run before touching the database.
- **Naming patterns** — optional regex enforcement for migration versions.
- **Configurable isolation** — set transaction isolation level globally or per migration.
- **Rich metadata** — records who applied each migration, when, on which host, in which environment, and how long it took.
- **Migration toolkit** — `squash`, `baseline`, and `import` commands to consolidate history or onboard an existing database.

## Installation

```bash
go get github.com/honeynil/queen
```

Requires Go 1.24+.

## Quick Start

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

    q := queen.New(postgres.New(db))
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

    if err := q.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Go Function Migrations

When a change is more than schema, register a Go function instead of (or alongside) SQL. It runs inside the same transaction:

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_emails",
    ManualChecksum: "v1", // bump when the function logic changes
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx,
            `UPDATE users SET email = LOWER(TRIM(email))`)
        return err
    },
})
```

`UpSQL` and `UpFunc` can be combined in a single migration.

## CLI

Because migrations are Go code, Queen's CLI is shipped as a library you embed in your own `main.go` together with your migrations. This way the CLI always knows about the exact set of migrations your application ships with — no separate registry, no file scanning.

```go
// cmd/migrate/main.go
package main

import (
    "github.com/honeynil/queen"
    "github.com/honeynil/queen/cli"
)

func main() {
    cli.Run(func(q *queen.Queen) {
        q.MustAdd(queen.M{Version: "001", Name: "create_users", UpSQL: `...`, DownSQL: `...`})
        q.MustAdd(queen.M{Version: "002", Name: "add_index",    UpSQL: `...`, DownSQL: `...`})
    })
}
```

Then run any command against your project:

```bash
go run ./cmd/migrate up --driver postgres --dsn "postgres://localhost/myapp?sslmode=disable"
go run ./cmd/migrate status
go run ./cmd/migrate plan
```

Available commands: `up`, `down`, `reset`, `goto`, `status`, `log`, `plan`, `explain`, `validate`, `check`, `gap`, `diff`, `doctor`, `create`, `init`, `squash`, `baseline`, `import`, `tui`.

Configuration can also come from a `.queen.yaml` file (`--use-config`), with per-environment settings (`--env production`).

## Dry Run

Inspect what `Up` or `Down` will do without applying anything:

```go
plans, _ := q.DryRun(ctx, queen.DirectionUp, 0)
for _, p := range plans {
    fmt.Printf("%s %s [%s] destructive=%v warnings=%v\n",
        p.Version, p.Name, p.Type, p.IsDestructive, p.Warnings)
}
```

## Distributed Locking

When multiple instances of your app or CI run migrations concurrently, Queen acquires a database-level lock so only one runs at a time:

```go
q := queen.NewWithConfig(driver, &queen.Config{
    TableName:   "queen_migrations",
    LockTimeout: 10 * time.Minute,
})
```

Set `SkipLock: true` for single-instance setups or local development.

## Supported Databases

| Database    | Driver                      | SQL Driver                                  |
|-------------|-----------------------------|---------------------------------------------|
| PostgreSQL  | `queen/drivers/postgres`    | `github.com/jackc/pgx/v5/stdlib`            |
| MySQL       | `queen/drivers/mysql`       | `github.com/go-sql-driver/mysql`            |
| SQLite      | `queen/drivers/sqlite`      | `github.com/mattn/go-sqlite3`               |
| ClickHouse  | `queen/drivers/clickhouse`  | `github.com/ClickHouse/clickhouse-go/v2`    |
| CockroachDB | `queen/drivers/cockroachdb` | `github.com/jackc/pgx/v5/stdlib`            |
| MSSQL       | `queen/drivers/mssql`       | `github.com/microsoft/go-mssqldb`           |

## Documentation

Full documentation: [queen-wiki.honeynil.tech](https://queen-wiki.honeynil.tech) _(in progress)_

## License

Apache License 2.0
