package store

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/woragis/streamer-backend/internal/bus"
)

func (s *Store) SetBus(b bus.Bus) {
	s.bus = b
}

func (s *Store) publish(roomID, domain, eventType string, payload any) {
	if s.bus != nil {
		_ = s.bus.Publish(context.Background(), roomID, domain, eventType, payload)
	}
}

func (s *Store) publishState(roomID, domain string, revision int64) {
	s.publish(roomID, domain, "state.updated", map[string]any{
		"revision": revision,
		"domain":   domain,
	})
}

func (s *Store) SetScene(ctx context.Context, roomID, scene string) error {
	doc, err := s.GetDocument(ctx, roomID, DocSession)
	if err != nil {
		return err
	}
	var session map[string]any
	if err := json.Unmarshal(doc.Data, &session); err != nil {
		return err
	}
	session["scene"] = scene
	updated, err := json.Marshal(session)
	if err != nil {
		return err
	}
	newDoc, err := s.PutDocument(ctx, roomID, DocSession, updated, nil)
	if err != nil {
		return err
	}
	s.publish(roomID, "all", "session.updated", map[string]any{
		"scene":    scene,
		"revision": newDoc.Revision,
	})
	return nil
}

func matchKeyword(content, trigger string) bool {
	content = strings.ToLower(strings.TrimSpace(content))
	trigger = strings.ToLower(strings.TrimSpace(trigger))
	if trigger == "" {
		return false
	}
	if strings.HasPrefix(trigger, "!") {
		parts := strings.Fields(content)
		for _, p := range parts {
			if p == trigger {
				return true
			}
		}
		return false
	}
	return strings.Contains(content, trigger)
}
