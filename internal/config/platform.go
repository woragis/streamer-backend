package config

import (
	"os"
	"strings"

	"github.com/woragis/streamer-backend/internal/defaults"
)

// WorkerRoomID is the default room watched by cmd/worker when no DB rows match yet.
func WorkerRoomID() string {
	roomID := strings.TrimSpace(os.Getenv("ROOM_ID"))
	if roomID == "" {
		return defaults.DefaultRoomID
	}
	return roomID
}
