package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/woragis/streamer-backend/internal/platform"
)

type platformSettingsRow struct {
	roomID                string
	youtubeEnabled        bool
	googleAPIKey          string
	youtubeChannelID      string
	youtubeIdleSeconds    int
	kickEnabled           bool
	kickChannelSlug       string
	kickWebhookSkipVerify bool
	updatedAt             string
}

func (s *Store) GetPlatformSettings(ctx context.Context, roomID string) (platform.PlatformSettings, error) {
	row, err := s.getPlatformSettingsRow(ctx, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return platform.DefaultPlatformSettings(roomID), nil
	}
	if err != nil {
		return platform.PlatformSettings{}, err
	}
	return row.toPublic(), nil
}

func (s *Store) GetPlatformSettingsResolved(ctx context.Context, roomID string) (platform.ResolvedSettings, error) {
	row, err := s.getPlatformSettingsRow(ctx, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return platform.DefaultResolvedSettings(roomID), nil
	}
	if err != nil {
		return platform.ResolvedSettings{}, err
	}
	return row.toResolved(), nil
}

func (s *Store) ResolveKickSettings(ctx context.Context, broadcasterSlug string) (platform.ResolvedSettings, bool, error) {
	broadcasterSlug = strings.TrimSpace(strings.ToLower(broadcasterSlug))

	rows, err := s.db.QueryContext(ctx, `
		SELECT room_id, youtube_enabled, google_api_key, youtube_channel_id, youtube_idle_seconds,
		       kick_enabled, kick_channel_slug, kick_webhook_skip_verify, updated_at
		FROM core_platform_settings
		WHERE kick_enabled = 1
	`)
	if err != nil {
		return platform.ResolvedSettings{}, false, err
	}
	defer rows.Close()

	var matches []platformSettingsRow
	for rows.Next() {
		row, err := scanPlatformSettingsRow(rows)
		if err != nil {
			return platform.ResolvedSettings{}, false, err
		}
		slug := strings.TrimSpace(strings.ToLower(row.kickChannelSlug))
		if slug != "" && broadcasterSlug != "" && slug != broadcasterSlug {
			continue
		}
		matches = append(matches, row)
	}
	if err := rows.Err(); err != nil {
		return platform.ResolvedSettings{}, false, err
	}
	if len(matches) == 0 {
		return platform.ResolvedSettings{}, false, nil
	}
	return matches[0].toResolved(), true, nil
}

func (s *Store) ListYouTubeReadyRooms(ctx context.Context) ([]platform.ResolvedSettings, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT room_id, youtube_enabled, google_api_key, youtube_channel_id, youtube_idle_seconds,
		       kick_enabled, kick_channel_slug, kick_webhook_skip_verify, updated_at
		FROM core_platform_settings
		WHERE youtube_enabled = 1
		  AND google_api_key <> ''
		  AND youtube_channel_id <> ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []platform.ResolvedSettings
	for rows.Next() {
		row, err := scanPlatformSettingsRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row.toResolved())
	}
	return out, rows.Err()
}

