package queen

import (
	"fmt"
	"regexp"
	"strconv"
)

type NamingPattern string

const (
	NamingPatternNone             NamingPattern = ""
	NamingPatternSequential       NamingPattern = "sequential"
	NamingPatternSequentialPadded NamingPattern = "sequential-padded"
	NamingPatternSemver           NamingPattern = "semver"
)

var (
	sequentialRE = regexp.MustCompile(`^\d+$`)
	semverRE     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

type NamingConfig struct {
	Pattern NamingPattern
	Padding int
	Enforce bool
}

func DefaultNamingConfig() *NamingConfig {
	return &NamingConfig{
		Pattern: NamingPatternNone,
		Padding: 3,
		Enforce: true,
	}
}

func (nc *NamingConfig) Validate(version string) error {
	if nc == nil || nc.Pattern == NamingPatternNone {
		return nil
	}

	switch nc.Pattern {
	case NamingPatternSequential:
		return validateSequential(version)
	case NamingPatternSequentialPadded:
		return validateSequentialPadded(version, nc.Padding)
	case NamingPatternSemver:
		return validateSemver(version)
	default:
		return fmt.Errorf("unknown naming pattern: %s", nc.Pattern)
	}
}

func validateSequential(version string) error {
	if !sequentialRE.MatchString(version) {
		return fmt.Errorf("version must be a positive integer (e.g., 1, 2, 3): got %q", version)
	}

	if len(version) > 1 && version[0] == '0' {
		return fmt.Errorf("version must not have leading zeros (use 'sequential-padded' pattern instead): got %q", version)
	}

	return nil
}

func validateSequentialPadded(version string, padding int) error {
	if padding <= 0 {
		padding = 3
	}

	if !sequentialRE.MatchString(version) {
		return fmt.Errorf("version must be %d-digit format (e.g., %s): got %q",
			padding, fmt.Sprintf("%0*d", padding, 1), version)
	}

	if len(version) != padding {
		return fmt.Errorf("version must be %d-digit format (e.g., %s): got %q",
			padding, fmt.Sprintf("%0*d", padding, 1), version)
	}

	return nil
}

func validateSemver(version string) error {
	if !semverRE.MatchString(version) {
		return fmt.Errorf("version must be semantic version format (e.g., 1.0.0, 1.1.0): got %q", version)
	}
	return nil
}

func (nc *NamingConfig) FindNextVersion(existingVersions []string) (string, error) {
	if nc == nil || nc.Pattern == NamingPatternNone {
		return "", fmt.Errorf("naming pattern not configured")
	}

	switch nc.Pattern {
	case NamingPatternSequential:
		return findNextSequential(existingVersions, false, 0)
	case NamingPatternSequentialPadded:
		padding := nc.Padding
		if padding <= 0 {
			padding = 3
		}
		return findNextSequential(existingVersions, true, padding)
	case NamingPatternSemver:
		return "", fmt.Errorf("auto-generation not supported for semver pattern, please specify version manually")
	default:
		return "", fmt.Errorf("unknown naming pattern: %s", nc.Pattern)
	}
}

func findNextSequential(versions []string, padded bool, padding int) (string, error) {
	maxVersion := 0

	for _, v := range versions {
		num, err := strconv.Atoi(v)
		if err != nil {
			continue
		}

		if num > maxVersion {
			maxVersion = num
		}
	}

	nextVersion := maxVersion + 1

	if padded {
		return fmt.Sprintf("%0*d", padding, nextVersion), nil
	}

	return strconv.Itoa(nextVersion), nil
}
