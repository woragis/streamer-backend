package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/leetcode"
	"github.com/woragis/streamer-backend/internal/timers"
)

var ErrLCNoActiveAttempt = errors.New("no open attempt for problem")

type lcRuntime struct {
	ActiveLiveSessionID *string
	Code                json.RawMessage
	Whiteboard          json.RawMessage
	Goals               json.RawMessage
	Copy                json.RawMessage
	LoadingProgress     int
	Timers              json.RawMessage
	Revision            int64
}

func (s *Store) EnsureLeetCode(ctx context.Context, roomID string) error {
	var problems, planItems int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM lc_problems WHERE room_id = ?),
			(SELECT COUNT(*) FROM lc_plan_items WHERE room_id = ?)
	`, roomID, roomID).Scan(&problems, &planItems)
	if err != nil {
		return err
	}
	if problems > 0 || planItems > 0 {
		return s.ensureLCRuntime(ctx, roomID)
	}

	doc, err := s.GetDocument(ctx, roomID, DocLeetCode)
	if err == nil {
		return s.importLeetCodeDoc(ctx, roomID, doc.Data)
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}

	return s.seedLeetCode(ctx, roomID)
}

func (s *Store) ensureLCRuntime(ctx context.Context, roomID string) error {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lc_runtime WHERE room_id = ?`, roomID).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := leetcode.NowISO()
	timersJSON, _ := json.Marshal(leetcode.DefaultTimers())
	goalsJSON, _ := json.Marshal(leetcode.Goals{DailyTarget: 5, WeeklyTarget: 30, Streak: 0})
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO lc_runtime (room_id, code, whiteboard, goals, copy, loading_progress, timers, revision, updated_at)
		VALUES (?, '{}', '{}', ?, '{}', 0, ?, 1, ?)
	`, roomID, string(goalsJSON), string(timersJSON), now)
	return err
}

func (s *Store) seedLeetCode(ctx context.Context, roomID string) error {
	var flat leetcode.FlatDocument
	if err := json.Unmarshal(defaults.LeetCodeState(), &flat); err != nil {
		return err
	}
	return s.importFlatLeetCode(ctx, roomID, flat)
}

func (s *Store) importLeetCodeDoc(ctx context.Context, roomID string, data json.RawMessage) error {
	var flat leetcode.FlatDocument
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}
	return s.importFlatLeetCode(ctx, roomID, flat)
}

func (s *Store) importFlatLeetCode(ctx context.Context, roomID string, flat leetcode.FlatDocument) error {
	now := leetcode.NowISO()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range flat.Plan {
		if item.ID == "" {
			item.ID = leetcode.NewID("plan")
		} else {
			item.ID = defaults.ScopedSeedID(roomID, item.ID)
		}
		if err := insertPlanItemTx(ctx, tx, roomID, item); err != nil {
			return err
		}
	}
	for _, p := range flat.Problems {
		if err := insertProblemTx(ctx, tx, roomID, p); err != nil {
			return err
		}
	}

	codeJSON, _ := json.Marshal(flat.Code)
	wbJSON, _ := json.Marshal(flat.Whiteboard)
	goalsJSON, _ := json.Marshal(flat.Goals)
	copyJSON, _ := json.Marshal(flat.Copy)
	timersData := flat.Timers
	if len(timersData) == 0 {
		timersData, _ = json.Marshal(leetcode.DefaultTimers())
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO lc_runtime (room_id, code, whiteboard, goals, copy, loading_progress, timers, revision, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(room_id) DO UPDATE SET
			code = excluded.code,
			whiteboard = excluded.whiteboard,
			goals = excluded.goals,
			copy = excluded.copy,
			loading_progress = excluded.loading_progress,
			timers = excluded.timers,
			updated_at = excluded.updated_at
	`, roomID, string(codeJSON), string(wbJSON), string(goalsJSON), string(copyJSON), flat.LoadingProgress, string(timersData), now)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetLeetCodeState(ctx context.Context, roomID string) (leetcode.State, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.State{}, err
	}

	rt, err := s.getLCRuntime(ctx, roomID)
	if err != nil {
		return leetcode.State{}, err
	}

	plan, err := s.listPlanItems(ctx, roomID)
	if err != nil {
		return leetcode.State{}, err
	}
	problems, err := s.listProblems(ctx, roomID)
	if err != nil {
		return leetcode.State{}, err
	}

	var code leetcode.CodeEditor
	var wb leetcode.Whiteboard
	var goals leetcode.Goals
	var copy leetcode.Copy
	_ = json.Unmarshal(rt.Code, &code)
	_ = json.Unmarshal(rt.Whiteboard, &wb)
	_ = json.Unmarshal(rt.Goals, &goals)
	_ = json.Unmarshal(rt.Copy, &copy)

	return leetcode.State{
		Revision:            rt.Revision,
		ActiveLiveSessionID: rt.ActiveLiveSessionID,
		Plan:                plan,
		Problems:            problems,
		Code:                code,
		Whiteboard:          wb,
		Goals:               goals,
		Copy:                copy,
		LoadingProgress:     rt.LoadingProgress,
		Timers:              rt.Timers,
	}, nil
}

