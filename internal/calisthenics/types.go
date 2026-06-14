package calisthenics

import "encoding/json"

type WorkoutStatus string
type ExerciseStatus string
type SetStatus string

const (
	WorkoutPlanned   WorkoutStatus = "planned"
	WorkoutActive    WorkoutStatus = "active"
	WorkoutCompleted WorkoutStatus = "completed"

	ExercisePending ExerciseStatus = "pending"
	ExerciseActive  ExerciseStatus = "active"
	ExerciseDone    ExerciseStatus = "done"

	SetPending   SetStatus = "pending"
	SetActive    SetStatus = "active"
	SetCompleted SetStatus = "completed"
	SetSkipped   SetStatus = "skipped"
)

type Workout struct {
	ID            string        `json:"id"`
	RoomID        string        `json:"roomId"`
	WorkoutType   string        `json:"workoutType"`
	Status        WorkoutStatus `json:"status"`
	LiveSessionID *string       `json:"liveSessionId,omitempty"`
	StartedAt     *string       `json:"startedAt,omitempty"`
	EndedAt       *string       `json:"endedAt,omitempty"`
	CreatedAt     string        `json:"createdAt"`
}

type Exercise struct {
	ID          string         `json:"id"`
	WorkoutID   string         `json:"workoutId"`
	Name        string         `json:"name"`
	MovementID  *string        `json:"movementId,omitempty"`
	PlannedSets int            `json:"plannedSets"`
	RepTarget   int            `json:"repTarget"`
	Status      ExerciseStatus `json:"status"`
	SortOrder   int            `json:"order"`
}

type Set struct {
	ID            string    `json:"id"`
	ExerciseID    string    `json:"exerciseId"`
	SetNumber     int       `json:"setNumber"`
	RepsTarget    int       `json:"repsTarget"`
	RepsCompleted int       `json:"repsCompleted"`
	Skipped       bool      `json:"skipped"`
	CompletedAt   *string   `json:"completedAt,omitempty"`
	Status        SetStatus `json:"status"`
}

// ExerciseView is the overlay-friendly shape (flat + nested sets).
type ExerciseView struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Sets               int     `json:"sets"`
	RepTarget          int     `json:"repTarget"`
	CompletedSets      int     `json:"completedSets"`
	RepsInCurrentSet   int     `json:"repsInCurrentSet"`
	TotalReps          int     `json:"totalReps"`
	Status             string  `json:"status"`
	Order              int     `json:"order"`
	MovementID         *string `json:"movementId,omitempty"`
	SetDetails         []Set   `json:"setDetails"`
}

type TodayGoal struct {
	Label    string `json:"label"`
	Progress int    `json:"progress"`
}

type State struct {
	Revision      int64          `json:"revision"`
	WorkoutID     string         `json:"workoutId"`
	WorkoutType   string         `json:"workoutType"`
	WorkoutStatus string         `json:"workoutStatus"`
	Exercises     []ExerciseView `json:"exercises"`
	TodayGoal     TodayGoal      `json:"todayGoal"`
	Timers        json.RawMessage `json:"timers"`
}

type CreateWorkoutInput struct {
	WorkoutType   string  `json:"workoutType"`
	Status        string  `json:"status,omitempty"`
	LiveSessionID *string `json:"liveSessionId,omitempty"`
}

type CreateExerciseInput struct {
	Name        string  `json:"name"`
	PlannedSets int     `json:"plannedSets"`
	RepTarget   int     `json:"repTarget"`
	MovementID  *string `json:"movementId,omitempty"`
	SortOrder   *int    `json:"order,omitempty"`
}

type UpdateExerciseInput struct {
	Name        *string `json:"name,omitempty"`
	PlannedSets *int    `json:"plannedSets,omitempty"`
	RepTarget   *int    `json:"repTarget,omitempty"`
	MovementID  *string `json:"movementId,omitempty"`
	SortOrder   *int    `json:"order,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type CreateSetInput struct {
	SetNumber  int `json:"setNumber"`
	RepsTarget int `json:"repsTarget"`
}

type UpdateSetInput struct {
	RepsTarget    *int  `json:"repsTarget,omitempty"`
	RepsCompleted *int  `json:"repsCompleted,omitempty"`
	Skipped       *bool `json:"skipped,omitempty"`
}

type UpdateWorkoutInput struct {
	WorkoutType *string `json:"workoutType,omitempty"`
	Status      *string `json:"status,omitempty"`
	EndedAt     *string `json:"endedAt,omitempty"`
}
