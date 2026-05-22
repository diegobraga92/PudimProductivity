package shared

import (
	"crypto/rand"
	"fmt"
)

// NewUUID generates a UUID v4 string.
func NewUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("failed to generate UUID: %v", err))
	}

	// Set version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
