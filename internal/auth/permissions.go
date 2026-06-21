package auth

import "strings"

// Has reports whether permissions satisfies the required permission.
// Supports exact match and wildcard patterns like "invoices.*".
func Has(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == "*" || p == required || wildcardMatch(p, required) {
			return true
		}
	}
	return false
}

func wildcardMatch(pattern, target string) bool {
	if !strings.HasSuffix(pattern, ".*") {
		return false
	}
	return strings.HasPrefix(target, strings.TrimSuffix(pattern, "*"))
}
