package queen

import "regexp"

// IsValidMigrationName checks if a migration name is valid.
func IsValidMigrationName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, name)
	return matched
}
