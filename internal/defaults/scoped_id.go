package defaults

import "strings"

// ScopedSeedID prefixes seed row IDs with the room so multi-room seed stays idempotent.
func ScopedSeedID(roomID, id string) string {
	if roomID == "" || id == "" {
		return id
	}
	prefix := roomID + ":"
	if strings.HasPrefix(id, prefix) {
		return id
	}
	return prefix + id
}
