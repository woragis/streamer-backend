package store_test

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/leetcode"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestLeetCodeMigrationAndProblemFlow(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetLeetCodeState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Problems) == 0 {
		t.Fatal("expected problems after seed")
	}

	pid := state.Problems[0].ID
	if _, err := st.ActivateProblem(ctx, defaults.DefaultRoomID, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SolveProblem(ctx, defaults.DefaultRoomID, pid); err != nil {
		t.Fatal(err)
	}

	stats, err := st.GetLeetCodeStats(ctx, defaults.DefaultRoomID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if stats.SolvedCount < 1 {
		t.Fatalf("expected at least 1 solved, got %d", stats.SolvedCount)
	}
}

func TestLeetCodeStreak(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	streak, err := st.GetLeetCodeStreak(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if streak.Streak < 0 {
		t.Fatalf("invalid streak %d", streak.Streak)
	}
}

func TestLeetCodeSessions(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	sess, err := st.CreateLiveSession(ctx, defaults.DefaultRoomID, leetcode.CreateLiveSessionInput{
		Domain: "leetcode", Platforms: []string{"youtube"},
	})
	if err != nil {
		t.Fatal(err)
	}

	stats, err := st.GetLeetCodeStats(ctx, defaults.DefaultRoomID, "", sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = stats
}
