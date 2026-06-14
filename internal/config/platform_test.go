package config

import "testing"

func TestLoadPlatformDefaults(t *testing.T) {
	t.Setenv("ROOM_ID", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("YOUTUBE_CHANNEL_ID", "")
	t.Setenv("KICK_CHANNEL_SLUG", "")

	cfg := LoadPlatform()
	if cfg.RoomID != "default" {
		t.Fatalf("expected default room, got %q", cfg.RoomID)
	}
	if cfg.YouTubeEnabled {
		t.Fatal("youtube should be disabled without credentials")
	}
	if cfg.KickEnabled {
		t.Fatal("kick should be disabled without slug")
	}
}

func TestLoadPlatformYouTubeAutoEnable(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key")
	t.Setenv("YOUTUBE_CHANNEL_ID", "UC123")

	cfg := LoadPlatform()
	if !cfg.YouTubeEnabled {
		t.Fatal("expected youtube enabled when key and channel set")
	}
}

func TestLoadPlatformKickAutoEnable(t *testing.T) {
	t.Setenv("KICK_CHANNEL_SLUG", "mychannel")

	cfg := LoadPlatform()
	if !cfg.KickEnabled {
		t.Fatal("expected kick enabled when slug set")
	}
}
