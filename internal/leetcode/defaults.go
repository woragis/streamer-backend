package leetcode

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func NewID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func DefaultTimers() map[string]any {
	timer := func(id, mode, label string, durationSeconds int) map[string]any {
		return map[string]any{
			"id": id, "mode": mode, "label": label,
			"durationSeconds": durationSeconds, "accumulatedSeconds": 0,
			"running": false, "startedAt": nil, "endsAt": nil,
		}
	}
	return map[string]any{
		"startingSoon": timer("startingSoon", "countdown", "Starting Soon", 300),
		"brb":          timer("brb", "countdown", "BRB", 300),
		"focus":        timer("focus", "countdown", "Focus", 1500),
	}
}
