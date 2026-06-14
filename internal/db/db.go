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

CREATE TABLE IF NOT EXISTS live_sessions (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	domain TEXT NOT NULL DEFAULT 'leetcode',
	title TEXT,
	platforms JSON NOT NULL DEFAULT '[]',
	started_at TEXT NOT NULL,
	ended_at TEXT,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS lc_runtime (
	room_id TEXT PRIMARY KEY,
	active_live_session_id TEXT,
	code JSON NOT NULL DEFAULT '{}',
	whiteboard JSON NOT NULL DEFAULT '{}',
	goals JSON NOT NULL DEFAULT '{}',
	copy JSON NOT NULL DEFAULT '{}',
	loading_progress INTEGER NOT NULL DEFAULT 0,
	timers JSON NOT NULL DEFAULT '{}',
	revision INTEGER NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS lc_plan_items (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	label TEXT NOT NULL,
	done INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS lc_problems (
	room_id TEXT NOT NULL,
	problem_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	difficulty TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'queued',
	solved_at TEXT,
	sort_order INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (room_id, problem_id),
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS lc_problem_attempts (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	problem_id INTEGER NOT NULL,
	live_session_id TEXT,
	started_at TEXT NOT NULL,
	solved_at TEXT,
	notes TEXT,
	FOREIGN KEY (room_id) REFERENCES rooms(id),
	FOREIGN KEY (live_session_id) REFERENCES live_sessions(id)
);

CREATE TABLE IF NOT EXISTS lc_topic_tags (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	name TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS lc_problem_topics (
	room_id TEXT NOT NULL,
	problem_id INTEGER NOT NULL,
	topic_id TEXT NOT NULL,
	PRIMARY KEY (room_id, problem_id, topic_id)
);

CREATE TABLE IF NOT EXISTS cal_movement_categories (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	name TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS cal_movements (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	slug TEXT NOT NULL,
	name TEXT NOT NULL,
	category_id TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	prerequisites JSON NOT NULL DEFAULT '[]',
	FOREIGN KEY (room_id) REFERENCES rooms(id),
	UNIQUE(room_id, slug)
);

CREATE TABLE IF NOT EXISTS cal_movement_proficiencies (
	room_id TEXT NOT NULL,
	movement_id TEXT NOT NULL,
	level TEXT NOT NULL DEFAULT 'unknown',
	notes TEXT NOT NULL DEFAULT '',
	best_hold_seconds INTEGER,
	best_reps INTEGER,
	progression_variant TEXT,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (room_id, movement_id),
	FOREIGN KEY (movement_id) REFERENCES cal_movements(id)
);

CREATE TABLE IF NOT EXISTS cal_skill_acquisitions (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	movement_id TEXT NOT NULL,
	live_session_id TEXT,
	acquired_at TEXT NOT NULL,
	proficiency_before TEXT NOT NULL,
	proficiency_after TEXT NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	evidence_url TEXT,
	acknowledged INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (room_id) REFERENCES rooms(id),
	FOREIGN KEY (movement_id) REFERENCES cal_movements(id)
);

CREATE TABLE IF NOT EXISTS cal_skill_practice_logs (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	movement_id TEXT NOT NULL,
	live_session_id TEXT,
	practiced_at TEXT NOT NULL,
	duration_seconds INTEGER NOT NULL DEFAULT 0,
	notes TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

CREATE TABLE IF NOT EXISTS core_users (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	platform TEXT NOT NULL,
	username TEXT NOT NULL,
	display_name TEXT NOT NULL DEFAULT '',
	first_seen_at TEXT NOT NULL,
	last_seen_at TEXT NOT NULL,
	UNIQUE(room_id, platform, username)
);

CREATE TABLE IF NOT EXISTS core_messages (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	live_session_id TEXT,
	platform TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at TEXT NOT NULL,
	deleted_at TEXT,
	FOREIGN KEY (user_id) REFERENCES core_users(id)
);

CREATE TABLE IF NOT EXISTS core_stream_events (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	live_session_id TEXT,
	event_type TEXT NOT NULL,
	platform TEXT NOT NULL DEFAULT '',
	username TEXT NOT NULL DEFAULT '',
	payload JSON NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS core_bot_rules (
	id TEXT PRIMARY KEY,
	room_id TEXT NOT NULL,
	name TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	trigger_type TEXT NOT NULL DEFAULT 'keyword',
	trigger_value TEXT NOT NULL,
	action_type TEXT NOT NULL,
	action_payload JSON NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL
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
