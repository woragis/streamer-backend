package leetcode

import "encoding/json"

type Difficulty string
type ProblemStatus string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"

	StatusQueued  ProblemStatus = "queued"
	StatusActive  ProblemStatus = "active"
	StatusSolved  ProblemStatus = "solved"
	StatusSkipped ProblemStatus = "skipped"
)

type LiveSession struct {
	ID        string   `json:"id"`
	RoomID    string   `json:"roomId"`
	Domain    string   `json:"domain"`
	Title     *string  `json:"title,omitempty"`
	Platforms []string `json:"platforms"`
	StartedAt string   `json:"startedAt"`
	EndedAt   *string  `json:"endedAt,omitempty"`
}

type PlanItem struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Done      bool   `json:"done"`
	SortOrder int    `json:"order"`
}

type Problem struct {
	ID          int           `json:"id"`
	Title       string        `json:"title"`
	Difficulty  Difficulty    `json:"difficulty"`
	Description string        `json:"description"`
	Status      ProblemStatus `json:"status"`
	SolvedAt    *string       `json:"solvedAt"`
	SortOrder   int           `json:"order"`
}

type CodeEditor struct {
	FileName string `json:"fileName"`
	Content  string `json:"content"`
}

type Whiteboard struct {
	Title    string   `json:"title"`
	Bullets  []string `json:"bullets"`
	Notes    []string `json:"notes"`
	Approach string   `json:"approach"`
}

type Goals struct {
	DailyTarget  int `json:"dailyTarget"`
	WeeklyTarget int `json:"weeklyTarget"`
	Streak       int `json:"streak"`
}

type Copy struct {
	StartingSoonSubtext string `json:"startingSoonSubtext"`
	BrbSubtext        string `json:"brbSubtext"`
	BrbMessage        string `json:"brbMessage"`
	UpNextLabel       string `json:"upNextLabel"`
}

type State struct {
	Revision            int64           `json:"revision"`
	ActiveLiveSessionID *string         `json:"activeLiveSessionId,omitempty"`
	Plan                []PlanItem      `json:"plan"`
	Problems            []Problem       `json:"problems"`
	Code                CodeEditor      `json:"code"`
	Whiteboard          Whiteboard      `json:"whiteboard"`
	Goals               Goals           `json:"goals"`
	Copy                Copy            `json:"copy"`
	LoadingProgress     int             `json:"loadingProgress"`
	Timers              json.RawMessage `json:"timers"`
}

type ProblemAttempt struct {
	ID            string  `json:"id"`
	RoomID        string  `json:"roomId"`
	ProblemID     int     `json:"problemId"`
	LiveSessionID *string `json:"liveSessionId,omitempty"`
	StartedAt     string  `json:"startedAt"`
	SolvedAt      *string `json:"solvedAt,omitempty"`
	Notes         *string `json:"notes,omitempty"`
}

type TopicTag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Stats struct {
	SolvedCount   int   `json:"solvedCount"`
	SkippedCount  int   `json:"skippedCount"`
	ActiveCount   int   `json:"activeCount"`
	ProblemIDs    []int `json:"problemIds,omitempty"`
	Streak        int   `json:"streak,omitempty"`
	Month         string `json:"month,omitempty"`
	LiveSessionID string `json:"liveSessionId,omitempty"`
}

type CreateLiveSessionInput struct {
	Domain    string   `json:"domain"`
	Title     *string  `json:"title,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
}

type UpdateLiveSessionInput struct {
	Title   *string `json:"title,omitempty"`
	EndedAt *string `json:"endedAt,omitempty"`
}

type CreatePlanItemInput struct {
	Label     string `json:"label"`
	SortOrder *int   `json:"order,omitempty"`
}

type UpdatePlanItemInput struct {
	Label     *string `json:"label,omitempty"`
	Done      *bool   `json:"done,omitempty"`
	SortOrder *int    `json:"order,omitempty"`
}

type CreateProblemInput struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Difficulty  Difficulty `json:"difficulty"`
	Description string     `json:"description,omitempty"`
	SortOrder   *int       `json:"order,omitempty"`
}

type UpdateProblemInput struct {
	Title       *string        `json:"title,omitempty"`
	Difficulty  *Difficulty    `json:"difficulty,omitempty"`
	Description *string        `json:"description,omitempty"`
	Status      *ProblemStatus `json:"status,omitempty"`
	SortOrder   *int           `json:"order,omitempty"`
}

type FlatDocument struct {
	Plan            []PlanItem      `json:"plan"`
	Problems        []Problem       `json:"problems"`
	Code            CodeEditor      `json:"code"`
	Whiteboard      Whiteboard      `json:"whiteboard"`
	Goals           Goals           `json:"goals"`
	Copy            Copy            `json:"copy"`
	LoadingProgress int             `json:"loadingProgress"`
	Timers          json.RawMessage `json:"timers"`
}
