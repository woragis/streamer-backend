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
