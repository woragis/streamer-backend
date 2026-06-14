package store

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/dedup"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/platform"
)

func TestIngestMessageDedup(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	ctx := context.Background()
	database := testutil.Open(t)

	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	st.SetDedup(dedup.New(goredis.NewClient(&goredis.Options{Addr: mr.Addr()})))

	roomID := defaults.DefaultRoomID
	in := platform.IngestMessageInput{
		Platform:   "youtube",
		Username:   "viewer1",
		Content:    "hello",
		ExternalID: "yt-001",
	}

	first, err := st.IngestMessage(ctx, roomID, in)
	if err != nil || first.Duplicate {
		t.Fatalf("expected first ingest ok, got %+v err=%v", first, err)
	}

	second, err := st.IngestMessage(ctx, roomID, in)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatal("expected duplicate on replay")
	}

	msgs, err := st.ListMessages(ctx, roomID, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message in db, got %d", len(msgs))
	}
}
