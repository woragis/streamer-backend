package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/woragis/streamer-backend/internal/calisthenics"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/timers"
)

var (
	ErrCalNoActiveWorkout = errors.New("no active workout")
	ErrCalInvalidSet      = errors.New("invalid set state")
)

type calRuntime struct {
	ActiveWorkoutID   *string
	TodayGoalLabel    string
	TodayGoalProgress int
	Timers            json.RawMessage
	Revision          int64
}

func (s *Store) EnsureCalisthenics(ctx context.Context, roomID string) error {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cal_workouts WHERE room_id = ?`, roomID).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return s.ensureCalRuntime(ctx, roomID)
	}

	doc, err := s.GetDocument(ctx, roomID, DocCalisthenics)
	if err == nil {
		return s.importCalisthenicsDoc(ctx, roomID, doc.Data)
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}

	return s.seedCalisthenics(ctx, roomID)
}

func (s *Store) ensureCalRuntime(ctx context.Context, roomID string) error {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cal_runtime WHERE room_id = ?`, roomID).Scan(&n)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	timersJSON, _ := json.Marshal(timers.DefaultCalisthenicsTimers())
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cal_runtime (room_id, today_goal_label, today_goal_progress, timers, revision, updated_at)
		VALUES (?, 'COMPLETE THE WORKOUT', 0, ?, 1, ?)
	`, roomID, string(timersJSON), now)
	return err
}

func (s *Store) seedCalisthenics(ctx context.Context, roomID string) error {
	var flat calisthenics.FlatDocument
	if err := json.Unmarshal(defaults.CalisthenicsState(), &flat); err != nil {
		return err
	}
	return s.importFlatCalisthenics(ctx, roomID, flat)
}

func (s *Store) importCalisthenicsDoc(ctx context.Context, roomID string, data json.RawMessage) error {
	var flat calisthenics.FlatDocument
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}
	return s.importFlatCalisthenics(ctx, roomID, flat)
}

func (s *Store) importFlatCalisthenics(ctx context.Context, roomID string, flat calisthenics.FlatDocument) error {
	now := calisthenics.NowISO()
	workoutID := calisthenics.NewID("wo")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cal_workouts (id, room_id, workout_type, status, started_at, created_at)
		VALUES (?, ?, ?, 'active', ?, ?)
	`, workoutID, roomID, flat.WorkoutType, now, now); err != nil {
		return err
	}

	for _, fex := range flat.Exercises {
		ex := calisthenics.ExerciseFromFlat(fex, workoutID)
		if err := insertExerciseTx(ctx, tx, ex); err != nil {
			return err
		}
		for _, set := range calisthenics.SetsFromFlat(fex) {
			if err := insertSetTx(ctx, tx, set); err != nil {
				return err
			}
		}
	}

	timersData := flat.Timers
	if len(timersData) == 0 {
		timersData, _ = json.Marshal(timers.DefaultCalisthenicsTimers())
	} else {
		var tm map[string]any
		_ = json.Unmarshal(timersData, &tm)
		if _, ok := tm["hold"]; !ok {
			defaults := timers.DefaultCalisthenicsTimers()
			tm["hold"] = defaults["hold"]
			timersData, _ = json.Marshal(tm)
		}
	}

	progress := flat.TodayGoal.Progress
	if progress == 0 {
		exercises, _ := listExercisesTx(ctx, tx, workoutID)
		progress = calisthenics.ComputeGoalProgress(exercises)
	}

	label := flat.TodayGoal.Label
	if label == "" {
		label = "COMPLETE THE WORKOUT"
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO cal_runtime (room_id, active_workout_id, today_goal_label, today_goal_progress, timers, revision, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT(room_id) DO UPDATE SET
			active_workout_id = excluded.active_workout_id,
			today_goal_label = excluded.today_goal_label,
			today_goal_progress = excluded.today_goal_progress,
			timers = excluded.timers,
			updated_at = excluded.updated_at
	`, roomID, workoutID, label, progress, string(timersData), now); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetCalisthenicsState(ctx context.Context, roomID string) (calisthenics.State, error) {
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return calisthenics.State{}, err
	}

	rt, err := s.getCalRuntime(ctx, roomID)
	if err != nil {
		return calisthenics.State{}, err
	}
	if rt.ActiveWorkoutID == nil {
		return calisthenics.State{}, ErrCalNoActiveWorkout
	}

	workout, err := s.getWorkout(ctx, *rt.ActiveWorkoutID)
	if err != nil {
		return calisthenics.State{}, err
	}

	exercises, err := s.listExercises(ctx, workout.ID)
	if err != nil {
		return calisthenics.State{}, err
	}

	views := make([]calisthenics.ExerciseView, 0, len(exercises))
	for _, ex := range exercises {
		sets, err := s.listSets(ctx, ex.ID)
		if err != nil {
			return calisthenics.State{}, err
		}
	views = append(views, calisthenics.BuildExerciseView(ex, sets))
	}

	alert, err := s.GetPendingSkillAlert(ctx, roomID)
	if err != nil {
		return calisthenics.State{}, err
	}

	return calisthenics.State{
		Revision:      rt.Revision,
		WorkoutID:     workout.ID,
		WorkoutType:   workout.WorkoutType,
		WorkoutStatus: string(workout.Status),
		Exercises:     views,
		TodayGoal: calisthenics.TodayGoal{
			Label:    rt.TodayGoalLabel,
			Progress: rt.TodayGoalProgress,
		},
		Timers:     rt.Timers,
		SkillAlert: alert,
	}, nil
}

func (s *Store) PutCalisthenicsState(ctx context.Context, roomID string, body json.RawMessage, expected *int64) (calisthenics.State, error) {
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return calisthenics.State{}, err
	}

	rt, err := s.getCalRuntime(ctx, roomID)
	if err != nil {
		return calisthenics.State{}, err
	}
	if expected != nil && rt.Revision != *expected {
		return calisthenics.State{}, ErrRevisionConflict
	}

	var input calisthenics.State
	if err := json.Unmarshal(body, &input); err != nil {
		return calisthenics.State{}, fmt.Errorf("invalid state: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return calisthenics.State{}, err
	}
	defer func() { _ = tx.Rollback() }()

	now := calisthenics.NowISO()
	workoutID := input.WorkoutID
	if workoutID == "" && rt.ActiveWorkoutID != nil {
		workoutID = *rt.ActiveWorkoutID
	}

	if workoutID != "" {
		status := input.WorkoutStatus
		if status == "" {
			status = "active"
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE cal_workouts SET workout_type = ?, status = ? WHERE id = ? AND room_id = ?
		`, input.WorkoutType, status, workoutID, roomID)
		if err != nil {
			return calisthenics.State{}, err
		}

		_, _ = tx.ExecContext(ctx, `DELETE FROM cal_sets WHERE exercise_id IN (SELECT id FROM cal_workout_exercises WHERE workout_id = ?)`, workoutID)
		_, _ = tx.ExecContext(ctx, `DELETE FROM cal_workout_exercises WHERE workout_id = ?`, workoutID)

		for _, view := range input.Exercises {
			ex := calisthenics.Exercise{
				ID:          view.ID,
				WorkoutID:   workoutID,
				Name:        view.Name,
				MovementID:  view.MovementID,
				PlannedSets: view.Sets,
				RepTarget:   view.RepTarget,
				Status:      calisthenics.ExerciseStatus(view.Status),
				SortOrder:   view.Order,
			}
			if ex.ID == "" {
				ex.ID = calisthenics.NewID("ex")
			}
			if err := insertExerciseTx(ctx, tx, ex); err != nil {
				return calisthenics.State{}, err
			}
			if len(view.SetDetails) > 0 {
				for _, set := range view.SetDetails {
					set.ExerciseID = ex.ID
					if set.ID == "" {
						set.ID = calisthenics.NewID("set")
					}
					if err := insertSetTx(ctx, tx, set); err != nil {
						return calisthenics.State{}, err
					}
				}
			} else {
				flat := calisthenics.FlatExercise{
					ID: view.ID, Name: view.Name, Sets: view.Sets, RepTarget: view.RepTarget,
					CompletedSets: view.CompletedSets, RepsInCurrentSet: view.RepsInCurrentSet,
					TotalReps: view.TotalReps, Status: view.Status, Order: view.Order,
				}
				for _, set := range calisthenics.SetsFromFlat(flat) {
					set.ExerciseID = ex.ID
					if err := insertSetTx(ctx, tx, set); err != nil {
						return calisthenics.State{}, err
					}
				}
			}
		}
	}

	timersJSON := input.Timers
	if len(timersJSON) == 0 {
		timersJSON = rt.Timers
	}

	progress := input.TodayGoal.Progress
	label := input.TodayGoal.Label
	if label == "" {
		label = rt.TodayGoalLabel
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE cal_runtime
		SET active_workout_id = ?, today_goal_label = ?, today_goal_progress = ?,
		    timers = ?, revision = revision + 1, updated_at = ?
		WHERE room_id = ?
	`, workoutID, label, progress, string(timersJSON), now, roomID)
	if err != nil {
		return calisthenics.State{}, err
	}

	if err := tx.Commit(); err != nil {
		return calisthenics.State{}, err
	}
	state, err := s.GetCalisthenicsState(ctx, roomID)
	if err != nil {
		return calisthenics.State{}, err
	}
	s.publishState(roomID, "calisthenics", state.Revision)
	return state, nil
}

func (s *Store) getCalRuntime(ctx context.Context, roomID string) (calRuntime, error) {
	var rt calRuntime
	var timersStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT active_workout_id, today_goal_label, today_goal_progress, timers, revision
		FROM cal_runtime WHERE room_id = ?
	`, roomID).Scan(&rt.ActiveWorkoutID, &rt.TodayGoalLabel, &rt.TodayGoalProgress, &timersStr, &rt.Revision)
	if err != nil {
		return rt, err
	}
	rt.Timers = json.RawMessage(timersStr)
	return rt, nil
}

