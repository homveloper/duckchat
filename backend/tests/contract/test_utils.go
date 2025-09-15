package contract

import (
	"strings"
)

// contains checks if a string contains a substring
func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 &&
		(str == substr || len(str) > len(substr) &&
			(strings.Contains(str, substr)))
}