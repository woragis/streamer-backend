package store

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/platform"
)

func TestPlatformSettingsCRUD(t *testing.T) {
	ctx := context.Background()
	database := testutil.Open(t)
	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roomID := defaults.DefaultRoomID

	settings, err := st.GetPlatformSettings(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.YouTube.Enabled || settings.Kick.Enabled {
		t.Fatalf("expected disabled defaults, got %+v", settings)
	}

	enabled := true
	idle := 45
	apiKey := "secret-key"
	channel := "UC123"
	slug := "mystream"
	skip := false

	updated, err := st.UpdatePlatformSettings(ctx, roomID, platform.UpdatePlatformSettingsInput{
		YouTube: &platform.UpdateYouTubeSettingsInput{
			Enabled:     &enabled,
			APIKey:      &apiKey,
			ChannelID:   &channel,
			IdleSeconds: &idle,
		},
		Kick: &platform.UpdateKickSettingsInput{
			Enabled:           &enabled,
			ChannelSlug:       &slug,
			WebhookSkipVerify: &skip,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.YouTube.Enabled || !updated.YouTube.HasAPIKey {
		t.Fatalf("expected youtube configured publicly, got %+v", updated.YouTube)
	}
	if updated.YouTube.ChannelID != channel {
		t.Fatalf("unexpected channel id: %q", updated.YouTube.ChannelID)
	}

	resolved, err := st.GetPlatformSettingsResolved(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GoogleAPIKey != apiKey {
		t.Fatalf("expected secret api key in resolved settings")
	}
	if !resolved.YouTubeReady() {
		t.Fatal("expected youtube ready")
	}
}

func TestResolveKickSettingsBySlug(t *testing.T) {
	ctx := context.Background()
	database := testutil.Open(t)
	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roomID := defaults.DefaultRoomID

	enabled := true
	slug := "mystream"
	_, err := st.UpdatePlatformSettings(ctx, roomID, platform.UpdatePlatformSettingsInput{
		Kick: &platform.UpdateKickSettingsInput{
			Enabled:     &enabled,
			ChannelSlug: &slug,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resolved, ok, err := st.ResolveKickSettings(ctx, "mystream")
	if err != nil || !ok {
		t.Fatalf("expected kick match, ok=%v err=%v", ok, err)
	}
	if resolved.RoomID != roomID {
		t.Fatalf("expected room %s, got %s", roomID, resolved.RoomID)
	}

	_, ok, err = st.ResolveKickSettings(ctx, "other")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no match for other slug")
	}
}

func TestListYouTubeReadyRooms(t *testing.T) {
	ctx := context.Background()
	database := testutil.Open(t)
	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	enabled := true
	apiKey := "key"
	channel := "UC999"
	_, err := st.UpdatePlatformSettings(ctx, defaults.DefaultRoomID, platform.UpdatePlatformSettingsInput{
		YouTube: &platform.UpdateYouTubeSettingsInput{
			Enabled:   &enabled,
			APIKey:    &apiKey,
			ChannelID: &channel,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rooms, err := st.ListYouTubeReadyRooms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rooms) != 1 || rooms[0].RoomID != defaults.DefaultRoomID {
		t.Fatalf("expected one ready room, got %+v", rooms)
	}
}
