package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a unique ID
func GenerateID(prefix string) string {
	// Generate random bytes
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp if crypto rand fails
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}

	// Combine timestamp with random bytes for uniqueness
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d_%s", prefix, timestamp, hex.EncodeToString(randomBytes))
}
