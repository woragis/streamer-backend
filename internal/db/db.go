package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS rooms (
	id TEXT PRIMARY KEY,
	active_domain TEXT NOT NULL DEFAULT 'leetcode',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS room_documents (
	room_id TEXT NOT NULL,
	doc_key TEXT NOT NULL,
	data JSON NOT NULL,
	revision INTEGER NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (room_id, doc_key),
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS cal_runtime (
	room_id TEXT PRIMARY KEY,
	active_workout_id TEXT,
	today_goal_label TEXT NOT NULL DEFAULT 'COMPLETE THE WORKOUT',
	today_goal_progress INTEGER NOT NULL DEFAULT 0,
	timers JSON NOT NULL DEFAULT '{}',
	revision INTEGER NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS cal_workouts (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	workout_type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	live_session_id TEXT,
	started_at TEXT,
	ended_at TEXT,
	created_at TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS cal_workout_exercises (
	id TEXT PRIMARY KEY,
	workout_id TEXT NOT NULL,
	name TEXT NOT NULL,
	movement_id TEXT,
	planned_sets INTEGER NOT NULL,
	rep_target INTEGER NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending',
	sort_order INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (workout_id) REFERENCES cal_workouts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS cal_sets (
	id TEXT PRIMARY KEY,
	exercise_id TEXT NOT NULL,
	set_number INTEGER NOT NULL,
	reps_target INTEGER NOT NULL,
	reps_completed INTEGER NOT NULL DEFAULT 0,
	skipped INTEGER NOT NULL DEFAULT 0,
	completed_at TEXT,
	FOREIGN KEY (exercise_id) REFERENCES cal_workout_exercises(id) ON DELETE CASCADE,
	UNIQUE(exercise_id, set_number)
);
`

func Open(databaseURL string) (*sql.DB, error) {
	if err := ensureDir(databaseURL); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return db, nil
}

func ensureDir(databaseURL string) error {
	dir := filepath.Dir(databaseURL)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