func (s *Store) PutLeetCodeState(ctx context.Context, roomID string, body json.RawMessage, expected *int64) (leetcode.State, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.State{}, err
	}

	rt, err := s.getLCRuntime(ctx, roomID)
	if err != nil {
		return leetcode.State{}, err
	}
	if expected != nil && rt.Revision != *expected {
		return leetcode.State{}, ErrRevisionConflict
	}

	var input leetcode.State
	if err := json.Unmarshal(body, &input); err != nil {
		return leetcode.State{}, fmt.Errorf("invalid state: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return leetcode.State{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, _ = tx.ExecContext(ctx, `DELETE FROM lc_plan_items WHERE room_id = ?`, roomID)
	for _, item := range input.Plan {
		if item.ID == "" {
			item.ID = leetcode.NewID("plan")
		}
		if err := insertPlanItemTx(ctx, tx, roomID, item); err != nil {
			return leetcode.State{}, err
		}
	}

	_, _ = tx.ExecContext(ctx, `DELETE FROM lc_problems WHERE room_id = ?`, roomID)
	for _, p := range input.Problems {
		if err := insertProblemTx(ctx, tx, roomID, p); err != nil {
			return leetcode.State{}, err
		}
	}

	codeJSON, _ := json.Marshal(input.Code)
	wbJSON, _ := json.Marshal(input.Whiteboard)
	goalsJSON, _ := json.Marshal(input.Goals)
	copyJSON, _ := json.Marshal(input.Copy)
	timersJSON := input.Timers
	if len(timersJSON) == 0 {
		timersJSON = rt.Timers
	}

	now := leetcode.NowISO()
	_, err = tx.ExecContext(ctx, `
		UPDATE lc_runtime SET
			active_live_session_id = ?,
			code = ?, whiteboard = ?, goals = ?, copy = ?,
			loading_progress = ?, timers = ?,
			revision = revision + 1, updated_at = ?
		WHERE room_id = ?
	`, input.ActiveLiveSessionID, string(codeJSON), string(wbJSON), string(goalsJSON), string(copyJSON),
		input.LoadingProgress, string(timersJSON), now, roomID)
	if err != nil {
		return leetcode.State{}, err
	}

	if err := tx.Commit(); err != nil {
		return leetcode.State{}, err
	}
	state, err := s.GetLeetCodeState(ctx, roomID)
	if err != nil {
		return leetcode.State{}, err
	}
	s.publishState(roomID, "leetcode", state.Revision)
	return state, nil
}

func (s *Store) getLCRuntime(ctx context.Context, roomID string) (lcRuntime, error) {
	var rt lcRuntime
	var code, wb, goals, copy, timersStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT active_live_session_id, code, whiteboard, goals, copy, loading_progress, timers, revision
		FROM lc_runtime WHERE room_id = ?
	`, roomID).Scan(&rt.ActiveLiveSessionID, &code, &wb, &goals, &copy, &rt.LoadingProgress, &timersStr, &rt.Revision)
	if err != nil {
		return rt, err
	}
	rt.Code = json.RawMessage(code)
	rt.Whiteboard = json.RawMessage(wb)
	rt.Goals = json.RawMessage(goals)
	rt.Copy = json.RawMessage(copy)
	rt.Timers = json.RawMessage(timersStr)
	return rt, nil
}

func (s *Store) bumpLCRevision(ctx context.Context, roomID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lc_runtime SET revision = revision + 1, updated_at = ? WHERE room_id = ?
	`, leetcode.NowISO(), roomID)
	return err
}

