package store_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/store"
)

func TestPutDocumentRevisionConflict(t *testing.T) {
	t.Parallel()

	databaseURL := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := db.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	st := store.New(sqlDB)
	ctx := context.Background()
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}

	doc, err := st.GetDocument(ctx, defaults.DefaultRoomID, store.DocSession)
	if err != nil {
		t.Fatal(err)
	}

	body := json.RawMessage(`{"scene":"brb"}`)
	rev := doc.Revision
	updated, err := st.PutDocument(ctx, defaults.DefaultRoomID, store.DocSession, body, &rev)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != doc.Revision+1 {
		t.Fatalf("expected revision %d, got %d", doc.Revision+1, updated.Revision)
	}

	stale := doc.Revision
	_, err = st.PutDocument(ctx, defaults.DefaultRoomID, store.DocSession, body, &stale)
	if err != store.ErrRevisionConflict {
		t.Fatalf("expected ErrRevisionConflict, got %v", err)
	}
}
