// Package checksum provides checksum calculation for migrations.
package checksum

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Calculate computes a SHA-256 checksum of the given migration content.
// It concatenates all provided strings and returns the hex-encoded hash.
//
// Whitespace is normalized before hashing to prevent false "modified" status
// when users reformat SQL without changing the actual logic.
func Calculate(content ...string) string {
	h := sha256.New()

	for _, c := range content {
		normalized := normalizeWhitespace(c)
		h.Write([]byte(normalized))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// normalizeWhitespace removes leading/trailing whitespace from each line
// and collapses multiple blank lines into one.
// This ensures formatting changes don't affect the checksum.
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines))

	prevEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip consecutive empty lines
		if trimmed == "" {
			if !prevEmpty {
				result = append(result, "")
				prevEmpty = true
			}
			continue
		}
		prevEmpty = false
		result = append(result, trimmed)
	}

	// Trim leading/trailing empty lines from result
	return strings.TrimSpace(strings.Join(result, "\n"))
}