func (s *Store) UpdatePlatformSettings(ctx context.Context, roomID string, in platform.UpdatePlatformSettingsInput) (platform.PlatformSettings, error) {
	current, err := s.GetPlatformSettingsResolved(ctx, roomID)
	if err != nil {
		return platform.PlatformSettings{}, err
	}

	next := current
	if in.YouTube != nil {
		if in.YouTube.Enabled != nil {
			next.YouTubeEnabled = *in.YouTube.Enabled
		}
		if in.YouTube.APIKey != nil {
			next.GoogleAPIKey = strings.TrimSpace(*in.YouTube.APIKey)
		}
		if in.YouTube.ChannelID != nil {
			next.YouTubeChannelID = strings.TrimSpace(*in.YouTube.ChannelID)
		}
		if in.YouTube.IdleSeconds != nil && *in.YouTube.IdleSeconds > 0 {
			next.YouTubeIdleSeconds = *in.YouTube.IdleSeconds
		}
	}
	if in.Kick != nil {
		if in.Kick.Enabled != nil {
			next.KickEnabled = *in.Kick.Enabled
		}
		if in.Kick.ChannelSlug != nil {
			next.KickChannelSlug = strings.TrimSpace(*in.Kick.ChannelSlug)
		}
		if in.Kick.WebhookSkipVerify != nil {
			next.KickWebhookSkipVerify = *in.Kick.WebhookSkipVerify
		}
	}

	now := platform.NowISO()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO core_platform_settings (
			room_id, youtube_enabled, google_api_key, youtube_channel_id, youtube_idle_seconds,
			kick_enabled, kick_channel_slug, kick_webhook_skip_verify, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(room_id) DO UPDATE SET
			youtube_enabled = excluded.youtube_enabled,
			google_api_key = excluded.google_api_key,
			youtube_channel_id = excluded.youtube_channel_id,
			youtube_idle_seconds = excluded.youtube_idle_seconds,
			kick_enabled = excluded.kick_enabled,
			kick_channel_slug = excluded.kick_channel_slug,
			kick_webhook_skip_verify = excluded.kick_webhook_skip_verify,
			updated_at = excluded.updated_at
	`,
		roomID, boolInt(next.YouTubeEnabled), next.GoogleAPIKey, next.YouTubeChannelID, next.YouTubeIdleSeconds,
		boolInt(next.KickEnabled), next.KickChannelSlug, boolInt(next.KickWebhookSkipVerify), now,
	)
	if err != nil {
		return platform.PlatformSettings{}, err
	}
	return s.GetPlatformSettings(ctx, roomID)
}

func (s *Store) getPlatformSettingsRow(ctx context.Context, roomID string) (platformSettingsRow, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT room_id, youtube_enabled, google_api_key, youtube_channel_id, youtube_idle_seconds,
		       kick_enabled, kick_channel_slug, kick_webhook_skip_verify, updated_at
		FROM core_platform_settings
		WHERE room_id = ?
	`, roomID)
	return scanPlatformSettingsRow(row)
}

func scanPlatformSettingsRow(scanner interface {
	Scan(dest ...any) error
}) (platformSettingsRow, error) {
	var (
		row            platformSettingsRow
		youtubeEnabled int
		kickEnabled    int
		skipVerify     int
	)
	err := scanner.Scan(
		&row.roomID, &youtubeEnabled, &row.googleAPIKey, &row.youtubeChannelID, &row.youtubeIdleSeconds,
		&kickEnabled, &row.kickChannelSlug, &skipVerify, &row.updatedAt,
	)
	if err != nil {
		return platformSettingsRow{}, err
	}
	row.youtubeEnabled = youtubeEnabled == 1
	row.kickEnabled = kickEnabled == 1
	row.kickWebhookSkipVerify = skipVerify == 1
	if row.youtubeIdleSeconds <= 0 {
		row.youtubeIdleSeconds = 30
	}
	return row, nil
}

func (r platformSettingsRow) toPublic() platform.PlatformSettings {
	return platform.PlatformSettings{
		RoomID: r.roomID,
		YouTube: platform.YouTubeSettings{
			Enabled:     r.youtubeEnabled,
			ChannelID:   r.youtubeChannelID,
			HasAPIKey:   r.googleAPIKey != "",
			IdleSeconds: r.youtubeIdleSeconds,
		},
		Kick: platform.KickSettings{
			Enabled:           r.kickEnabled,
			ChannelSlug:       r.kickChannelSlug,
			WebhookSkipVerify: r.kickWebhookSkipVerify,
		},
		UpdatedAt: r.updatedAt,
	}
}

func (r platformSettingsRow) toResolved() platform.ResolvedSettings {
	return platform.ResolvedSettings{
		RoomID:                r.roomID,
		YouTubeEnabled:        r.youtubeEnabled,
		GoogleAPIKey:          r.googleAPIKey,
		YouTubeChannelID:      r.youtubeChannelID,
		YouTubeIdleSeconds:    r.youtubeIdleSeconds,
		KickEnabled:           r.kickEnabled,
		KickChannelSlug:       r.kickChannelSlug,
		KickWebhookSkipVerify: r.kickWebhookSkipVerify,
	}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
