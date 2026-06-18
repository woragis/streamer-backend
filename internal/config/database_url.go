package config

import "strings"

// NormalizeDatabaseURL ensures Railway/managed Postgres URLs work with pgx.
func NormalizeDatabaseURL(raw string) string {
	u := trimQuotes(strings.TrimSpace(raw))
	if u == "" {
		return u
	}
	lower := strings.ToLower(u)
	if strings.Contains(lower, "sslmode=") {
		return u
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	// Railway private network — TLS is not used on *.railway.internal
	if strings.Contains(lower, ".railway.internal") {
		return u + sep + "sslmode=disable"
	}
	// Public Railway proxy hostnames
	if strings.Contains(lower, "railway.app") || strings.Contains(lower, "rlwy.net") {
		return u + sep + "sslmode=require"
	}
	return u
}
