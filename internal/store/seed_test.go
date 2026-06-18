package store_test

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestSeedAllRooms(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	if err := st.Seed(ctx); err != nil {
		t.Fatalf("second seed should be idempotent: %v", err)
	}

	for _, roomID := range []string{defaults.DefaultRoomID, defaults.RoomCodes, defaults.RoomCalisthenics} {
		exists, err := st.RoomExists(ctx, roomID)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("room %s not seeded", roomID)
		}
		state, err := st.GetLeetCodeState(ctx, roomID)
		if err != nil {
			t.Fatalf("room %s leetcode: %v", roomID, err)
		}
		if len(state.Plan) == 0 || len(state.Problems) == 0 {
			t.Fatalf("room %s missing leetcode seed data", roomID)
		}
	}
}