func (s *Store) bumpCalRevision(ctx context.Context, roomID string) error {
	now := calisthenics.NowISO()
	_, err := s.db.ExecContext(ctx, `
		UPDATE cal_runtime SET revision = revision + 1, updated_at = ? WHERE room_id = ?
	`, now, roomID)
	return err
}

func (s *Store) syncGoalProgress(ctx context.Context, workoutID, roomID string) error {
	exercises, err := s.listExercises(ctx, workoutID)
	if err != nil {
		return err
	}
	progress := calisthenics.ComputeGoalProgress(exercises)
	_, err = s.db.ExecContext(ctx, `
		UPDATE cal_runtime SET today_goal_progress = ? WHERE room_id = ?
	`, progress, roomID)
	return err
}

/* ─── Workouts ─── */

func (s *Store) ListWorkouts(ctx context.Context, roomID string) ([]calisthenics.Workout, error) {
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, room_id, workout_type, status, live_session_id, started_at, ended_at, created_at
		FROM cal_workouts WHERE room_id = ? ORDER BY created_at DESC
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkouts(rows)
}

func (s *Store) CreateWorkout(ctx context.Context, roomID string, in calisthenics.CreateWorkoutInput) (calisthenics.Workout, error) {
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return calisthenics.Workout{}, err
	}
	id := calisthenics.NewID("wo")
	now := calisthenics.NowISO()
	status := in.Status
	if status == "" {
		status = string(calisthenics.WorkoutActive)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cal_workouts (id, room_id, workout_type, status, live_session_id, started_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, in.WorkoutType, status, in.LiveSessionID, now, now)
	if err != nil {
		return calisthenics.Workout{}, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE cal_runtime SET active_workout_id = ? WHERE room_id = ?`, id, roomID)
	_ = s.bumpCalRevision(ctx, roomID)
	return s.getWorkout(ctx, id)
}

func (s *Store) GetWorkoutByID(ctx context.Context, roomID, workoutID string) (calisthenics.Workout, error) {
	return s.getWorkoutScoped(ctx, roomID, workoutID)
}

func (s *Store) UpdateWorkout(ctx context.Context, roomID, workoutID string, in calisthenics.UpdateWorkoutInput) (calisthenics.Workout, error) {
	w, err := s.getWorkoutScoped(ctx, roomID, workoutID)
	if err != nil {
		return calisthenics.Workout{}, err
	}
	if in.WorkoutType != nil {
		w.WorkoutType = *in.WorkoutType
	}
	if in.Status != nil {
		w.Status = calisthenics.WorkoutStatus(*in.Status)
	}
	if in.EndedAt != nil {
		w.EndedAt = in.EndedAt
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE cal_workouts SET workout_type = ?, status = ?, ended_at = ? WHERE id = ?
	`, w.WorkoutType, w.Status, w.EndedAt, workoutID)
	if err != nil {
		return calisthenics.Workout{}, err
	}
	_ = s.bumpCalRevision(ctx, roomID)
	return s.getWorkout(ctx, workoutID)
}

