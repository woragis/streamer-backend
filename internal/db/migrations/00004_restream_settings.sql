-- +goose Up
CREATE TABLE IF NOT EXISTS core_restream_settings (
	room_id TEXT PRIMARY KEY,
	enabled INTEGER NOT NULL DEFAULT 0,
	ingest_key TEXT NOT NULL DEFAULT '',
	kick_rtmp_url TEXT NOT NULL DEFAULT '',
	kick_stream_key TEXT NOT NULL DEFAULT '',
	youtube_rtmp_url TEXT NOT NULL DEFAULT 'rtmp://a.rtmp.youtube.com/live2',
	youtube_stream_key TEXT NOT NULL DEFAULT '',
	last_publish_at TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

-- +goose Down
DROP TABLE IF EXISTS core_restream_settings;
