-- +goose Up
CREATE INDEX IF NOT EXISTS idx_core_messages_room_created ON core_messages(room_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_core_messages_room_user ON core_messages(room_id, user_id);
CREATE INDEX IF NOT EXISTS idx_lc_problems_room_status ON lc_problems(room_id, status);
CREATE INDEX IF NOT EXISTS idx_lc_attempts_room_session ON lc_problem_attempts(room_id, live_session_id);
CREATE INDEX IF NOT EXISTS idx_cal_acquisitions_room_date ON cal_skill_acquisitions(room_id, acquired_at);
CREATE INDEX IF NOT EXISTS idx_live_sessions_room_started ON live_sessions(room_id, started_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_live_sessions_room_started;
DROP INDEX IF EXISTS idx_cal_acquisitions_room_date;
DROP INDEX IF EXISTS idx_lc_attempts_room_session;
DROP INDEX IF EXISTS idx_lc_problems_room_status;
DROP INDEX IF EXISTS idx_core_messages_room_user;
DROP INDEX IF EXISTS idx_core_messages_room_created;
