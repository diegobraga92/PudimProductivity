package sharedkernel

import (
	"errors"

	"github.com/google/uuid"
)

// ID is the canonical identifier for entities across the bounded contexts. It
// is a UUID v4 serialized as a string; contexts must treat it as opaque.
type ID string

// NewID returns a new random (v4) identifier.
func NewID() ID {
	return ID(uuid.New().String())
}

// Parse validates s as a well-formed identifier and returns it as an ID.
func Parse(s string) (ID, error) {
	if s == "" {
		return "", errors.New("sharedkernel: empty id")
	}
	if _, err := uuid.Parse(s); err != nil {
		return "", errors.New("sharedkernel: invalid id: " + s)
	}
	return ID(s), nil
}

// String returns the identifier as a string.
func (id ID) String() string { return string(id) }

// IsZero reports whether the identifier is the zero value.
func (id ID) IsZero() bool { return id == "" }
