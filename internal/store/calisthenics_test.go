package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/woragis/streamer-backend/internal/calisthenics"
	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestCalisthenicsMigrationAndSetActions(t *testing.T) {
	t.Parallel()

	databaseURL := filepath.Join(t.TempDir(), "cal.db")
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

	state, err := st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(state.Exercises))
	}
	if len(state.Exercises[0].SetDetails) != 5 {
		t.Fatalf("expected 5 sets on first exercise, got %d", len(state.Exercises[0].SetDetails))
	}

	activeSet := findCurrentSet(state.Exercises[0].SetDetails)
	if activeSet == nil {
		t.Fatal("expected a current set")
	}

	updated, err := st.IncrementRep(ctx, defaults.DefaultRoomID, activeSet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RepsCompleted != activeSet.RepsCompleted+1 {
		t.Fatalf("expected rep increment, got %d", updated.RepsCompleted)
	}

	_, err = st.CompleteSet(ctx, defaults.DefaultRoomID, updated.ID)
	if err != nil {
		t.Fatal(err)
	}

	state2, err := st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if state2.Exercises[0].CompletedSets < state.Exercises[0].CompletedSets+1 {
		t.Fatalf("expected completed sets to increase")
	}
}

func TestCreateWorkoutWithExercise(t *testing.T) {
	t.Parallel()

	databaseURL := filepath.Join(t.TempDir(), "cal2.db")
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

	w, err := st.CreateWorkout(ctx, defaults.DefaultRoomID, calisthenics.CreateWorkoutInput{
		WorkoutType: "PUSH DAY",
	})
	if err != nil {
		t.Fatal(err)
	}

	ex, err := st.CreateExercise(ctx, defaults.DefaultRoomID, w.ID, calisthenics.CreateExerciseInput{
		Name: "DIPS", PlannedSets: 3, RepTarget: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ex.SetDetails) != 3 {
		t.Fatalf("expected 3 sets, got %d", len(ex.SetDetails))
	}
}

func findCurrentSet(sets []calisthenics.Set) *calisthenics.Set {
	for i := range sets {
		st := calisthenics.DeriveSetStatus(sets[i])
		if st != calisthenics.SetCompleted && st != calisthenics.SetSkipped {
			return &sets[i]
		}
	}
	return nil
}
