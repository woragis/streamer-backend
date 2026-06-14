package calisthenics

import "time"

func DeriveSetStatus(s Set) SetStatus {
	if s.Skipped {
		return SetSkipped
	}
	if s.CompletedAt != nil || s.RepsCompleted >= s.RepsTarget {
		return SetCompleted
	}
	return SetPending
}

func BuildExerciseView(ex Exercise, sets []Set) ExerciseView {
	completedSets := 0
	totalReps := 0
	repsInCurrent := 0
	foundCurrent := false

	details := make([]Set, 0, len(sets))
	for _, s := range sets {
		s.Status = DeriveSetStatus(s)
		details = append(details, s)
		if s.Skipped {
			continue
		}
		totalReps += s.RepsCompleted
		if s.Status == SetCompleted {
			completedSets++
			continue
		}
		if !foundCurrent {
			repsInCurrent = s.RepsCompleted
			foundCurrent = true
		}
	}

	return ExerciseView{
		ID:               ex.ID,
		Name:             ex.Name,
		Sets:             ex.PlannedSets,
		RepTarget:        ex.RepTarget,
		CompletedSets:    completedSets,
		RepsInCurrentSet: repsInCurrent,
		TotalReps:        totalReps,
		Status:           string(ex.Status),
		Order:            ex.SortOrder,
		MovementID:       ex.MovementID,
		SetDetails:       details,
	}
}

func ComputeGoalProgress(exercises []Exercise) int {
	if len(exercises) == 0 {
		return 0
	}
	done := 0
	for _, ex := range exercises {
		if ex.Status == ExerciseDone {
			done++
		}
	}
	return (done * 100) / len(exercises)
}

func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
