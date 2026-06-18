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
	if strings.Contains(lower, "railway") || strings.Contains(lower, "rlwy.net") {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		return u + sep + "sslmode=require"
	}
	return u
}
