# Queen CLI

Command-line interface for managing database migrations with Queen.

## Quick Start

```bash
# Initialize project
queen init --driver postgres

# Create migration
queen create add_users

# Apply migrations
queen up

# Check status
queen status
```

## Configuration

Configuration priority (highest to lowest):
1. Command-line flags
2. Environment variables (`QUEEN_DRIVER`, `QUEEN_DSN`, etc.)
3. Config file `.queen.yaml` (requires `--use-config`)

## Commands Reference

### Core Commands

Apply and rollback migrations.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen up` | Apply pending migrations | `queen up`<br>`queen up --steps 3`<br>`queen up --to 050` |
| `queen down` | Rollback migrations | `queen down`<br>`queen down --steps 2`<br>`queen down --to 040` |
| `queen goto` | Migrate to specific version (up or down) | `queen goto 045` |
| `queen reset` | Rollback all migrations | `queen reset` |

### Information Commands

View migration status and history.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen status` | Show current migration status | `queen status`<br>`queen status --json`<br>`queen status --pending-only` |
| `queen log` | Show migration history | `queen log --last 10`<br>`queen log --since 2026-01-01`<br>`queen log --with-duration` |
| `queen version` | Show current database version | `queen version` |
| `queen diff` | Compare two versions | `queen diff 001 005`<br>`queen diff current +3`<br>`queen diff --show-sql` |

### Management Commands

Create and manage migrations.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen create` | Create new migration | `queen create add_users`<br>`queen create migrate_data --type go`<br>`queen create --interactive` |
| `queen squash` | Combine multiple migrations into one | `queen squash 001,002,003 --into merged`<br>`queen squash --from 001 --to 010` |
| `queen baseline` | Create baseline migration from current schema | `queen baseline --name initial_schema`<br>`queen baseline --at 050` |

### Diagnostics Commands

Validate and diagnose migration health.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen doctor` | Full health check and diagnostics | `queen doctor`<br>`queen doctor --deep`<br>`queen doctor --gaps`<br>`queen doctor --fix` |
| `queen check` | Quick validation for CI/CD | `queen check --ci`<br>`queen check --no-pending` |
| `queen validate` | Validate migration definitions | `queen validate` |
| `queen plan` | Show execution plan (dry-run) | `queen plan`<br>`queen plan --direction down`<br>`queen plan --json` |
| `queen explain` | Explain specific migration | `queen explain 001`<br>`queen explain 001 --json` |

### Gap Detection Commands

Detect and manage migration gaps.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen gap detect` | Detect migration gaps | `queen gap detect --json` |
| `queen gap fill` | Fill detected gaps | `queen gap fill`<br>`queen gap fill 003,004`<br>`queen gap fill --mark-applied` |
| `queen gap analyze` | Analyze gap dependencies | `queen gap analyze` |
| `queen gap ignore` | Ignore specific gaps | `queen gap ignore 003 --reason "manually applied"` |

### Utility Commands

Project initialization and tools.

| Command | Description | Examples |
|---------|-------------|----------|
| `queen init` | Initialize Queen in project | `queen init`<br>`queen init --driver postgres`<br>`queen init --with-config`<br>`queen init --interactive` |
| `queen import` | Import migrations from other systems | `queen import --from goose ./migrations` |
| `queen tui` | Launch interactive Terminal UI mode | `queen tui` |

## Global Flags

Available for all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--driver` | Database driver (postgres, mysql, sqlite, clickhouse) | - |
| `--dsn` | Database connection string | - |
| `--table` | Migration table name | `queen_migrations` |
| `--timeout` | Lock timeout (e.g. 30m, 1h) | - |
| `--use-config` | Enable config file (.queen.yaml) | `false` |
| `--env` | Environment from config file | - |
| `--unlock-production` | Unlock production environment | `false` |
| `--yes` | Skip confirmation prompts (for CI/CD) | `false` |
| `--json` | Output in JSON format | `false` |
| `--verbose` | Verbose output | `false` |

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `QUEEN_DRIVER` | Database driver | `postgres` |
| `QUEEN_DSN` | Database connection string | `postgres://localhost/myapp` |
| `QUEEN_TABLE` | Migration table name | `queen_migrations` |
| `QUEEN_LOCK_TIMEOUT` | Lock timeout | `30m` |

## TUI Mode (Terminal UI)

Queen includes an interactive Terminal UI for managing migrations visually.

### Launching TUI
```bash
queen tui
```

### Features
- **Migrations View**: Visual list of all migrations with status (pending/applied)
- **Gaps View**: Automatic gap detection with detailed diagnostics
- **Interactive Navigation**: Navigate with arrow keys, apply/rollback with enter
- **Real-time Updates**: Live status updates after operations
- **Gap Management**: Fill gaps or add them to .queenignore interactively

### Keyboard Shortcuts

**Navigation:**
- `↑/k` - Move cursor up
- `↓/j` - Move cursor down
- `g` - Jump to top
- `G` - Jump to bottom

**Views:**
- `1` - Migrations view
- `2` - Gaps detection view
- `3/?` - Help view

**Actions (Migrations View):**
- `enter` - Apply pending migration / Rollback applied migration
- `u` - Apply migration up to cursor
- `d` - Rollback migration from cursor

**Actions (Gaps View):**
- `enter/f` - Fill the selected gap
- `i` - Ignore the selected gap (add to .queenignore)

**General:**
- `r` - Refresh data
- `q/Ctrl+C` - Quit

### Example TUI Workflow
1. Launch TUI: `queen tui`
2. Press `2` to view gaps
3. Navigate to a gap and press `f` to fill it
4. Press `1` to return to migrations view
5. Navigate to a migration and press `enter` to apply it
6. Press `r` to refresh status

## Common Workflows

### Development Workflow
```bash
# Create and apply migration
queen create add_feature
queen up

# Check what happened
queen status
queen log --last 5
```

### Rollback Workflow
```bash
# Rollback last migration
queen down

# Rollback to specific version
queen down --to 040

# Or use goto
queen goto 040
```

### CI/CD Workflow
```bash
# Validate migrations
queen check --ci

# Apply with auto-confirm
queen up --yes

# Verify
queen status --json
```

### Gap Detection Workflow
```bash
# Detect gaps
queen gap detect

# Analyze impact
queen gap analyze

# Fill gaps
queen gap fill --apply
```

### Diagnostics Workflow
```bash
# Full health check
queen doctor

# Check for gaps
queen doctor --gaps

# Deep schema validation
queen doctor --deep

# Auto-fix issues
queen doctor --fix
```

## Supported Databases

| Database | Driver | Connection String Example |
|----------|--------|---------------------------|
| PostgreSQL | `postgres` | `postgres://user:pass@localhost/db?sslmode=disable` |
| MySQL | `mysql` | `user:pass@tcp(localhost:3306)/db?parseTime=true` |
| SQLite | `sqlite` | `./app.db?_journal_mode=WAL` |
| ClickHouse | `clickhouse` | `tcp://localhost:9000/db` |
| MSSQL | `mssql` | `sqlserver://user:pass@localhost:1433?database=db` |

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Migration failed |
| 4 | Gap detected (with --ci flag) |
| 5 | Validation failed |

## Getting Help

```bash
# General help
queen --help

# Command-specific help
queen up --help
queen doctor --help

# Version information
queen version
```

For detailed documentation and examples, visit: https://github.com/honeynil/queen
