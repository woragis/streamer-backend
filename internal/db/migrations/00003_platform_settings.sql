-- +goose Up
CREATE TABLE IF NOT EXISTS core_platform_settings (
	room_id TEXT PRIMARY KEY,
	youtube_enabled INTEGER NOT NULL DEFAULT 0,
	google_api_key TEXT NOT NULL DEFAULT '',
	youtube_channel_id TEXT NOT NULL DEFAULT '',
	youtube_idle_seconds INTEGER NOT NULL DEFAULT 30,
	kick_enabled INTEGER NOT NULL DEFAULT 0,
	kick_channel_slug TEXT NOT NULL DEFAULT '',
	kick_webhook_skip_verify INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (room_id) REFERENCES rooms(id)
);

-- +goose Down
DROP TABLE IF EXISTS core_platform_settings;