/* ─── Live Sessions ─── */

func (s *Store) ListLiveSessions(ctx context.Context, roomID string) ([]leetcode.LiveSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, room_id, domain, title, platforms, started_at, ended_at
		FROM live_sessions WHERE room_id = ? ORDER BY started_at DESC
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []leetcode.LiveSession
	for rows.Next() {
		sess, err := scanLiveSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *Store) CreateLiveSession(ctx context.Context, roomID string, in leetcode.CreateLiveSessionInput) (leetcode.LiveSession, error) {
	id := leetcode.NewID("live")
	now := leetcode.NowISO()
	domain := in.Domain
	if domain == "" {
		domain = "leetcode"
	}
	platforms, _ := json.Marshal(in.Platforms)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO live_sessions (id, room_id, domain, title, platforms, started_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, roomID, domain, in.Title, string(platforms), now)
	if err != nil {
		return leetcode.LiveSession{}, err
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE lc_runtime SET active_live_session_id = ? WHERE room_id = ?
	`, id, roomID)
	_ = s.bumpLCRevision(ctx, roomID)
	return s.GetLiveSession(ctx, roomID, id)
}

func (s *Store) GetLiveSession(ctx context.Context, roomID, sessionID string) (leetcode.LiveSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, room_id, domain, title, platforms, started_at, ended_at
		FROM live_sessions WHERE id = ? AND room_id = ?
	`, sessionID, roomID)
	sess, err := scanLiveSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return leetcode.LiveSession{}, ErrNotFound
	}
	return sess, err
}

func (s *Store) UpdateLiveSession(ctx context.Context, roomID, sessionID string, in leetcode.UpdateLiveSessionInput) (leetcode.LiveSession, error) {
	sess, err := s.GetLiveSession(ctx, roomID, sessionID)
	if err != nil {
		return leetcode.LiveSession{}, err
	}
	if in.Title != nil {
		sess.Title = in.Title
	}
	if in.EndedAt != nil {
		sess.EndedAt = in.EndedAt
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE live_sessions SET title = ?, ended_at = ? WHERE id = ?
	`, sess.Title, sess.EndedAt, sessionID)
	if err != nil {
		return leetcode.LiveSession{}, err
	}
	return s.GetLiveSession(ctx, roomID, sessionID)
}

/* ─── Plan ─── */

func (s *Store) ListPlanItems(ctx context.Context, roomID string) ([]leetcode.PlanItem, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return nil, err
	}
	return s.listPlanItems(ctx, roomID)
}

func (s *Store) CreatePlanItem(ctx context.Context, roomID string, in leetcode.CreatePlanItemInput) (leetcode.PlanItem, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.PlanItem{}, err
	}
	items, _ := s.listPlanItems(ctx, roomID)
	order := len(items)
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	item := leetcode.PlanItem{ID: leetcode.NewID("plan"), Label: in.Label, SortOrder: order}
	if err := insertPlanItemTx(ctx, s.db, roomID, item); err != nil {
		return leetcode.PlanItem{}, err
	}
	_ = s.bumpLCRevision(ctx, roomID)
	return item, nil
}

func (s *Store) UpdatePlanItem(ctx context.Context, roomID, itemID string, in leetcode.UpdatePlanItemInput) (leetcode.PlanItem, error) {
	item, err := s.getPlanItem(ctx, roomID, itemID)
	if err != nil {
		return leetcode.PlanItem{}, err
	}
	if in.Label != nil {
		item.Label = *in.Label
	}
	if in.Done != nil {
		item.Done = *in.Done
	}
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	done := 0
	if item.Done {
		done = 1
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE lc_plan_items SET label = ?, done = ?, sort_order = ? WHERE id = ? AND room_id = ?
	`, item.Label, done, item.SortOrder, itemID, roomID)
	if err != nil {
		return leetcode.PlanItem{}, err
	}
	_ = s.bumpLCRevision(ctx, roomID)
	return item, nil
}