func (s *Store) DeleteWorkout(ctx context.Context, roomID, workoutID string) error {
	if _, err := s.getWorkoutScoped(ctx, roomID, workoutID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM cal_workouts WHERE id = ?`, workoutID)
	if err != nil {
		return err
	}
	return s.bumpCalRevision(ctx, roomID)
}

func (s *Store) getWorkout(ctx context.Context, id string) (calisthenics.Workout, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, room_id, workout_type, status, live_session_id, started_at, ended_at, created_at
		FROM cal_workouts WHERE id = ?
	`, id)
	return scanWorkout(row)
}

func (s *Store) getWorkoutScoped(ctx context.Context, roomID, id string) (calisthenics.Workout, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, room_id, workout_type, status, live_session_id, started_at, ended_at, created_at
		FROM cal_workouts WHERE id = ? AND room_id = ?
	`, id, roomID)
	w, err := scanWorkout(row)
	if errors.Is(err, sql.ErrNoRows) {
		return calisthenics.Workout{}, ErrNotFound
	}
	return w, err
}

/* ─── Exercises ─── */

func (s *Store) ListExercises(ctx context.Context, roomID, workoutID string) ([]calisthenics.Exercise, error) {
	if _, err := s.getWorkoutScoped(ctx, roomID, workoutID); err != nil {
		return nil, err
	}
	return s.listExercises(ctx, workoutID)
}

func (s *Store) CreateExercise(ctx context.Context, roomID, workoutID string, in calisthenics.CreateExerciseInput) (calisthenics.ExerciseView, error) {
	if _, err := s.getWorkoutScoped(ctx, roomID, workoutID); err != nil {
		return calisthenics.ExerciseView{}, err
	}

	exercises, _ := s.listExercises(ctx, workoutID)
	order := len(exercises)
	if in.SortOrder != nil {
		order = *in.SortOrder
	}

	hasActive := false
	for _, e := range exercises {
		if e.Status == calisthenics.ExerciseActive {
			hasActive = true
			break
		}
	}

	ex := calisthenics.Exercise{
		ID:          calisthenics.NewID("ex"),
		WorkoutID:   workoutID,
		Name:        in.Name,
		MovementID:  in.MovementID,
		PlannedSets: in.PlannedSets,
		RepTarget:   in.RepTarget,
		Status:      calisthenics.ExercisePending,
		SortOrder:   order,
	}
	if !hasActive {
		ex.Status = calisthenics.ExerciseActive
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertExerciseTx(ctx, tx, ex); err != nil {
		return calisthenics.ExerciseView{}, err
	}
	for i := 1; i <= in.PlannedSets; i++ {
		set := calisthenics.Set{
			ID:         calisthenics.NewID("set"),
			ExerciseID: ex.ID,
			SetNumber:  i,
			RepsTarget: in.RepTarget,
		}
		if err := insertSetTx(ctx, tx, set); err != nil {
			return calisthenics.ExerciseView{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return calisthenics.ExerciseView{}, err
	}
	_ = s.bumpCalRevision(ctx, roomID)
	sets, _ := s.listSets(ctx, ex.ID)
	return calisthenics.BuildExerciseView(ex, sets), nil
}

func (s *Store) UpdateExercise(ctx context.Context, roomID, exerciseID string, in calisthenics.UpdateExerciseInput) (calisthenics.ExerciseView, error) {
	ex, workoutID, err := s.getExerciseScoped(ctx, roomID, exerciseID)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	if in.Name != nil {
		ex.Name = *in.Name
	}
	if in.PlannedSets != nil {
		ex.PlannedSets = *in.PlannedSets
	}
	if in.RepTarget != nil {
		ex.RepTarget = *in.RepTarget
	}
	if in.MovementID != nil {
		ex.MovementID = in.MovementID
	}
	if in.SortOrder != nil {
		ex.SortOrder = *in.SortOrder
	}
	if in.Status != nil {
		ex.Status = calisthenics.ExerciseStatus(*in.Status)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE cal_workout_exercises
		SET name = ?, planned_sets = ?, rep_target = ?, movement_id = ?, sort_order = ?, status = ?
		WHERE id = ?
	`, ex.Name, ex.PlannedSets, ex.RepTarget, ex.MovementID, ex.SortOrder, ex.Status, exerciseID)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	_ = s.bumpCalRevision(ctx, roomID)
	_ = s.syncGoalProgress(ctx, workoutID, roomID)
	sets, _ := s.listSets(ctx, exerciseID)
	return calisthenics.BuildExerciseView(ex, sets), nil
}

