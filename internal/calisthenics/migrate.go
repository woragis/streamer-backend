package calisthenics

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

func NewID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

// FlatExercise is the Phase A JSON exercise shape.
type FlatExercise struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Sets             int    `json:"sets"`
	RepTarget        int    `json:"repTarget"`
	CompletedSets    int    `json:"completedSets"`
	RepsInCurrentSet int    `json:"repsInCurrentSet"`
	TotalReps        int    `json:"totalReps"`
	Status           string `json:"status"`
	Order            int    `json:"order"`
}

type FlatDocument struct {
	WorkoutType string         `json:"workoutType"`
	Exercises   []FlatExercise `json:"exercises"`
	TodayGoal   TodayGoal      `json:"todayGoal"`
	Timers      json.RawMessage `json:"timers"`
}

func SetsFromFlat(ex FlatExercise) []Set {
	sets := make([]Set, 0, ex.Sets)
	for i := 1; i <= ex.Sets; i++ {
		repsCompleted := 0
		skipped := false
		var completedAt *string

		if i <= ex.CompletedSets {
			repsCompleted = ex.RepTarget
			ts := NowISO()
			completedAt = &ts
		} else if i == ex.CompletedSets+1 {
			repsCompleted = ex.RepsInCurrentSet
		}

		sets = append(sets, Set{
			ID:            NewID("set"),
			ExerciseID:    ex.ID,
			SetNumber:     i,
			RepsTarget:    ex.RepTarget,
			RepsCompleted: repsCompleted,
			Skipped:       skipped,
			CompletedAt:   completedAt,
			Status:        DeriveSetStatus(Set{RepsTarget: ex.RepTarget, RepsCompleted: repsCompleted, Skipped: skipped, CompletedAt: completedAt}),
		})
	}
	return sets
}

func ExerciseFromFlat(ex FlatExercise, workoutID string) Exercise {
	status := ExerciseStatus(ex.Status)
	if status == "" {
		status = ExercisePending
	}
	return Exercise{
		ID:          ex.ID,
		WorkoutID:   workoutID,
		Name:        ex.Name,
		PlannedSets: ex.Sets,
		RepTarget:   ex.RepTarget,
		Status:      status,
		SortOrder:   ex.Order,
	}
}
