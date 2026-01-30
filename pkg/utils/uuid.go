package utils

import "github.com/google/uuid"

// IsUUID checks if the string is a valid UUID
func IsUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