func (s *Store) DeleteExercise(ctx context.Context, roomID, exerciseID string) error {
	if _, _, err := s.getExerciseScoped(ctx, roomID, exerciseID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM cal_workout_exercises WHERE id = ?`, exerciseID)
	if err != nil {
		return err
	}
	return s.bumpCalRevision(ctx, roomID)
}

func (s *Store) ActivateExercise(ctx context.Context, roomID, exerciseID string) (calisthenics.ExerciseView, error) {
	ex, workoutID, err := s.getExerciseScoped(ctx, roomID, exerciseID)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		UPDATE cal_workout_exercises SET status = 'pending'
		WHERE workout_id = ? AND status = 'active'
	`, workoutID)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE cal_workout_exercises SET status = 'active' WHERE id = ?`, exerciseID)
	if err != nil {
		return calisthenics.ExerciseView{}, err
	}
	if err := tx.Commit(); err != nil {
		return calisthenics.ExerciseView{}, err
	}
	ex.Status = calisthenics.ExerciseActive
	_ = s.bumpCalRevision(ctx, roomID)
	sets, _ := s.listSets(ctx, exerciseID)
	return calisthenics.BuildExerciseView(ex, sets), nil
}

/* ─── Sets ─── */

func (s *Store) ListSets(ctx context.Context, roomID, exerciseID string) ([]calisthenics.Set, error) {
	if _, _, err := s.getExerciseScoped(ctx, roomID, exerciseID); err != nil {
		return nil, err
	}
	return s.listSets(ctx, exerciseID)
}

func (s *Store) IncrementRep(ctx context.Context, roomID, setID string) (calisthenics.Set, error) {
	set, ex, workoutID, err := s.getSetScoped(ctx, roomID, setID)
	if err != nil {
		return calisthenics.Set{}, err
	}
	if set.Skipped || set.CompletedAt != nil {
		return calisthenics.Set{}, ErrCalInvalidSet
	}
	set.RepsCompleted++
	now := calisthenics.NowISO()
	if set.RepsCompleted >= set.RepsTarget {
		set.CompletedAt = &now
	}
	if err := s.updateSetRow(ctx, set); err != nil {
		return calisthenics.Set{}, err
	}
	if set.CompletedAt != nil {
		if err := s.afterSetCompleted(ctx, roomID, workoutID, ex.ID); err != nil {
			return calisthenics.Set{}, err
		}
	}
	_ = s.bumpCalRevision(ctx, roomID)
	set.Status = calisthenics.DeriveSetStatus(set)
	return set, nil
}

func (s *Store) CompleteSet(ctx context.Context, roomID, setID string) (calisthenics.Set, error) {
	set, ex, workoutID, err := s.getSetScoped(ctx, roomID, setID)
	if err != nil {
		return calisthenics.Set{}, err
	}
	if set.Skipped {
		return calisthenics.Set{}, ErrCalInvalidSet
	}
	if set.RepsCompleted < set.RepsTarget {
		set.RepsCompleted = set.RepsTarget
	}
	now := calisthenics.NowISO()
	set.CompletedAt = &now
	if err := s.updateSetRow(ctx, set); err != nil {
		return calisthenics.Set{}, err
	}
	if err := s.afterSetCompleted(ctx, roomID, workoutID, ex.ID); err != nil {
		return calisthenics.Set{}, err
	}
	_ = s.bumpCalRevision(ctx, roomID)
	set.Status = calisthenics.DeriveSetStatus(set)
	return set, nil
}

func (s *Store) SkipSet(ctx context.Context, roomID, setID string) (calisthenics.Set, error) {
	set, ex, workoutID, err := s.getSetScoped(ctx, roomID, setID)
	if err != nil {
		return calisthenics.Set{}, err
	}
	set.Skipped = true
	set.RepsCompleted = 0
	set.CompletedAt = nil
	if err := s.updateSetRow(ctx, set); err != nil {
		return calisthenics.Set{}, err
	}
	if err := s.afterSetCompleted(ctx, roomID, workoutID, ex.ID); err != nil {
		return calisthenics.Set{}, err
	}
	_ = s.bumpCalRevision(ctx, roomID)
	set.Status = calisthenics.SetSkipped
	return set, nil
}

func (s *Store) afterSetCompleted(ctx context.Context, roomID, workoutID, exerciseID string) error {
	sets, err := s.listSets(ctx, exerciseID)
	if err != nil {
		return err
	}
	allDone := true
	for _, set := range sets {
		st := calisthenics.DeriveSetStatus(set)
		if st != calisthenics.SetCompleted && st != calisthenics.SetSkipped {
			allDone = false
			break
		}
	}

	if allDone {
		_, err = s.db.ExecContext(ctx, `UPDATE cal_workout_exercises SET status = 'done' WHERE id = ?`, exerciseID)
		if err != nil {
			return err
		}
		var nextID string
		err = s.db.QueryRowContext(ctx, `
			SELECT id FROM cal_workout_exercises
			WHERE workout_id = ? AND status = 'pending'
			ORDER BY sort_order LIMIT 1
		`, workoutID).Scan(&nextID)
		if err == nil {
			_, _ = s.db.ExecContext(ctx, `UPDATE cal_workout_exercises SET status = 'active' WHERE id = ?`, nextID)
		}
	} else {
		rt, err := s.getCalRuntime(ctx, roomID)
		if err != nil {
			return err
		}
		var timersMap map[string]any
		if err := json.Unmarshal(rt.Timers, &timersMap); err != nil {
			timersMap = timers.DefaultCalisthenicsTimers()
		}
		_ = timers.StartTimerInMap(timersMap, "rest", time.Now().UnixMilli())
		updated, _ := json.Marshal(timersMap)
		_, _ = s.db.ExecContext(ctx, `UPDATE cal_runtime SET timers = ? WHERE room_id = ?`, string(updated), roomID)
	}

	return s.syncGoalProgress(ctx, workoutID, roomID)
}

func (s *Store) UpdateCalTimer(ctx context.Context, roomID, timerID, action string, body json.RawMessage) (json.RawMessage, error) {
	rt, err := s.getCalRuntime(ctx, roomID)
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
	_, err = s.db.ExecContext(ctx, `UPDATE cal_runtime SET timers = ?, revision = revision + 1 WHERE room_id = ?`, string(updated), roomID)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(updated), nil
}

func (s *Store) GetCalTimers(ctx context.Context, roomID string) (json.RawMessage, error) {
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return nil, err
	}
	rt, err := s.getCalRuntime(ctx, roomID)
	return rt.Timers, err
}

/* ─── helpers ─── */

func insertExerciseTx(ctx context.Context, tx *sql.Tx, ex calisthenics.Exercise) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO cal_workout_exercises (id, workout_id, name, movement_id, planned_sets, rep_target, status, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ex.ID, ex.WorkoutID, ex.Name, ex.MovementID, ex.PlannedSets, ex.RepTarget, ex.Status, ex.SortOrder)
	return err
}

func insertSetTx(ctx context.Context, tx *sql.Tx, set calisthenics.Set) error {
	skipped := 0
	if set.Skipped {
		skipped = 1
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO cal_sets (id, exercise_id, set_number, reps_target, reps_completed, skipped, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, set.ID, set.ExerciseID, set.SetNumber, set.RepsTarget, set.RepsCompleted, skipped, set.CompletedAt)
	return err
}

func (s *Store) listExercises(ctx context.Context, workoutID string) ([]calisthenics.Exercise, error) {
	return listExercisesTx(ctx, s.db, workoutID)
}

func listExercisesTx(ctx context.Context, q sqlExecutor, workoutID string) ([]calisthenics.Exercise, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, workout_id, name, movement_id, planned_sets, rep_target, status, sort_order
		FROM cal_workout_exercises WHERE workout_id = ? ORDER BY sort_order
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []calisthenics.Exercise
	for rows.Next() {
		var ex calisthenics.Exercise
		if err := rows.Scan(&ex.ID, &ex.WorkoutID, &ex.Name, &ex.MovementID, &ex.PlannedSets, &ex.RepTarget, &ex.Status, &ex.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}

func (s *Store) listSets(ctx context.Context, exerciseID string) ([]calisthenics.Set, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, exercise_id, set_number, reps_target, reps_completed, skipped, completed_at
		FROM cal_sets WHERE exercise_id = ? ORDER BY set_number
	`, exerciseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []calisthenics.Set
	for rows.Next() {
		set, err := scanSet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, set)
	}
	return out, rows.Err()
}

