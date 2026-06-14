package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/woragis/streamer-backend/internal/db"
)

func DefaultURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	if u := os.Getenv("DATABASE_URL"); u != "" {
		return u
	}
	return "postgres://streamer:streamer@localhost:5432/streamer?sslmode=disable"
}

func Open(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(DefaultURL())
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	Reset(t, database)
	return database
}

func Reset(t *testing.T, database *db.DB) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"core_bot_rules", "core_stream_events", "core_messages", "core_users",
		"cal_skill_practice_logs", "cal_skill_acquisitions", "cal_movement_proficiencies",
		"cal_movements", "cal_movement_categories", "lc_problem_topics", "lc_topic_tags",
		"lc_problem_attempts", "lc_problems", "lc_plan_items", "lc_runtime",
		"live_sessions", "cal_sets", "cal_workout_exercises", "cal_workouts", "cal_runtime",
		"room_documents", "rooms",
	}
	for _, table := range tables {
		if _, err := database.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("reset %s: %v", table, err)
		}
	}
}
