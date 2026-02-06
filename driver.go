package queen

import (
	"context"
	"database/sql"
	"time"
)

// Driver is the interface that database-specific migration drivers must implement.
type Driver interface {
	Init(ctx context.Context) error
	GetApplied(ctx context.Context) ([]Applied, error)
	Record(ctx context.Context, m *Migration, meta *MigrationMetadata) error
	Remove(ctx context.Context, version string) error
	Lock(ctx context.Context, timeout time.Duration) error
	Unlock(ctx context.Context) error
	Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error
	Close() error
}

// MigrationMetadata contains metadata collected during migration execution.
type MigrationMetadata struct {
	AppliedBy    string
	DurationMS   int64
	Hostname     string
	Environment  string
	Action       string
	Status       string
	ErrorMessage string
}

// Applied represents a migration that has been applied.
type Applied struct {
	Version      string
	Name         string
	AppliedAt    time.Time
	Checksum     string
	AppliedBy    string
	DurationMS   int64
	Hostname     string
	Environment  string
	Action       string
	Status       string
	ErrorMessage string
}
