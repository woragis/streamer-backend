package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/woragis/streamer-backend/internal/bus"
	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/dedup"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/queue"
)

var ErrNotFound = errors.New("not found")
var ErrRevisionConflict = errors.New("revision conflict")
var ErrDuplicateIngest = errors.New("duplicate ingest")

const (
	DocBranding     = "branding"
	DocSession      = "session"
	DocStreamTimer  = "stream_timer"
	DocLeetCode     = "leetcode_state"
	DocCalisthenics = "calisthenics_state"
)

type Document struct {
	Data      json.RawMessage
	Revision  int64
	UpdatedAt string
}

type Store struct {
	db    *db.DB
	bus   bus.Bus
	queue *queue.IngestQueue
	dedup *dedup.Store
}

func New(database *db.DB) *Store {
	return &Store{db: database}
}

func (s *Store) SetQueue(q *queue.IngestQueue) {
	s.queue = q
}

func (s *Store) SetDedup(d *dedup.Store) {
	s.dedup = d
}

func (s *Store) QueueEnabled() bool {
	return s.queue != nil && s.queue.Enabled()
}

func (s *Store) Seed(ctx context.Context) error {
	rooms := []struct {
		id     string
		domain string
	}{
		{defaults.DefaultRoomID, "leetcode"},
		{defaults.RoomCodes, "leetcode"},
		{defaults.RoomCalisthenics, "calisthenics"},
	}

	for _, room := range rooms {
		if err := s.seedRoom(ctx, room.id, room.domain); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seedRoom(ctx context.Context, roomID, activeDomain string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO rooms (id, active_domain, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, roomID, activeDomain, now, now)
	if err != nil {
		return fmt.Errorf("seed room %s: %w", roomID, err)
	}

	seeds := map[string]json.RawMessage{
		DocBranding:     defaults.BrandingForRoom(roomID),
		DocSession:      defaults.Session(),
		DocStreamTimer:  defaults.StreamTimer(),
		DocLeetCode:     defaults.LeetCodeState(),
		DocCalisthenics: defaults.CalisthenicsState(),
	}

	for key, data := range seeds {
		if err := insertDocIfMissing(ctx, tx, roomID, key, data, now); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := s.EnsureLeetCode(ctx, roomID); err != nil {
		return err
	}
	if err := s.EnsureCalisthenics(ctx, roomID); err != nil {
		return err
	}
	if err := s.EnsureSkillCatalog(ctx, roomID); err != nil {
		return err
	}
	return s.EnsurePlatform(ctx, roomID)
}

func insertDocIfMissing(ctx context.Context, tx *db.Tx, roomID, key string, data json.RawMessage, now string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO room_documents (room_id, doc_key, data, revision, updated_at)
		VALUES (?, ?, ?, 1, ?)
		ON CONFLICT(room_id, doc_key) DO NOTHING
	`, roomID, key, string(data), now)
	if err != nil {
		return fmt.Errorf("seed %s: %w", key, err)
	}
	return nil
}

func (s *Store) GetDocument(ctx context.Context, roomID, key string) (Document, error) {
	var doc Document
	var data string
	err := s.db.QueryRowContext(ctx, `
		SELECT data, revision, updated_at
		FROM room_documents
		WHERE room_id = ? AND doc_key = ?
	`, roomID, key).Scan(&data, &doc.Revision, &doc.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, err
	}
	doc.Data = json.RawMessage(data)
	return doc, nil
}

func (s *Store) PutDocument(ctx context.Context, roomID, key string, data json.RawMessage, expectedRevision *int64) (Document, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	if expectedRevision != nil {
		res, err := s.db.ExecContext(ctx, `
			UPDATE room_documents
			SET data = ?, revision = revision + 1, updated_at = ?
			WHERE room_id = ? AND doc_key = ? AND revision = ?
		`, string(data), now, roomID, key, *expectedRevision)
		if err != nil {
			return Document{}, err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			current, getErr := s.GetDocument(ctx, roomID, key)
			if getErr != nil {
				return Document{}, getErr
			}
			if current.Revision != *expectedRevision {
				return Document{}, ErrRevisionConflict
			}
			return Document{}, ErrNotFound
		}
	} else {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO room_documents (room_id, doc_key, data, revision, updated_at)
			VALUES (?, ?, ?, 1, ?)
			ON CONFLICT(room_id, doc_key) DO UPDATE SET
				data = excluded.data,
				revision = room_documents.revision + 1,
				updated_at = excluded.updated_at
		`, roomID, key, string(data), now)
		if err != nil {
			return Document{}, err
		}
	}

	return s.GetDocument(ctx, roomID, key)
}

func (s *Store) RoomExists(ctx context.Context, roomID string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM rooms WHERE id = ?`, roomID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func MergeRevision(data json.RawMessage, revision int64) json.RawMessage {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		out := map[string]any{"revision": revision, "payload": json.RawMessage(data)}
		b, _ := json.Marshal(out)
		return b
	}
	obj["revision"] = revision
	b, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	return b
}

func ParseExpectedRevision(ifMatch string, body json.RawMessage) *int64 {
	if ifMatch != "" {
		var rev int64
		if _, err := fmt.Sscan(ifMatch, &rev); err == nil {
			return &rev
		}
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil
	}
	raw, ok := obj["revision"]
	if !ok {
		return nil
	}
	var rev int64
	if err := json.Unmarshal(raw, &rev); err != nil {
		return nil
	}
	return &rev
}

func StripRevisionField(body json.RawMessage) json.RawMessage {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return body
	}
	delete(obj, "revision")
	b, err := json.Marshal(obj)
	if err != nil {
		return body
	}
	return b
}
