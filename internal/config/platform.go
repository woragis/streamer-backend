package config

import (
	"os"
	"strings"

	"github.com/woragis/streamer-backend/internal/defaults"
)

type PlatformConfig struct {
	RoomID string

	YouTubeEnabled     bool
	GoogleAPIKey       string
	YouTubeChannelID   string
	YouTubeIdleSeconds int

	KickEnabled            bool
	KickChannelSlug        string
	KickWebhookSkipVerify  bool
}

func LoadPlatform() PlatformConfig {
	roomID := strings.TrimSpace(os.Getenv("ROOM_ID"))
	if roomID == "" {
		roomID = defaults.DefaultRoomID
	}

	youtubeEnabled := envTruthy("YOUTUBE_ENABLED")
	googleAPIKey := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	youtubeChannelID := strings.TrimSpace(os.Getenv("YOUTUBE_CHANNEL_ID"))
	if googleAPIKey != "" && youtubeChannelID != "" {
		youtubeEnabled = true
	}

	idleSeconds := envInt("YOUTUBE_IDLE_SECONDS", 30)

	kickEnabled := envTruthy("KICK_ENABLED")
	kickSlug := strings.TrimSpace(os.Getenv("KICK_CHANNEL_SLUG"))
	if kickSlug != "" {
		kickEnabled = true
	}

	return PlatformConfig{
		RoomID:               roomID,
		YouTubeEnabled:       youtubeEnabled,
		GoogleAPIKey:         googleAPIKey,
		YouTubeChannelID:     youtubeChannelID,
		YouTubeIdleSeconds:   idleSeconds,
		KickEnabled:          kickEnabled,
		KickChannelSlug:      kickSlug,
		KickWebhookSkipVerify: envTruthy("KICK_WEBHOOK_SKIP_VERIFY"),
	}
}

func envTruthy(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "true" || v == "1" || v == "yes"
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	var n int
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return fallback
	}
	return n
}
