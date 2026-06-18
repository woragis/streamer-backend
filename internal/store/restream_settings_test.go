package store_test

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/restream"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestRestreamPublishAuth(t *testing.T) {
	sqlDB := testutil.Open(t)
	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	settings, err := st.RegenerateRestreamIngestKey(ctx, defaults.RoomCodes)
	if err != nil {
		t.Fatal(err)
	}
	if settings.IngestKey == "" {
		t.Fatal("expected ingest key")
	}

	_, err = st.UpdateRestreamSettings(ctx, defaults.RoomCodes, restream.UpdateInput{
		Enabled: ptrBool(true),
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, err := st.ValidateRestreamPublish(ctx, "live/codes", settings.IngestKey)
	if err != nil || !ok {
		t.Fatalf("expected publish ok, got ok=%v err=%v", ok, err)
	}

	ok, err = st.ValidateRestreamPublish(ctx, "live/codes", "wrong")
	if err != nil || ok {
		t.Fatalf("expected publish denied, got ok=%v err=%v", ok, err)
	}
}

func ptrBool(v bool) *bool { return &v }
