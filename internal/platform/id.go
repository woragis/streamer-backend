package platform

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/woragis/streamer-backend/internal/defaults"
)

func NewID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func DefaultBotRules(roomID string) []BotRule {
	now := NowISO()
	scene := func(s string) json.RawMessage {
		b, _ := json.Marshal(map[string]string{"scene": s})
		return b
	}
	scope := defaults.ScopedSeedID
	return []BotRule{
		{ID: scope(roomID, "rule-brb"), RoomID: roomID, Name: "BRB command", Enabled: true, TriggerType: "keyword", TriggerValue: "!brb", ActionType: "set_scene", ActionPayload: scene("brb"), CreatedAt: now},
		{ID: scope(roomID, "rule-live"), RoomID: roomID, Name: "Live command", Enabled: true, TriggerType: "keyword", TriggerValue: "!live", ActionType: "set_scene", ActionPayload: scene("live"), CreatedAt: now},
		{ID: scope(roomID, "rule-whiteboard"), RoomID: roomID, Name: "Whiteboard command", Enabled: true, TriggerType: "keyword", TriggerValue: "!whiteboard", ActionType: "set_scene", ActionPayload: scene("whiteboard"), CreatedAt: now},
	}
}
