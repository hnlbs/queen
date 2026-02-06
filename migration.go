package queen

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"github.com/honeynil/queen/internal/checksum"
)

type MigrationFunc func(ctx context.Context, tx *sql.Tx) error

// Migration represents a database migration.
type Migration struct {
	Version        string
	Name           string
	UpSQL          string
	DownSQL        string
	UpFunc         MigrationFunc
	DownFunc       MigrationFunc
	ManualChecksum string
	IsolationLevel sql.IsolationLevel

	checksumOnce *sync.Once
	checksum     string
}

type M = Migration

func (m *Migration) Validate() error {
	if m.Version == "" || strings.Contains(m.Version, " ") || !IsValidMigrationVersion(m.Version) {
		return ErrInvalidMigration
	}

	if len(m.Name) > 63 {
		return ErrNameTooLong
	}

	if m.Name == "" || !IsValidMigrationName(m.Name) {
		return ErrInvalidMigrationName
	}

	if m.UpSQL == "" && m.UpFunc == nil {
		return ErrInvalidMigration
	}

	return nil
}

const noChecksumMarker = "no-checksum-go-func"

func (m *Migration) Checksum() string {
	if m.checksumOnce == nil {
		m.checksumOnce = &sync.Once{}
	}

	m.checksumOnce.Do(func() {
		if m.ManualChecksum != "" {
			m.checksum = m.ManualChecksum
			return
		}

		if m.UpSQL != "" || m.DownSQL != "" {
			m.checksum = checksum.Calculate(m.UpSQL, m.DownSQL)
			return
		}

		m.checksum = noChecksumMarker
	})

	return m.checksum
}

func (m *Migration) HasRollback() bool {
	return m.DownSQL != "" || m.DownFunc != nil
}

func (m *Migration) IsDestructive() bool {
	if m.DownSQL == "" {
		return false
	}

	sql := strings.ToUpper(m.DownSQL)

	destructiveKeywords := []string{
		"DROP TABLE",
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
	}

	for _, keyword := range destructiveKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}

	return false
}

func (m *Migration) executeUp(ctx context.Context, tx *sql.Tx) error {
	if m.UpFunc != nil {
		return m.UpFunc(ctx, tx)
	}

	if m.UpSQL != "" {
		_, err := tx.ExecContext(ctx, m.UpSQL)
		return err
	}

	return ErrInvalidMigration
}

func (m *Migration) executeDown(ctx context.Context, tx *sql.Tx) error {
	if m.DownFunc != nil {
		return m.DownFunc(ctx, tx)
	}

	if m.DownSQL != "" {
		_, err := tx.ExecContext(ctx, m.DownSQL)
		return err
	}

	return ErrInvalidMigration
}
