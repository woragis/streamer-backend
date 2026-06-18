package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/woragis/streamer-backend/internal/calisthenics"
)

func (s *Store) EnsureSkillCatalog(ctx context.Context, roomID string) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cal_movements WHERE room_id = ?`, roomID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	categories, movements := calisthenics.DefaultSkillCatalog(roomID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, c := range categories {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cal_movement_categories (id, room_id, name) VALUES (?, ?, ?)
			ON CONFLICT(id) DO NOTHING
		`, c.ID, roomID, c.Name); err != nil {
			return err
		}
	}

	now := calisthenics.NowISO()
	for _, m := range movements {
		prereq, _ := json.Marshal(m.Prerequisites)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cal_movements (id, room_id, slug, name, category_id, description, prerequisites)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (id) DO NOTHING
		`, m.ID, roomID, m.Slug, m.Name, m.CategoryID, m.Description, string(prereq)); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cal_movement_proficiencies (room_id, movement_id, level, updated_at)
			VALUES (?, ?, 'unknown', ?)
			ON CONFLICT (room_id, movement_id) DO NOTHING
		`, roomID, m.ID, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) ListMovementCategories(ctx context.Context, roomID string) ([]calisthenics.MovementCategory, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, room_id, name FROM cal_movement_categories WHERE room_id = ? ORDER BY name
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []calisthenics.MovementCategory
	for rows.Next() {
		var c calisthenics.MovementCategory
		if err := rows.Scan(&c.ID, &c.RoomID, &c.Name); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ListMovements(ctx context.Context, roomID, level string) ([]calisthenics.Movement, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return nil, err
	}

	q := `
		SELECT m.id, m.room_id, m.slug, m.name, m.category_id, m.description, m.prerequisites,
		       p.level, p.notes, p.best_hold_seconds, p.best_reps, p.progression_variant, p.updated_at
		FROM cal_movements m
		LEFT JOIN cal_movement_proficiencies p ON p.movement_id = m.id AND p.room_id = m.room_id
		WHERE m.room_id = ?
	`
	args := []any{roomID}
	if level != "" {
		q += ` AND p.level = ?`
		args = append(args, level)
	}
	q += ` ORDER BY m.name`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []calisthenics.Movement
	for rows.Next() {
		m, err := scanMovementWithProficiency(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMovement(ctx context.Context, roomID, movementID string) (calisthenics.Movement, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return calisthenics.Movement{}, err
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT m.id, m.room_id, m.slug, m.name, m.category_id, m.description, m.prerequisites,
		       p.level, p.notes, p.best_hold_seconds, p.best_reps, p.progression_variant, p.updated_at
		FROM cal_movements m
		LEFT JOIN cal_movement_proficiencies p ON p.movement_id = m.id AND p.room_id = m.room_id
		WHERE m.room_id = ? AND m.id = ?
	`, roomID, movementID)
	m, err := scanMovementWithProficiency(row)
	if errors.Is(err, sql.ErrNoRows) {
		return calisthenics.Movement{}, ErrNotFound
	}
	return m, err
}

func (s *Store) CreateMovement(ctx context.Context, roomID string, in calisthenics.CreateMovementInput) (calisthenics.Movement, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return calisthenics.Movement{}, err
	}
	id := in.Slug
	if id == "" {
		id = calisthenics.NewID("mv")
	}
	prereq, _ := json.Marshal(in.Prerequisites)
	now := calisthenics.NowISO()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cal_movements (id, room_id, slug, name, category_id, description, prerequisites)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, in.Slug, in.Name, in.CategoryID, in.Description, string(prereq))
	if err != nil {
		return calisthenics.Movement{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cal_movement_proficiencies (room_id, movement_id, level, updated_at)
		VALUES (?, ?, 'unknown', ?)
	`, roomID, id, now)
	if err != nil {
		return calisthenics.Movement{}, err
	}
	return s.GetMovement(ctx, roomID, id)
}

func (s *Store) UpdateMovement(ctx context.Context, roomID, movementID string, in calisthenics.UpdateMovementInput) (calisthenics.Movement, error) {
	m, err := s.GetMovement(ctx, roomID, movementID)
	if err != nil {
		return calisthenics.Movement{}, err
	}
	if in.Name != nil {
		m.Name = *in.Name
	}
	if in.CategoryID != nil {
		m.CategoryID = *in.CategoryID
	}
	if in.Description != nil {
		m.Description = *in.Description
	}
	if in.Prerequisites != nil {
		m.Prerequisites = in.Prerequisites
	}
	prereq, _ := json.Marshal(m.Prerequisites)
	_, err = s.db.ExecContext(ctx, `
		UPDATE cal_movements SET name = ?, category_id = ?, description = ?, prerequisites = ?
		WHERE id = ? AND room_id = ?
	`, m.Name, m.CategoryID, m.Description, string(prereq), movementID, roomID)
	if err != nil {
		return calisthenics.Movement{}, err
	}
	return s.GetMovement(ctx, roomID, movementID)
}

func (s *Store) DeleteMovement(ctx context.Context, roomID, movementID string) error {
	if _, err := s.GetMovement(ctx, roomID, movementID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM cal_movements WHERE id = ? AND room_id = ?`, movementID, roomID)
	return err
}

func (s *Store) GetProficiency(ctx context.Context, roomID, movementID string) (calisthenics.MovementProficiency, error) {
	if _, err := s.GetMovement(ctx, roomID, movementID); err != nil {
		return calisthenics.MovementProficiency{}, err
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT movement_id, level, notes, best_hold_seconds, best_reps, progression_variant, updated_at
		FROM cal_movement_proficiencies WHERE room_id = ? AND movement_id = ?
	`, roomID, movementID)
	return scanProficiency(row)
}

func (s *Store) UpdateProficiency(ctx context.Context, roomID, movementID string, in calisthenics.UpdateProficiencyInput) (calisthenics.MovementProficiency, error) {
	if !calisthenics.ValidProficiencyLevel(in.Level) {
		return calisthenics.MovementProficiency{}, fmt.Errorf("invalid proficiency level")
	}
	if _, err := s.GetMovement(ctx, roomID, movementID); err != nil {
		return calisthenics.MovementProficiency{}, err
	}

	p, err := s.GetProficiency(ctx, roomID, movementID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return calisthenics.MovementProficiency{}, err
	}

	p.MovementID = movementID
	p.Level = in.Level
	if in.Notes != nil {
		p.Notes = *in.Notes
	}
	if in.BestHoldSeconds != nil {
		p.BestHoldSeconds = in.BestHoldSeconds
	}
	if in.BestReps != nil {
		p.BestReps = in.BestReps
	}
	if in.ProgressionVariant != nil {
		p.ProgressionVariant = in.ProgressionVariant
	}
	p.UpdatedAt = calisthenics.NowISO()

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cal_movement_proficiencies (room_id, movement_id, level, notes, best_hold_seconds, best_reps, progression_variant, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(room_id, movement_id) DO UPDATE SET
			level = excluded.level,
			notes = excluded.notes,
			best_hold_seconds = excluded.best_hold_seconds,
			best_reps = excluded.best_reps,
			progression_variant = excluded.progression_variant,
			updated_at = excluded.updated_at
	`, roomID, movementID, p.Level, p.Notes, p.BestHoldSeconds, p.BestReps, p.ProgressionVariant, p.UpdatedAt)
	if err != nil {
		return calisthenics.MovementProficiency{}, err
	}
	return p, nil
}

func (s *Store) ListAcquisitions(ctx context.Context, roomID, month, liveSessionID, movementID string) ([]calisthenics.SkillAcquisition, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return nil, err
	}

	q := `
		SELECT a.id, a.room_id, a.movement_id, m.name, a.live_session_id, a.acquired_at,
		       a.proficiency_before, a.proficiency_after, a.notes, a.evidence_url, a.acknowledged
		FROM cal_skill_acquisitions a
		JOIN cal_movements m ON m.id = a.movement_id
		WHERE a.room_id = ?
	`
	args := []any{roomID}
	if month != "" {
		prefix := month
		if len(month) == 7 {
			prefix = month + "-"
		}
		q += ` AND a.acquired_at LIKE ?`
		args = append(args, prefix+"%")
	}
	if liveSessionID != "" {
		q += ` AND a.live_session_id = ?`
		args = append(args, liveSessionID)
	}
	if movementID != "" {
		q += ` AND a.movement_id = ?`
		args = append(args, movementID)
	}
	q += ` ORDER BY a.acquired_at DESC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAcquisitions(rows)
}

func (s *Store) CreateAcquisition(ctx context.Context, roomID string, in calisthenics.CreateAcquisitionInput) (calisthenics.SkillAcquisition, error) {
	if !calisthenics.ValidProficiencyLevel(in.ProficiencyAfter) {
		return calisthenics.SkillAcquisition{}, fmt.Errorf("invalid proficiencyAfter level")
	}

	m, err := s.GetMovement(ctx, roomID, in.MovementID)
	if err != nil {
		return calisthenics.SkillAcquisition{}, err
	}

	before := calisthenics.ProficiencyUnknown
	if in.ProficiencyBefore != nil {
		before = *in.ProficiencyBefore
	} else if m.Proficiency != nil {
		before = m.Proficiency.Level
	}

	now := calisthenics.NowISO()
	id := calisthenics.NewID("acq")
	updateProf := true
	if in.UpdateProficiency != nil {
		updateProf = *in.UpdateProficiency
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO cal_skill_acquisitions
		(id, room_id, movement_id, live_session_id, acquired_at, proficiency_before, proficiency_after, notes, evidence_url, acknowledged)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
	`, id, roomID, in.MovementID, in.LiveSessionID, now, before, in.ProficiencyAfter, in.Notes, in.EvidenceURL)
	if err != nil {
		return calisthenics.SkillAcquisition{}, err
	}

	if updateProf {
		notes := in.Notes
		_, err = s.UpdateProficiency(ctx, roomID, in.MovementID, calisthenics.UpdateProficiencyInput{
			Level: in.ProficiencyAfter,
			Notes: &notes,
		})
		if err != nil {
			return calisthenics.SkillAcquisition{}, err
		}
	}

	_ = s.bumpCalRevision(ctx, roomID)

	return calisthenics.SkillAcquisition{
		ID:                id,
		RoomID:            roomID,
		MovementID:        in.MovementID,
		MovementName:      m.Name,
		LiveSessionID:     in.LiveSessionID,
		AcquiredAt:        now,
		ProficiencyBefore: before,
		ProficiencyAfter:  in.ProficiencyAfter,
		Notes:             in.Notes,
		EvidenceURL:       in.EvidenceURL,
		Acknowledged:      false,
	}, nil
}

func (s *Store) GetAcquisition(ctx context.Context, roomID, acquisitionID string) (calisthenics.SkillAcquisition, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.room_id, a.movement_id, m.name, a.live_session_id, a.acquired_at,
		       a.proficiency_before, a.proficiency_after, a.notes, a.evidence_url, a.acknowledged
		FROM cal_skill_acquisitions a
		JOIN cal_movements m ON m.id = a.movement_id
		WHERE a.room_id = ? AND a.id = ?
	`, roomID, acquisitionID)
	items, err := scanAcquisitionsRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return calisthenics.SkillAcquisition{}, ErrNotFound
	}
	return items, err
}

func (s *Store) DeleteAcquisition(ctx context.Context, roomID, acquisitionID string) error {
	if _, err := s.GetAcquisition(ctx, roomID, acquisitionID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM cal_skill_acquisitions WHERE id = ? AND room_id = ?`, acquisitionID, roomID)
	if err != nil {
		return err
	}
	return s.bumpCalRevision(ctx, roomID)
}

func (s *Store) AcknowledgeAcquisition(ctx context.Context, roomID, acquisitionID string) error {
	if _, err := s.GetAcquisition(ctx, roomID, acquisitionID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE cal_skill_acquisitions SET acknowledged = 1 WHERE id = ? AND room_id = ?
	`, acquisitionID, roomID)
	if err != nil {
		return err
	}
	return s.bumpCalRevision(ctx, roomID)
}

func (s *Store) GetPendingSkillAlert(ctx context.Context, roomID string) (*calisthenics.SkillAlert, error) {
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.movement_id, m.name, a.acquired_at, a.notes, a.proficiency_after
		FROM cal_skill_acquisitions a
		JOIN cal_movements m ON m.id = a.movement_id
		WHERE a.room_id = ? AND a.acknowledged = 0
		ORDER BY a.acquired_at DESC LIMIT 1
	`, roomID)

	var alert calisthenics.SkillAlert
	var after string
	err := row.Scan(&alert.AcquisitionID, &alert.MovementID, &alert.MovementName, &alert.AcquiredAt, &alert.Notes, &after)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	alert.ProficiencyAfter = calisthenics.ProficiencyLevel(after)
	return &alert, nil
}

func (s *Store) GetSkillStats(ctx context.Context, roomID, month, liveSessionID string) (calisthenics.SkillStats, error) {
	items, err := s.ListAcquisitions(ctx, roomID, month, liveSessionID, "")
	if err != nil {
		return calisthenics.SkillStats{}, err
	}
	stats := calisthenics.SkillStats{
		AcquisitionCount: len(items),
		Month:          month,
		LiveSessionID:  liveSessionID,
	}
	for _, a := range items {
		stats.MovementIDs = append(stats.MovementIDs, a.MovementID)
	}
	return stats, nil
}

func (s *Store) GetMovementHistory(ctx context.Context, roomID, movementID string) (map[string]any, error) {
	if _, err := s.GetMovement(ctx, roomID, movementID); err != nil {
		return nil, err
	}
	acquisitions, err := s.ListAcquisitions(ctx, roomID, "", "", movementID)
	if err != nil {
		return nil, err
	}
	prof, err := s.GetProficiency(ctx, roomID, movementID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"movementId":   movementID,
		"proficiency":  prof,
		"acquisitions": acquisitions,
	}, nil
}

func (s *Store) CreatePracticeLog(ctx context.Context, roomID string, in calisthenics.CreatePracticeLogInput) error {
	if _, err := s.GetMovement(ctx, roomID, in.MovementID); err != nil {
		return err
	}
	id := calisthenics.NewID("practice")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cal_skill_practice_logs (id, room_id, movement_id, live_session_id, practiced_at, duration_seconds, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, in.MovementID, in.LiveSessionID, calisthenics.NowISO(), in.DurationSeconds, in.Notes)
	return err
}

func scanMovementWithProficiency(row scannable) (calisthenics.Movement, error) {
	var m calisthenics.Movement
	var prereqStr string
	var level, notes, updatedAt sql.NullString
	var bestHold, bestReps sql.NullInt64
	var variant sql.NullString

	err := row.Scan(&m.ID, &m.RoomID, &m.Slug, &m.Name, &m.CategoryID, &m.Description, &prereqStr,
		&level, &notes, &bestHold, &bestReps, &variant, &updatedAt)
	if err != nil {
		return m, err
	}
	_ = json.Unmarshal([]byte(prereqStr), &m.Prerequisites)
	if level.Valid {
		p := calisthenics.MovementProficiency{
			MovementID: m.ID,
			Level:      calisthenics.ProficiencyLevel(level.String),
			Notes:      notes.String,
			UpdatedAt:  updatedAt.String,
		}
		if bestHold.Valid {
			v := int(bestHold.Int64)
			p.BestHoldSeconds = &v
		}
		if bestReps.Valid {
			v := int(bestReps.Int64)
			p.BestReps = &v
		}
		if variant.Valid {
			p.ProgressionVariant = &variant.String
		}
		m.Proficiency = &p
	}
	return m, nil
}

func scanProficiency(row scannable) (calisthenics.MovementProficiency, error) {
	var p calisthenics.MovementProficiency
	var bestHold, bestReps sql.NullInt64
	var variant sql.NullString
	err := row.Scan(&p.MovementID, &p.Level, &p.Notes, &bestHold, &bestReps, &variant, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	if bestHold.Valid {
		v := int(bestHold.Int64)
		p.BestHoldSeconds = &v
	}
	if bestReps.Valid {
		v := int(bestReps.Int64)
		p.BestReps = &v
	}
	if variant.Valid {
		p.ProgressionVariant = &variant.String
	}
	return p, nil
}

func scanAcquisitions(rows *sql.Rows) ([]calisthenics.SkillAcquisition, error) {
	var out []calisthenics.SkillAcquisition
	for rows.Next() {
		a, err := scanAcquisitionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAcquisitionsRow(row scannable) (calisthenics.SkillAcquisition, error) {
	return scanAcquisitionRow(row)
}

func scanAcquisitionRow(row scannable) (calisthenics.SkillAcquisition, error) {
	var a calisthenics.SkillAcquisition
	var before, after string
	var ack int
	err := row.Scan(&a.ID, &a.RoomID, &a.MovementID, &a.MovementName, &a.LiveSessionID, &a.AcquiredAt,
		&before, &after, &a.Notes, &a.EvidenceURL, &ack)
	if err != nil {
		return a, err
	}
	a.ProficiencyBefore = calisthenics.ProficiencyLevel(before)
	a.ProficiencyAfter = calisthenics.ProficiencyLevel(after)
	a.Acknowledged = ack == 1
	return a, nil
}
