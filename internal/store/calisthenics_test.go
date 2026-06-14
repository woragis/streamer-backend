package store_test

import (
	"context"
	"testing"

	"github.com/woragis/streamer-backend/internal/calisthenics"
	"github.com/woragis/streamer-backend/internal/db/testutil"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestCalisthenicsMigrationAndSetActions(t *testing.T) {
	sqlDB := testutil.Open(t)

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	state, err := st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Exercises) == 0 {
		t.Fatal("expected exercises after seed")
	}

	current := findCurrentSet(state.Exercises[0].SetDetails)
	if current == nil {
		t.Fatal("expected active set")
	}

	_, err = st.IncrementRep(ctx, defaults.DefaultRoomID, current.ID)
	if err != nil {
		t.Fatal(err)
	}

	state, err = st.GetCalisthenicsState(ctx, defaults.DefaultRoomID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ex := range state.Exercises {
		for _, set := range ex.SetDetails {
			if set.ID == current.ID && set.RepsCompleted > 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected incremented rep in state")
	}
}

func TestCreateWorkoutWithExercise(t *testing.T) {
	sqlDB := testutil.Open(t)

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
