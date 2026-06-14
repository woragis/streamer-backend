package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/woragis/streamer-backend/internal/platform"
)

func (s *Store) EnsurePlatform(ctx context.Context, roomID string) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM core_bot_rules WHERE room_id = ?`, roomID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, rule := range platform.DefaultBotRules(roomID) {
		enabled := 0
		if rule.Enabled {
			enabled = 1
		}
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO core_bot_rules (id, room_id, name, enabled, trigger_type, trigger_value, action_type, action_payload, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rule.ID, roomID, rule.Name, enabled, rule.TriggerType, rule.TriggerValue, rule.ActionType, string(rule.ActionPayload), rule.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpsertUser(ctx context.Context, roomID string, platformName, username, displayName string) (platform.User, error) {
	now := platform.NowISO()
	if displayName == "" {
		displayName = username
	}
	id := platform.NewID("user")

	var existingID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM core_users WHERE room_id = ? AND platform = ? AND username = ?
	`, roomID, platformName, username).Scan(&existingID)

	if err == nil {
		_, err = s.db.ExecContext(ctx, `
			UPDATE core_users SET display_name = ?, last_seen_at = ? WHERE id = ?
		`, displayName, now, existingID)
		if err != nil {
			return platform.User{}, err
		}
		return s.getUser(ctx, existingID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return platform.User{}, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO core_users (id, room_id, platform, username, display_name, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, platformName, username, displayName, now, now)
	if err != nil {
		return platform.User{}, err
	}
	return s.getUser(ctx, id)
}

func (s *Store) getUser(ctx context.Context, userID string) (platform.User, error) {
	var u platform.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, room_id, platform, username, display_name, first_seen_at, last_seen_at
		FROM core_users WHERE id = ?
	`, userID).Scan(&u.ID, &u.RoomID, &u.Platform, &u.Username, &u.DisplayName, &u.FirstSeenAt, &u.LastSeenAt)
	return u, err
}

func (s *Store) IngestMessage(ctx context.Context, roomID string, in platform.IngestMessageInput) (platform.IngestResult, error) {
	if in.Platform == "" || in.Username == "" || in.Content == "" {
		return platform.IngestResult{}, fmt.Errorf("platform, username and content required")
	}
	if s.dedup != nil && in.ExternalID != "" {
		ok, err := s.dedup.MarkIfNew(ctx, "msg", in.Platform, in.ExternalID)
		if err != nil {
			return platform.IngestResult{}, err
		}
		if !ok {
			return platform.IngestResult{Duplicate: true}, nil
		}
	}
	result, err := s.ingestMessageDirect(ctx, roomID, in)
	if err != nil && s.dedup != nil && in.ExternalID != "" {
		_ = s.dedup.Release(ctx, "msg", in.Platform, in.ExternalID)
	}
	return result, err
}

func (s *Store) ingestMessageDirect(ctx context.Context, roomID string, in platform.IngestMessageInput) (platform.IngestResult, error) {
	if err := s.EnsurePlatform(ctx, roomID); err != nil {
		return platform.IngestResult{}, err
	}

	user, err := s.UpsertUser(ctx, roomID, in.Platform, in.Username, in.DisplayName)
	if err != nil {
		return platform.IngestResult{}, err
	}

	now := platform.NowISO()
	msgID := platform.NewID("msg")
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO core_messages (id, room_id, user_id, live_session_id, platform, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, msgID, roomID, user.ID, in.LiveSessionID, in.Platform, in.Content, now)
	if err != nil {
		return platform.IngestResult{}, err
	}

	msg := platform.Message{
		ID: msgID, RoomID: roomID, UserID: user.ID, Platform: in.Platform,
		Username: user.Username, DisplayName: user.DisplayName,
		LiveSessionID: in.LiveSessionID, Content: in.Content, CreatedAt: now,
	}

	results, err := s.evaluateRules(ctx, roomID, in.Content)
	if err != nil {
		return platform.IngestResult{}, err
	}

	s.publish(roomID, "all", "message.created", msg)

	return platform.IngestResult{Message: msg, TriggeredRules: results}, nil
}

func (s *Store) IngestStreamEvent(ctx context.Context, roomID string, in platform.IngestEventInput) (platform.StreamEvent, error) {
	if in.Type == "" {
		return platform.StreamEvent{}, fmt.Errorf("type required")
	}
	if s.dedup != nil && in.ExternalID != "" {
		ok, err := s.dedup.MarkIfNew(ctx, "evt", in.Platform, in.ExternalID)
		if err != nil {
			return platform.StreamEvent{}, err
		}
		if !ok {
			return platform.StreamEvent{}, ErrDuplicateIngest
		}
	}
	ev, err := s.ingestStreamEventDirect(ctx, roomID, in)
	if err != nil && s.dedup != nil && in.ExternalID != "" {
		_ = s.dedup.Release(ctx, "evt", in.Platform, in.ExternalID)
	}
	return ev, err
}

func (s *Store) ingestStreamEventDirect(ctx context.Context, roomID string, in platform.IngestEventInput) (platform.StreamEvent, error) {
	now := platform.NowISO()
	id := platform.NewID("evt")
	payload := in.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO core_stream_events (id, room_id, live_session_id, event_type, platform, username, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, in.LiveSessionID, in.Type, in.Platform, in.Username, string(payload), now)
	if err != nil {
		return platform.StreamEvent{}, err
	}

	ev := platform.StreamEvent{
		ID: id, RoomID: roomID, LiveSessionID: in.LiveSessionID,
		Type: in.Type, Platform: in.Platform, Username: in.Username,
		Payload: payload, CreatedAt: now,
	}

	// sync latest alerts into session doc for overlay
	if err := s.applyStreamEventToSession(ctx, roomID, in.Type, in.Username, payload); err != nil {
		return platform.StreamEvent{}, err
	}

	s.publish(roomID, "all", "event.created", ev)
	return ev, nil
}

func (s *Store) applyStreamEventToSession(ctx context.Context, roomID, eventType, username string, payload json.RawMessage) error {
	doc, err := s.GetDocument(ctx, roomID, DocSession)
	if err != nil {
		return err
	}
	var session map[string]any
	if err := json.Unmarshal(doc.Data, &session); err != nil {
		return err
	}
	events, _ := session["streamEvents"].(map[string]any)
	if events == nil {
		events = map[string]any{}
	}

	var val string
	switch eventType {
	case "subscriber":
		if username != "" {
			val = username
		} else {
			var p map[string]any
			_ = json.Unmarshal(payload, &p)
			if a, ok := p["amount"].(string); ok {
				val = username + " - " + a
			}
		}
		events["latestSubscriber"] = val
	case "follower":
		events["latestFollower"] = username
	case "donation":
		var p map[string]any
		_ = json.Unmarshal(payload, &p)
		if msg, ok := p["message"].(string); ok && msg != "" {
			val = username + " - " + msg
		} else if amount, ok := p["amount"].(string); ok {
			val = username + " - " + amount
		} else {
			val = username
		}
		events["latestDonation"] = val
	default:
		return nil
	}

	session["streamEvents"] = events
	updated, _ := json.Marshal(session)
	_, err = s.PutDocument(ctx, roomID, DocSession, updated, nil)
	return err
}

func (s *Store) ListMessages(ctx context.Context, roomID string, limit int, includeDeleted bool) ([]platform.Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `
		SELECT m.id, m.room_id, m.user_id, u.platform, u.username, u.display_name,
		       m.live_session_id, m.content, m.created_at, m.deleted_at
		FROM core_messages m
		JOIN core_users u ON u.id = m.user_id
		WHERE m.room_id = ?
	`
	if !includeDeleted {
		q += ` AND m.deleted_at IS NULL`
	}
	q += ` ORDER BY m.created_at DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, q, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (s *Store) DeleteMessage(ctx context.Context, roomID, messageID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE core_messages SET deleted_at = ? WHERE id = ? AND room_id = ? AND deleted_at IS NULL
	`, platform.NowISO(), messageID, roomID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	s.publish(roomID, "all", "message.deleted", map[string]string{"id": messageID})
	return nil
}

func (s *Store) ListStreamEvents(ctx context.Context, roomID string, limit int) ([]platform.StreamEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, room_id, live_session_id, event_type, platform, username, payload, created_at
		FROM core_stream_events WHERE room_id = ? ORDER BY created_at DESC LIMIT ?
	`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []platform.StreamEvent
	for rows.Next() {
		var ev platform.StreamEvent
		var payload string
		if err := rows.Scan(&ev.ID, &ev.RoomID, &ev.LiveSessionID, &ev.Type, &ev.Platform, &ev.Username, &payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.Payload = json.RawMessage(payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func scanMessages(rows *sql.Rows) ([]platform.Message, error) {
	var out []platform.Message
	for rows.Next() {
		var m platform.Message
		var deleted sql.NullString
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.Platform, &m.Username, &m.DisplayName,
			&m.LiveSessionID, &m.Content, &m.CreatedAt, &deleted); err != nil {
			return nil, err
		}
		m.Deleted = deleted.Valid
		out = append(out, m)
	}
	return out, rows.Err()
}

/* ─── Bot Rules ─── */

func (s *Store) ListRules(ctx context.Context, roomID string) ([]platform.BotRule, error) {
	if err := s.EnsurePlatform(ctx, roomID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, room_id, name, enabled, trigger_type, trigger_value, action_type, action_payload, created_at
		FROM core_bot_rules WHERE room_id = ? ORDER BY created_at
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []platform.BotRule
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRule(ctx context.Context, roomID string, in platform.CreateRuleInput) (platform.BotRule, error) {
	id := platform.NewID("rule")
	now := platform.NowISO()
	enabled := 1
	if in.Enabled != nil && !*in.Enabled {
		enabled = 0
	}
	triggerType := in.TriggerType
	if triggerType == "" {
		triggerType = "keyword"
	}
	payload := in.ActionPayload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO core_bot_rules (id, room_id, name, enabled, trigger_type, trigger_value, action_type, action_payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, roomID, in.Name, enabled, triggerType, in.TriggerValue, in.ActionType, string(payload), now)
	if err != nil {
		return platform.BotRule{}, err
	}
	return s.getRule(ctx, roomID, id)
}

func (s *Store) UpdateRule(ctx context.Context, roomID, ruleID string, in platform.UpdateRuleInput) (platform.BotRule, error) {
	r, err := s.getRule(ctx, roomID, ruleID)
	if err != nil {
		return platform.BotRule{}, err
	}
	if in.Name != nil {
		r.Name = *in.Name
	}
	if in.TriggerValue != nil {
		r.TriggerValue = *in.TriggerValue
	}
	if in.ActionType != nil {
		r.ActionType = *in.ActionType
	}
	if in.ActionPayload != nil {
		r.ActionPayload = *in.ActionPayload
	}
	if in.Enabled != nil {
		r.Enabled = *in.Enabled
	}
	enabled := 0
	if r.Enabled {
		enabled = 1
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE core_bot_rules SET name = ?, enabled = ?, trigger_value = ?, action_type = ?, action_payload = ?
		WHERE id = ? AND room_id = ?
	`, r.Name, enabled, r.TriggerValue, r.ActionType, string(r.ActionPayload), ruleID, roomID)
	if err != nil {
		return platform.BotRule{}, err
	}
	return s.getRule(ctx, roomID, ruleID)
}

func (s *Store) DeleteRule(ctx context.Context, roomID, ruleID string) error {
	if _, err := s.getRule(ctx, roomID, ruleID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM core_bot_rules WHERE id = ? AND room_id = ?`, ruleID, roomID)
	return err
}

func (s *Store) getRule(ctx context.Context, roomID, ruleID string) (platform.BotRule, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, room_id, name, enabled, trigger_type, trigger_value, action_type, action_payload, created_at
		FROM core_bot_rules WHERE id = ? AND room_id = ?
	`, ruleID, roomID)
	r, err := scanRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return platform.BotRule{}, ErrNotFound
	}
	return r, err
}

func scanRule(row scannable) (platform.BotRule, error) {
	var r platform.BotRule
	var enabled int
	var payload string
	err := row.Scan(&r.ID, &r.RoomID, &r.Name, &enabled, &r.TriggerType, &r.TriggerValue, &r.ActionType, &payload, &r.CreatedAt)
	r.Enabled = enabled == 1
	r.ActionPayload = json.RawMessage(payload)
	return r, err
}

func (s *Store) evaluateRules(ctx context.Context, roomID, content string) ([]platform.RuleResult, error) {
	rules, err := s.ListRules(ctx, roomID)
	if err != nil {
		return nil, err
	}
	var results []platform.RuleResult
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.TriggerType != "keyword" || !matchKeyword(content, rule.TriggerValue) {
			continue
		}
		res := platform.RuleResult{
			RuleID: rule.ID, RuleName: rule.Name, ActionType: rule.ActionType,
		}
		switch rule.ActionType {
		case "set_scene":
			var p map[string]string
			_ = json.Unmarshal(rule.ActionPayload, &p)
			scene := p["scene"]
			if scene != "" {
				if err := s.SetScene(ctx, roomID, scene); err == nil {
					res.Applied = true
					res.Detail = rule.ActionPayload
					s.publish(roomID, "all", "rule.triggered", res)
				}
			}
		default:
			continue
		}
		results = append(results, res)
	}
	return results, nil
}

/* ─── Dashboard ─── */

func (s *Store) GetDashboard(ctx context.Context, roomID, month string) (platform.Dashboard, error) {
	if month == "" {
		month = platform.NowISO()[:7]
	}
	prefix := month
	if len(month) == 7 {
		prefix = month + "-"
	}

	lcStats, _ := s.GetLeetCodeStats(ctx, roomID, month, "")
	streak, _ := s.GetLeetCodeStreak(ctx, roomID)
	calStats, _ := s.GetSkillStats(ctx, roomID, month, "")

	var liveCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM live_sessions WHERE room_id = ? AND started_at LIKE ?
	`, roomID, prefix+"%").Scan(&liveCount)

	var workoutCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM cal_workouts WHERE room_id = ? AND created_at LIKE ?
	`, roomID, prefix+"%").Scan(&workoutCount)

	var msgCount, viewerCount int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM core_messages WHERE room_id = ? AND created_at LIKE ? AND deleted_at IS NULL
	`, roomID, prefix+"%").Scan(&msgCount)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM core_messages WHERE room_id = ? AND created_at LIKE ? AND deleted_at IS NULL
	`, roomID, prefix+"%").Scan(&viewerCount)

	timeline, _ := s.buildTimeline(ctx, roomID, prefix)

	return platform.Dashboard{
		Month: month,
		LeetCode: platform.DashboardLeetCode{
			SolvedCount:  lcStats.SolvedCount,
			LiveSessions: liveCount,
			Streak:       streak.Streak,
		},
		Calisthenics: platform.DashboardCal{
			AcquisitionCount: calStats.AcquisitionCount,
			Workouts:         workoutCount,
		},
		Chat: platform.DashboardChat{
			MessageCount:  msgCount,
			UniqueViewers: viewerCount,
		},
		ProficiencyTimeline: timeline,
	}, nil
}

func (s *Store) buildTimeline(ctx context.Context, roomID, monthPrefix string) ([]platform.TimelinePoint, error) {
	var points []platform.TimelinePoint

	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(solved_at, 1, 10) AS day, COUNT(*) FROM lc_problems
		WHERE room_id = ? AND status = 'solved' AND solved_at LIKE ?
		GROUP BY day ORDER BY day
	`, roomID, monthPrefix+"%")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var count int
			if err := rows.Scan(&day, &count); err != nil {
				break
			}
			points = append(points, platform.TimelinePoint{Date: day, Label: "problems_solved", Count: count, Domain: "leetcode"})
		}
	}

	rows2, err := s.db.QueryContext(ctx, `
		SELECT substr(acquired_at, 1, 10) AS day, COUNT(*) FROM cal_skill_acquisitions
		WHERE room_id = ? AND acquired_at LIKE ?
		GROUP BY day ORDER BY day
	`, roomID, monthPrefix+"%")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var day string
			var count int
			if err := rows2.Scan(&day, &count); err != nil {
				break
			}
			points = append(points, platform.TimelinePoint{Date: day, Label: "skills_acquired", Count: count, Domain: "calisthenics"})
		}
	}

	return points, nil
}