func (s *Store) updateSetRow(ctx context.Context, set calisthenics.Set) error {
	skipped := 0
	if set.Skipped {
		skipped = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE cal_sets SET reps_completed = ?, skipped = ?, completed_at = ? WHERE id = ?
	`, set.RepsCompleted, skipped, set.CompletedAt, set.ID)
	return err
}

func (s *Store) getExerciseScoped(ctx context.Context, roomID, exerciseID string) (calisthenics.Exercise, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.workout_id, e.name, e.movement_id, e.planned_sets, e.rep_target, e.status, e.sort_order, w.room_id
		FROM cal_workout_exercises e
		JOIN cal_workouts w ON w.id = e.workout_id
		WHERE e.id = ? AND w.room_id = ?
	`, exerciseID, roomID)
	var ex calisthenics.Exercise
	var wRoom string
	err := row.Scan(&ex.ID, &ex.WorkoutID, &ex.Name, &ex.MovementID, &ex.PlannedSets, &ex.RepTarget, &ex.Status, &ex.SortOrder, &wRoom)
	if errors.Is(err, sql.ErrNoRows) {
		return calisthenics.Exercise{}, "", ErrNotFound
	}
	return ex, ex.WorkoutID, err
}

func (s *Store) getSetScoped(ctx context.Context, roomID, setID string) (calisthenics.Set, calisthenics.Exercise, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.exercise_id, s.set_number, s.reps_target, s.reps_completed, s.skipped, s.completed_at,
		       e.id, e.workout_id, e.name, e.movement_id, e.planned_sets, e.rep_target, e.status, e.sort_order
		FROM cal_sets s
		JOIN cal_workout_exercises e ON e.id = s.exercise_id
		JOIN cal_workouts w ON w.id = e.workout_id
		WHERE s.id = ? AND w.room_id = ?
	`, setID, roomID)
	var set calisthenics.Set
	var ex calisthenics.Exercise
	var skipped int
	err := row.Scan(
		&set.ID, &set.ExerciseID, &set.SetNumber, &set.RepsTarget, &set.RepsCompleted, &skipped, &set.CompletedAt,
		&ex.ID, &ex.WorkoutID, &ex.Name, &ex.MovementID, &ex.PlannedSets, &ex.RepTarget, &ex.Status, &ex.SortOrder,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return calisthenics.Set{}, calisthenics.Exercise{}, "", ErrNotFound
	}
	set.Skipped = skipped == 1
	set.Status = calisthenics.DeriveSetStatus(set)
	return set, ex, ex.WorkoutID, err
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func scanWorkouts(rows *sql.Rows) ([]calisthenics.Workout, error) {
	var out []calisthenics.Workout
	for rows.Next() {
		w, err := scanWorkout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanWorkout(row scannable) (calisthenics.Workout, error) {
	var w calisthenics.Workout
	err := row.Scan(&w.ID, &w.RoomID, &w.WorkoutType, &w.Status, &w.LiveSessionID, &w.StartedAt, &w.EndedAt, &w.CreatedAt)
	return w, err
}

func scanSet(row scannable) (calisthenics.Set, error) {
	var set calisthenics.Set
	var skipped int
	err := row.Scan(&set.ID, &set.ExerciseID, &set.SetNumber, &set.RepsTarget, &set.RepsCompleted, &skipped, &set.CompletedAt)
	set.Skipped = skipped == 1
	set.Status = calisthenics.DeriveSetStatus(set)
	return set, err
}
