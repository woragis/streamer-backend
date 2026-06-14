package calisthenics_test

import (
	"testing"

	"github.com/woragis/streamer-backend/internal/calisthenics"
)

func TestBuildExerciseView(t *testing.T) {
	ex := calisthenics.Exercise{
		ID: "ex-1", Name: "PULL-UPS", PlannedSets: 3, RepTarget: 10,
		Status: calisthenics.ExerciseActive, SortOrder: 0,
	}
	sets := []calisthenics.Set{
		{SetNumber: 1, RepsTarget: 10, RepsCompleted: 10, CompletedAt: ptr("2025-01-01T00:00:00Z")},
		{SetNumber: 2, RepsTarget: 10, RepsCompleted: 8},
		{SetNumber: 3, RepsTarget: 10, RepsCompleted: 0},
	}
	view := calisthenics.BuildExerciseView(ex, sets)
	if view.CompletedSets != 1 {
		t.Fatalf("completedSets=%d want 1", view.CompletedSets)
	}
	if view.RepsInCurrentSet != 8 {
		t.Fatalf("repsInCurrentSet=%d want 8", view.RepsInCurrentSet)
	}
	if view.TotalReps != 18 {
		t.Fatalf("totalReps=%d want 18", view.TotalReps)
	}
}

func TestSetsFromFlat(t *testing.T) {
	flat := calisthenics.FlatExercise{
		ID: "ex-1", Sets: 3, RepTarget: 10,
		CompletedSets: 1, RepsInCurrentSet: 5,
	}
	sets := calisthenics.SetsFromFlat(flat)
	if len(sets) != 3 {
		t.Fatalf("len=%d want 3", len(sets))
	}
	if sets[0].RepsCompleted != 10 {
		t.Fatalf("set1 reps=%d want 10", sets[0].RepsCompleted)
	}
	if sets[1].RepsCompleted != 5 {
		t.Fatalf("set2 reps=%d want 5", sets[1].RepsCompleted)
	}
}

func ptr(s string) *string { return &s }
