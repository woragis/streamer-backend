package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/woragis/streamer-backend/internal/restream"
)

type restreamRow struct {
	roomID           string
	enabled          bool
	ingestKey        string
	kickRTMPURL      string
	kickStreamKey    string
	youtubeRTMPURL   string
	youtubeStreamKey string
	lastPublishAt    string
	updatedAt        string
}

func (s *Store) GetRestreamSettingsPublic(ctx context.Context, roomID, obsServer string) (restream.Settings, error) {
	row, err := s.ensureRestreamRow(ctx, roomID)
	if err != nil {
		return restream.Settings{}, err
	}
	return row.toPublic(obsServer, false), nil
}

func (s *Store) GetRestreamSettingsResolved(ctx context.Context, roomID string) (restream.ResolvedSettings, error) {
	row, err := s.ensureRestreamRow(ctx, roomID)
	if err != nil {
		return restream.ResolvedSettings{}, err
	}
	return row.toResolved(), nil
}

func (s *Store) UpdateRestreamSettings(ctx context.Context, roomID string, in restream.UpdateInput) (restream.Settings, error) {
	row, err := s.ensureRestreamRow(ctx, roomID)
	if err != nil {
		return restream.Settings{}, err
	}

	if in.Enabled != nil {
		row.enabled = *in.Enabled
	}
	if in.KickRTMPURL != nil {
		row.kickRTMPURL = strings.TrimSpace(*in.KickRTMPURL)
	}
	if in.KickStreamKey != nil {
		row.kickStreamKey = strings.TrimSpace(*in.KickStreamKey)
	}
	if in.YouTubeRTMPURL != nil {
		row.youtubeRTMPURL = strings.TrimSpace(*in.YouTubeRTMPURL)
		if row.youtubeRTMPURL == "" {
			row.youtubeRTMPURL = restream.DefaultYouTubeRTMPURL
		}
	}
	if in.YouTubeKey != nil {
		row.youtubeStreamKey = strings.TrimSpace(*in.YouTubeKey)
	}
	if in.IngestKey != nil {
		key := strings.TrimSpace(*in.IngestKey)
		if key != "" {
			row.ingestKey = key
		}
	}

	row.updatedAt = restream.NowISO()
	if err := s.saveRestreamRow(ctx, row); err != nil {
		return restream.Settings{}, err
	}
	return row.toPublic("", false), nil
}

func (s *Store) RegenerateRestreamIngestKey(ctx context.Context, roomID string) (restream.Settings, error) {
	row, err := s.ensureRestreamRow(ctx, roomID)
	if err != nil {
		return restream.Settings{}, err
	}
	key, err := restream.GenerateIngestKey()
	if err != nil {
		return restream.Settings{}, err
	}
	row.ingestKey = key
	row.updatedAt = restream.NowISO()
	if err := s.saveRestreamRow(ctx, row); err != nil {
		return restream.Settings{}, err
	}
	pub := row.toPublic("", true)
	return pub, nil
}