func (s *Store) DeletePlanItem(ctx context.Context, roomID, itemID string) error {
	if _, err := s.getPlanItem(ctx, roomID, itemID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM lc_plan_items WHERE id = ? AND room_id = ?`, itemID, roomID)
	if err != nil {
		return err
	}
	return s.bumpLCRevision(ctx, roomID)
}

func (s *Store) TogglePlanItem(ctx context.Context, roomID, itemID string) (leetcode.PlanItem, error) {
	item, err := s.getPlanItem(ctx, roomID, itemID)
	if err != nil {
		return leetcode.PlanItem{}, err
	}
	done := !item.Done
	return s.UpdatePlanItem(ctx, roomID, itemID, leetcode.UpdatePlanItemInput{Done: &done})
}

/* ─── Problems ─── */

func (s *Store) ListProblems(ctx context.Context, roomID string) ([]leetcode.Problem, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return nil, err
	}
	return s.listProblems(ctx, roomID)
}

func (s *Store) CreateProblem(ctx context.Context, roomID string, in leetcode.CreateProblemInput) (leetcode.Problem, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.Problem{}, err
	}
	problems, _ := s.listProblems(ctx, roomID)
	order := len(problems)
	if in.SortOrder != nil {
		order = *in.SortOrder
	}
	p := leetcode.Problem{
		ID: in.ID, Title: in.Title, Difficulty: in.Difficulty,
		Description: in.Description, Status: leetcode.StatusQueued, SortOrder: order,
	}
	if err := insertProblemTx(ctx, s.db, roomID, p); err != nil {
		return leetcode.Problem{}, err
	}
	_ = s.bumpLCRevision(ctx, roomID)
	return p, nil
}

func (s *Store) GetProblem(ctx context.Context, roomID string, problemID int) (leetcode.Problem, error) {
	return s.getProblem(ctx, roomID, problemID)
}

func (s *Store) UpdateProblem(ctx context.Context, roomID string, problemID int, in leetcode.UpdateProblemInput) (leetcode.Problem, error) {
	p, err := s.getProblem(ctx, roomID, problemID)
	if err != nil {
		return leetcode.Problem{}, err
	}
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Difficulty != nil {
		p.Difficulty = *in.Difficulty
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.Status != nil {
		p.Status = *in.Status
	}
	if in.SortOrder != nil {
		p.SortOrder = *in.SortOrder
	}
	if err := s.updateProblemRow(ctx, roomID, p); err != nil {
		return leetcode.Problem{}, err
	}
	_ = s.bumpLCRevision(ctx, roomID)
	return p, nil
}

func (s *Store) DeleteProblem(ctx context.Context, roomID string, problemID int) error {
	if _, err := s.getProblem(ctx, roomID, problemID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM lc_problems WHERE room_id = ? AND problem_id = ?`, roomID, problemID)
	if err != nil {
		return err
	}
	return s.bumpLCRevision(ctx, roomID)
}

