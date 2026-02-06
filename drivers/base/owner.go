// Package base provides common functionality for database drivers.
package base

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateOwnerID generates a unique owner identifier for lock ownership tracking.
func GenerateOwnerID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