func (s *Store) ValidateRestreamPublish(ctx context.Context, path, key string) (bool, error) {
	roomID, ok := restream.RoomFromIngestPath(path)
	if !ok {
		return false, nil
	}
	row, err := s.getRestreamRow(ctx, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !row.enabled || row.ingestKey == "" {
		return false, nil
	}
	key = strings.TrimSpace(key)
	if key == "" || key != row.ingestKey {
		return false, nil
	}
	now := restream.NowISO()
	_, err = s.db.ExecContext(ctx, `
		UPDATE core_restream_settings SET last_publish_at = ?, updated_at = ? WHERE room_id = ?
	`, now, now, roomID)
	return true, err
}

func (s *Store) ensureRestreamRow(ctx context.Context, roomID string) (restreamRow, error) {
	row, err := s.getRestreamRow(ctx, roomID)
	if err == nil {
		if row.youtubeRTMPURL == "" {
			row.youtubeRTMPURL = restream.DefaultYouTubeRTMPURL
		}
		return row, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return restreamRow{}, err
	}
	key, err := restream.GenerateIngestKey()
	if err != nil {
		return restreamRow{}, err
	}
	row = restreamRow{
		roomID:           roomID,
		ingestKey:        key,
		youtubeRTMPURL:   restream.DefaultYouTubeRTMPURL,
		updatedAt:        restream.NowISO(),
	}
	if err := s.saveRestreamRow(ctx, row); err != nil {
		return restreamRow{}, err
	}
	return row, nil
}

func (s *Store) getRestreamRow(ctx context.Context, roomID string) (restreamRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT room_id, enabled, ingest_key, kick_rtmp_url, kick_stream_key,
		       youtube_rtmp_url, youtube_stream_key, last_publish_at, updated_at
		FROM core_restream_settings
		WHERE room_id = ?
	`, roomID)
	return scanRestreamRow(row)
}

func (s *Store) saveRestreamRow(ctx context.Context, row restreamRow) error {
	if row.youtubeRTMPURL == "" {
		row.youtubeRTMPURL = restream.DefaultYouTubeRTMPURL
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO core_restream_settings (
			room_id, enabled, ingest_key, kick_rtmp_url, kick_stream_key,
			youtube_rtmp_url, youtube_stream_key, last_publish_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(room_id) DO UPDATE SET
			enabled = excluded.enabled,
			ingest_key = excluded.ingest_key,
			kick_rtmp_url = excluded.kick_rtmp_url,
			kick_stream_key = excluded.kick_stream_key,
			youtube_rtmp_url = excluded.youtube_rtmp_url,
			youtube_stream_key = excluded.youtube_stream_key,
			last_publish_at = excluded.last_publish_at,
			updated_at = excluded.updated_at
	`,
		row.roomID, boolInt(row.enabled), row.ingestKey, row.kickRTMPURL, row.kickStreamKey,
		row.youtubeRTMPURL, row.youtubeStreamKey, row.lastPublishAt, row.updatedAt,
	)
	return err
}

func scanRestreamRow(scanner interface {
	Scan(dest ...any) error
}) (restreamRow, error) {
	var (
		row     restreamRow
		enabled int
	)
	err := scanner.Scan(
		&row.roomID, &enabled, &row.ingestKey, &row.kickRTMPURL, &row.kickStreamKey,
		&row.youtubeRTMPURL, &row.youtubeStreamKey, &row.lastPublishAt, &row.updatedAt,
	)
	if err != nil {
		return restreamRow{}, err
	}
	row.enabled = enabled == 1
	return row, nil
}

func (r restreamRow) toResolved() restream.ResolvedSettings {
	return restream.ResolvedSettings{
		RoomID:           r.roomID,
		Enabled:          r.enabled,
		IngestKey:        r.ingestKey,
		KickRTMPURL:      r.kickRTMPURL,
		KickStreamKey:    r.kickStreamKey,
		YouTubeRTMPURL:   r.youtubeRTMPURL,
		YouTubeStreamKey: r.youtubeStreamKey,
	}
}

func (r restreamRow) toPublic(obsServer string, revealIngestKey bool) restream.Settings {
	out := restream.Settings{
		RoomID:         r.roomID,
		Enabled:        r.enabled,
		IngestPath:     restream.IngestPathForRoom(r.roomID),
		HasIngestKey:   r.ingestKey != "",
		HasKickKey:     r.kickStreamKey != "",
		HasYouTubeKey:  r.youtubeStreamKey != "",
		KickRTMPURL:    r.kickRTMPURL,
		YouTubeRTMPURL: r.youtubeRTMPURL,
		ObsServer:      strings.TrimRight(obsServer, "/"),
		LastPublishAt:  r.lastPublishAt,
		UpdatedAt:      r.updatedAt,
	}
	if obsServer != "" && r.ingestKey != "" {
		out.ObsStreamKey = r.ingestKey
	}
	if !revealIngestKey {
		out.IngestKey = ""
		out.ObsStreamKey = ""
	} else if obsServer != "" {
		out.ObsStreamKey = r.ingestKey
	}
	return out
}