func (s *Store) ActivateProblem(ctx context.Context, roomID string, problemID int) (leetcode.Problem, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.Problem{}, err
	}

	rt, err := s.getLCRuntime(ctx, roomID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return leetcode.Problem{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		UPDATE lc_problems SET status = 'queued'
		WHERE room_id = ? AND status = 'active'
	`, roomID)
	if err != nil {
		return leetcode.Problem{}, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE lc_problems SET status = 'active'
		WHERE room_id = ? AND problem_id = ?
	`, roomID, problemID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	now := leetcode.NowISO()
	attemptID := leetcode.NewID("attempt")
	_, err = tx.ExecContext(ctx, `
		INSERT INTO lc_problem_attempts (id, room_id, problem_id, live_session_id, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, attemptID, roomID, problemID, rt.ActiveLiveSessionID, now)
	if err != nil {
		return leetcode.Problem{}, err
	}

	p, err := s.getProblemTx(ctx, tx, roomID, problemID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	var wb leetcode.Whiteboard
	_ = json.Unmarshal(rt.Whiteboard, &wb)
	wb.Title = p.Title
	wbJSON, _ := json.Marshal(wb)
	_, _ = tx.ExecContext(ctx, `UPDATE lc_runtime SET whiteboard = ?, revision = revision + 1, updated_at = ? WHERE room_id = ?`,
		string(wbJSON), now, roomID)

	if err := tx.Commit(); err != nil {
		return leetcode.Problem{}, err
	}
	return p, nil
}

func (s *Store) SolveProblem(ctx context.Context, roomID string, problemID int) (leetcode.Problem, error) {
	now := leetcode.NowISO()
	if _, err := s.getProblem(ctx, roomID, problemID); err != nil {
		return leetcode.Problem{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return leetcode.Problem{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		UPDATE lc_problems SET status = 'solved', solved_at = ?
		WHERE room_id = ? AND problem_id = ?
	`, now, roomID, problemID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	var attemptID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM lc_problem_attempts
		WHERE room_id = ? AND problem_id = ? AND solved_at IS NULL
		ORDER BY started_at DESC LIMIT 1
	`, roomID, problemID).Scan(&attemptID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE lc_problem_attempts SET solved_at = ? WHERE id = ?`, now, attemptID)
		if err != nil {
			return leetcode.Problem{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return leetcode.Problem{}, err
	}

	if err := tx.Commit(); err != nil {
		return leetcode.Problem{}, err
	}

	streak, err := s.computeStreak(ctx, roomID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	rt, err := s.getLCRuntime(ctx, roomID)
	if err != nil {
		return leetcode.Problem{}, err
	}
	var goals leetcode.Goals
	_ = json.Unmarshal(rt.Goals, &goals)
	goals.Streak = streak
	goalsJSON, _ := json.Marshal(goals)
	_, err = s.db.ExecContext(ctx, `
		UPDATE lc_runtime SET goals = ?, revision = revision + 1, updated_at = ? WHERE room_id = ?
	`, string(goalsJSON), now, roomID)
	if err != nil {
		return leetcode.Problem{}, err
	}

	return s.getProblem(ctx, roomID, problemID)
}

func (s *Store) SkipProblem(ctx context.Context, roomID string, problemID int) (leetcode.Problem, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lc_problems SET status = 'skipped' WHERE room_id = ? AND problem_id = ?
	`, roomID, problemID)
	if err != nil {
		return leetcode.Problem{}, err
	}
	_ = s.bumpLCRevision(ctx, roomID)
	return s.getProblem(ctx, roomID, problemID)
}

/* ─── Stats ─── */

func (s *Store) GetLeetCodeStats(ctx context.Context, roomID, month, liveSessionID string) (leetcode.Stats, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return leetcode.Stats{}, err
	}

	stats := leetcode.Stats{Month: month, LiveSessionID: liveSessionID}

	if liveSessionID != "" {
		rows, err := s.db.QueryContext(ctx, `
			SELECT DISTINCT problem_id FROM lc_problem_attempts
			WHERE room_id = ? AND live_session_id = ? AND solved_at IS NOT NULL
		`, roomID, liveSessionID)
		if err != nil {
			return stats, err
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return stats, err
			}
			stats.ProblemIDs = append(stats.ProblemIDs, id)
			stats.SolvedCount++
		}
		return stats, rows.Err()
	}

	if month != "" {
		prefix := month
		if len(month) == 7 {
			prefix = month + "-"
		}
		rows, err := s.db.QueryContext(ctx, `
			SELECT problem_id FROM lc_problems
			WHERE room_id = ? AND status = 'solved' AND solved_at LIKE ?
		`, roomID, prefix+"%")
		if err != nil {
			return stats, err
		}
		defer rows.Close()
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return stats, err
			}
			stats.ProblemIDs = append(stats.ProblemIDs, id)
			stats.SolvedCount++
		}
		return stats, rows.Err()
	}

	return stats, nil
}

func (s *Store) GetLeetCodeStreak(ctx context.Context, roomID string) (leetcode.Stats, error) {
	streak, err := s.computeStreak(ctx, roomID)
	if err != nil {
		return leetcode.Stats{}, err
	}
	return leetcode.Stats{Streak: streak}, nil
}

func (s *Store) ListProblemAttempts(ctx context.Context, roomID, liveSessionID string) ([]leetcode.ProblemAttempt, error) {
	q := `SELECT id, room_id, problem_id, live_session_id, started_at, solved_at, notes FROM lc_problem_attempts WHERE room_id = ?`
	args := []any{roomID}
	if liveSessionID != "" {
		q += ` AND live_session_id = ?`
		args = append(args, liveSessionID)
	}
	q += ` ORDER BY started_at DESC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []leetcode.ProblemAttempt
	for rows.Next() {
		var a leetcode.ProblemAttempt
		if err := rows.Scan(&a.ID, &a.RoomID, &a.ProblemID, &a.LiveSessionID, &a.StartedAt, &a.SolvedAt, &a.Notes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) UpdateLCTimer(ctx context.Context, roomID, timerID, action string, body json.RawMessage) (json.RawMessage, error) {
	rt, err := s.getLCRuntime(ctx, roomID)
	if err != nil {
		return nil, err
	}
	var timersMap map[string]any
	if err := json.Unmarshal(rt.Timers, &timersMap); err != nil {
		return nil, err
	}

	if action != "" {
		t, ok := timers.GetTimer(timersMap, timerID)
		if !ok {
			return nil, ErrNotFound
		}
		if err := timers.ApplyAction(t, action, time.Now().UnixMilli()); err != nil {
			return nil, err
		}
	} else if len(body) > 0 {
		var patch map[string]any
		if err := json.Unmarshal(body, &patch); err != nil {
			return nil, err
		}
		timersMap[timerID] = patch
	}

	updated, _ := json.Marshal(timersMap)
	_, err = s.db.ExecContext(ctx, `
		UPDATE lc_runtime SET timers = ?, revision = revision + 1, updated_at = ? WHERE room_id = ?
	`, string(updated), leetcode.NowISO(), roomID)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(updated), nil
}

func (s *Store) GetLCTimers(ctx context.Context, roomID string) (json.RawMessage, error) {
	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return nil, err
	}
	rt, err := s.getLCRuntime(ctx, roomID)
	return rt.Timers, err
}

/* ─── helpers ─── */

func insertPlanItemTx(ctx context.Context, q sqlExecutor, roomID string, item leetcode.PlanItem) error {
	done := 0
	if item.Done {
		done = 1
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO lc_plan_items (id, room_id, label, done, sort_order)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING
	`, item.ID, roomID, item.Label, done, item.SortOrder)
	return err
}

func insertProblemTx(ctx context.Context, q sqlExecutor, roomID string, p leetcode.Problem) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO lc_problems (room_id, problem_id, title, difficulty, description, status, solved_at, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, roomID, p.ID, p.Title, p.Difficulty, p.Description, p.Status, p.SolvedAt, p.SortOrder)
	return err
}

func (s *Store) updateProblemRow(ctx context.Context, roomID string, p leetcode.Problem) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE lc_problems SET title = ?, difficulty = ?, description = ?, status = ?, solved_at = ?, sort_order = ?
		WHERE room_id = ? AND problem_id = ?
	`, p.Title, p.Difficulty, p.Description, p.Status, p.SolvedAt, p.SortOrder, roomID, p.ID)
	return err
}

func (s *Store) listPlanItems(ctx context.Context, roomID string) ([]leetcode.PlanItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, label, done, sort_order FROM lc_plan_items
		WHERE room_id = ? ORDER BY sort_order
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []leetcode.PlanItem
	for rows.Next() {
		var item leetcode.PlanItem
		var done int
		if err := rows.Scan(&item.ID, &item.Label, &done, &item.SortOrder); err != nil {
			return nil, err
		}
		item.Done = done == 1
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) listProblems(ctx context.Context, roomID string) ([]leetcode.Problem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT problem_id, title, difficulty, description, status, solved_at, sort_order
		FROM lc_problems WHERE room_id = ? ORDER BY sort_order
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []leetcode.Problem
	for rows.Next() {
		p, err := scanProblem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) getPlanItem(ctx context.Context, roomID, itemID string) (leetcode.PlanItem, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, label, done, sort_order FROM lc_plan_items WHERE id = ? AND room_id = ?
	`, itemID, roomID)
	var item leetcode.PlanItem
	var done int
	err := row.Scan(&item.ID, &item.Label, &done, &item.SortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	item.Done = done == 1
	return item, err
}

func (s *Store) getProblem(ctx context.Context, roomID string, problemID int) (leetcode.Problem, error) {
	return s.getProblemTx(ctx, s.db, roomID, problemID)
}

func (s *Store) getProblemTx(ctx context.Context, q sqlExecutor, roomID string, problemID int) (leetcode.Problem, error) {
	row := q.QueryRowContext(ctx, `
		SELECT problem_id, title, difficulty, description, status, solved_at, sort_order
		FROM lc_problems WHERE room_id = ? AND problem_id = ?
	`, roomID, problemID)
	p, err := scanProblem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func scanProblem(row scannable) (leetcode.Problem, error) {
	var p leetcode.Problem
	err := row.Scan(&p.ID, &p.Title, &p.Difficulty, &p.Description, &p.Status, &p.SolvedAt, &p.SortOrder)
	return p, err
}

func scanLiveSession(row scannable) (leetcode.LiveSession, error) {
	var sess leetcode.LiveSession
	var platformsStr string
	err := row.Scan(&sess.ID, &sess.RoomID, &sess.Domain, &sess.Title, &platformsStr, &sess.StartedAt, &sess.EndedAt)
	if err != nil {
		return sess, err
	}
	_ = json.Unmarshal([]byte(platformsStr), &sess.Platforms)
	if sess.Platforms == nil {
		sess.Platforms = []string{}
	}
	return sess, nil
}

func (s *Store) computeStreak(ctx context.Context, roomID string) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT substr(solved_at, 1, 10) AS day
		FROM lc_problems
		WHERE room_id = ? AND status = 'solved' AND solved_at IS NOT NULL
	`, roomID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var days []string
	for rows.Next() {
		var day string
		if err := rows.Scan(&day); err != nil {
			return 0, err
		}
		days = append(days, day)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(days) == 0 {
		return 0, nil
	}

	sort.Sort(sort.Reverse(sort.StringSlice(days)))
	today := time.Now().UTC().Format("2006-01-02")
	yesterday := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02")

	streak := 0
	expected := today
	if days[0] != today && days[0] != yesterday {
		return 0, nil
	}
	if days[0] == yesterday {
		expected = yesterday
	}

	for _, day := range days {
		if day != expected {
			break
		}
		streak++
		t, _ := time.Parse("2006-01-02", expected)
		expected = t.Add(-24 * time.Hour).Format("2006-01-02")
	}
	return streak, nil
}
