package queen

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestNamingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *NamingConfig
		version string
		wantErr bool
	}{
		// NamingPatternNone
		{
			name:    "none pattern allows anything",
			config:  &NamingConfig{Pattern: NamingPatternNone},
			version: "anything_goes_123",
			wantErr: false,
		},
		{
			name:    "nil config allows anything",
			config:  nil,
			version: "anything",
			wantErr: false,
		},

		// NamingPatternSequential
		{
			name:    "sequential valid: 1",
			config:  &NamingConfig{Pattern: NamingPatternSequential},
			version: "1",
			wantErr: false,
		},
		{
			name:    "sequential valid: 123",
			config:  &NamingConfig{Pattern: NamingPatternSequential},
			version: "123",
			wantErr: false,
		},
		{
			name:    "sequential invalid: 001 (padded)",
			config:  &NamingConfig{Pattern: NamingPatternSequential},
			version: "001",
			wantErr: true,
		},
		{
			name:    "sequential invalid: abc",
			config:  &NamingConfig{Pattern: NamingPatternSequential},
			version: "abc",
			wantErr: true,
		},

		// NamingPatternSequentialPadded
		{
			name:    "sequential-padded valid: 001",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			version: "001",
			wantErr: false,
		},
		{
			name:    "sequential-padded valid: 999",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			version: "999",
			wantErr: false,
		},
		{
			name:    "sequential-padded invalid: 1 (not padded)",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			version: "1",
			wantErr: true,
		},
		{
			name:    "sequential-padded invalid: 0001 (too long)",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			version: "0001",
			wantErr: true,
		},
		{
			name:    "sequential-padded with padding 4: 0001",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 4},
			version: "0001",
			wantErr: false,
		},
		{
			name:    "sequential-padded default padding (3)",
			config:  &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 0},
			version: "001",
			wantErr: false,
		},

		// NamingPatternSemver
		{
			name:    "semver valid: 1.0.0",
			config:  &NamingConfig{Pattern: NamingPatternSemver},
			version: "1.0.0",
			wantErr: false,
		},
		{
			name:    "semver valid: 10.20.30",
			config:  &NamingConfig{Pattern: NamingPatternSemver},
			version: "10.20.30",
			wantErr: false,
		},
		{
			name:    "semver invalid: v1.0.0 (with v prefix)",
			config:  &NamingConfig{Pattern: NamingPatternSemver},
			version: "v1.0.0",
			wantErr: true,
		},
		{
			name:    "semver invalid: 1.0 (missing patch)",
			config:  &NamingConfig{Pattern: NamingPatternSemver},
			version: "1.0",
			wantErr: true,
		},

		// Unknown pattern
		{
			name:    "unknown pattern",
			config:  &NamingConfig{Pattern: "invalid"},
			version: "001",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNamingConfig_FindNextVersion(t *testing.T) {
	tests := []struct {
		name     string
		config   *NamingConfig
		existing []string
		want     string
		wantErr  bool
	}{
		// Sequential
		{
			name:     "sequential: first version",
			config:   &NamingConfig{Pattern: NamingPatternSequential},
			existing: []string{},
			want:     "1",
			wantErr:  false,
		},
		{
			name:     "sequential: next after 5",
			config:   &NamingConfig{Pattern: NamingPatternSequential},
			existing: []string{"1", "2", "3", "4", "5"},
			want:     "6",
			wantErr:  false,
		},
		{
			name:     "sequential: with gaps",
			config:   &NamingConfig{Pattern: NamingPatternSequential},
			existing: []string{"1", "3", "7"},
			want:     "8",
			wantErr:  false,
		},

		// SequentialPadded
		{
			name:     "sequential-padded: first version",
			config:   &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			existing: []string{},
			want:     "001",
			wantErr:  false,
		},
		{
			name:     "sequential-padded: next after 005",
			config:   &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3},
			existing: []string{"001", "002", "003", "004", "005"},
			want:     "006",
			wantErr:  false,
		},
		{
			name:     "sequential-padded: padding 4",
			config:   &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 4},
			existing: []string{"0001", "0002"},
			want:     "0003",
			wantErr:  false,
		},
		{
			name:     "sequential-padded: default padding (3)",
			config:   &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 0},
			existing: []string{},
			want:     "001",
			wantErr:  false,
		},

		// Semver
		{
			name:     "semver: not supported",
			config:   &NamingConfig{Pattern: NamingPatternSemver},
			existing: []string{},
			want:     "",
			wantErr:  true,
		},

		// No config
		{
			name:     "nil config",
			config:   nil,
			existing: []string{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "none pattern",
			config:   &NamingConfig{Pattern: NamingPatternNone},
			existing: []string{},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.FindNextVersion(tt.existing)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindNextVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindNextVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueen_Add_WithNamingValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *NamingConfig
		migrations  []M
		wantErr     bool
		errContains string
	}{
		{
			name:   "valid sequential-padded migrations",
			config: &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3, Enforce: true},
			migrations: []M{
				{Version: "001", Name: "first", UpSQL: "CREATE TABLE users"},
				{Version: "002", Name: "second", UpSQL: "CREATE TABLE posts"},
			},
			wantErr: false,
		},
		{
			name:   "invalid sequential-padded: wrong format",
			config: &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3, Enforce: true},
			migrations: []M{
				{Version: "1", Name: "first", UpSQL: "CREATE TABLE users"},
			},
			wantErr:     true,
			errContains: "naming pattern validation failed",
		},
		{
			name:   "enforce=false: warning only (no error)",
			config: &NamingConfig{Pattern: NamingPatternSequentialPadded, Padding: 3, Enforce: false},
			migrations: []M{
				{Version: "1", Name: "first", UpSQL: "CREATE TABLE users"},
			},
			wantErr: false,
		},
		{
			name:   "nil config: no validation",
			config: nil,
			migrations: []M{
				{Version: "anything", Name: "first", UpSQL: "CREATE TABLE users"},
				{Version: "v1.0.0", Name: "second", UpSQL: "CREATE TABLE posts"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &mockDriver{}
			config := &Config{
				TableName: "test_migrations",
				Naming:    tt.config,
			}
			q := NewWithConfig(driver, config)

			for _, m := range tt.migrations {
				err := q.Add(m)
				if (err != nil) != tt.wantErr {
					t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				}
				if err != nil && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Add() error = %v, should contain %q", err, tt.errContains)
				}
			}
		})
	}
}

// mockDriver for testing
type mockDriver struct{}

func (d *mockDriver) Init(ctx context.Context) error                        { return nil }
func (d *mockDriver) Close() error                                          { return nil }
func (d *mockDriver) Lock(ctx context.Context, timeout time.Duration) error { return nil }
func (d *mockDriver) Unlock(ctx context.Context) error                      { return nil }
func (d *mockDriver) GetApplied(ctx context.Context) ([]Applied, error)     { return nil, nil }
func (d *mockDriver) Record(ctx context.Context, migration *Migration, meta *MigrationMetadata) error {
	return nil
}
func (d *mockDriver) Remove(ctx context.Context, version string) error { return nil }
func (d *mockDriver) Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error {
	return nil
}
