package serialization

import (
	"github.com/google/uuid"
)

// GenerateUniqueID generates a unique identifier for a node.
func GenerateUniqueID() string {
	return uuid.New().String()
}
