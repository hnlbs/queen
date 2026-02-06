package queen

import "regexp"

var (
	migrationNameRE    = regexp.MustCompile(`^[a-z0-9_]+$`)
	migrationVersionRE = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// IsValidMigrationName reports whether name is a valid migration name.
// Valid names contain only lowercase letters, digits, and underscores.
func IsValidMigrationName(name string) bool {
	return migrationNameRE.MatchString(name)
}

// IsValidMigrationVersion reports whether version is a valid migration version.
// Valid versions contain only letters, digits, dots, dashes, and underscores.
func IsValidMigrationVersion(version string) bool {
	return migrationVersionRE.MatchString(version)
}
