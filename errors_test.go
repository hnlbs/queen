package queen

import (
	"errors"
	"testing"
)

func TestMigrationError(t *testing.T) {
	t.Run("with full context", func(t *testing.T) {
		cause := errors.New("table already exists")
		err := newMigrationError("001", "create_users", "up", "postgres", cause)

		var migErr *MigrationError
		if !errors.As(err, &migErr) {
			t.Fatal("expected *MigrationError")
		}

		if migErr.Version != "001" {
			t.Errorf("Version = %q, want %q", migErr.Version, "001")
		}
		if migErr.Name != "create_users" {
			t.Errorf("Name = %q, want %q", migErr.Name, "create_users")
		}
		if migErr.Operation != "up" {
			t.Errorf("Operation = %q, want %q", migErr.Operation, "up")
		}
		if migErr.Driver != "postgres" {
			t.Errorf("Driver = %q, want %q", migErr.Driver, "postgres")
		}

		expected := "migration 001 (create_users) failed during up operation on postgres: table already exists"
		if migErr.Error() != expected {
			t.Errorf("Error() = %q, want %q", migErr.Error(), expected)
		}

		if !errors.Is(err, cause) {
			t.Error("expected error to unwrap to cause")
		}
	})

	t.Run("with operation only", func(t *testing.T) {
		cause := errors.New("no down migration defined")
		err := newMigrationError("002", "add_column", "down", "", cause)

		var migErr *MigrationError
		if !errors.As(err, &migErr) {
			t.Fatal("expected *MigrationError")
		}
		expected := "migration 002 (add_column) failed during down: no down migration defined"
		if migErr.Error() != expected {
			t.Errorf("Error() = %q, want %q", migErr.Error(), expected)
		}
	})

	t.Run("minimal context", func(t *testing.T) {
		cause := errors.New("something went wrong")
		err := newMigrationError("003", "test_migration", "", "", cause)

		var migErr *MigrationError
		if !errors.As(err, &migErr) {
			t.Fatal("expected *MigrationError")
		}
		expected := "migration 003 (test_migration): something went wrong"
		if migErr.Error() != expected {
			t.Errorf("Error() = %q, want %q", migErr.Error(), expected)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		cause := errors.New("root cause")
		err := newMigrationError("001", "test", "up", "mysql", cause)

		unwrapped := errors.Unwrap(err)
		if !errors.Is(unwrapped, cause) {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
		}
	})
}
