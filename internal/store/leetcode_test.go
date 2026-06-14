package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/leetcode"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestLeetCodeMigrationAndProblemFlow(t *testing.T) {
	t.Parallel()

	databaseURL := filepath.Join(t.TempDir(), "lc.db")
	sqlDB, err := db.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetLeetCodeState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Problems) != 4 {
		t.Fatalf("expected 4 problems, got %d", len(state.Problems))
	}
	if len(state.Plan) != 3 {
		t.Fatalf("expected 3 plan items, got %d", len(state.Plan))
	}

	sess, err := st.CreateLiveSession(ctx, defaults.DefaultRoomID, leetcode.CreateLiveSessionInput{
		Domain: "leetcode", Platforms: []string{"youtube"},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = st.ActivateProblem(ctx, defaults.DefaultRoomID, 239)
	if err != nil {
		t.Fatal(err)
	}

	_, err = st.SolveProblem(ctx, defaults.DefaultRoomID, 239)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := st.GetLeetCodeStats(ctx, defaults.DefaultRoomID, "", sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stats.SolvedCount != 1 {
		t.Fatalf("expected 1 solved in session, got %d", stats.SolvedCount)
	}

	attempts, err := st.ListProblemAttempts(ctx, defaults.DefaultRoomID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) == 0 {
		t.Fatal("expected at least one attempt")
	}
	if attempts[0].SolvedAt == nil {
		t.Fatal("expected attempt to be solved")
	}
}

func TestLeetCodeStreak(t *testing.T) {
	t.Parallel()

	databaseURL := filepath.Join(t.TempDir(), "lc2.db")
	sqlDB, err := db.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

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
