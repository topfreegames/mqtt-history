package app

import "strings"

// IsNotFoundError returns true if error is of not found type
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "not found")
}
