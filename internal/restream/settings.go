package restream

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const DefaultYouTubeRTMPURL = "rtmp://a.rtmp.youtube.com/live2"

type Settings struct {
	RoomID         string `json:"roomId"`
	Enabled        bool   `json:"enabled"`
	IngestPath     string `json:"ingestPath"`
	IngestKey      string `json:"ingestKey,omitempty"`
	HasIngestKey   bool   `json:"hasIngestKey"`
	HasKickKey     bool   `json:"hasKickKey"`
	HasYouTubeKey  bool   `json:"hasYouTubeKey"`
	KickRTMPURL    string `json:"kickRtmpUrl"`
	YouTubeRTMPURL string `json:"youtubeRtmpUrl"`
	ObsServer      string `json:"obsServer"`
	ObsStreamKey   string `json:"obsStreamKey,omitempty"`
	LastPublishAt  string `json:"lastPublishAt,omitempty"`
	UpdatedAt      string `json:"updatedAt"`
}

type UpdateInput struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	IngestKey      *string `json:"ingestKey,omitempty"`
	KickStreamKey  *string `json:"kickStreamKey,omitempty"`
	KickRTMPURL    *string `json:"kickRtmpUrl,omitempty"`
	YouTubeKey     *string `json:"youtubeStreamKey,omitempty"`
	YouTubeRTMPURL *string `json:"youtubeRtmpUrl,omitempty"`
}

type ResolvedSettings struct {
	RoomID           string
	Enabled          bool
	IngestKey        string
	KickRTMPURL      string
	KickStreamKey    string
	YouTubeRTMPURL   string
	YouTubeStreamKey string
}

type RelayDestination struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type RelayConfig struct {
	RoomID       string             `json:"roomId"`
	SourceURL    string             `json:"sourceUrl"`
	Destinations []RelayDestination `json:"destinations"`
}

func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func IngestPathForRoom(roomID string) string {
	return "live/" + strings.TrimSpace(roomID)
}

func RoomFromIngestPath(path string) (string, bool) {
	path = strings.TrimSpace(strings.TrimPrefix(path, "/"))
	if !strings.HasPrefix(path, "live/") {
		return "", false
	}
	room := strings.TrimSpace(strings.TrimPrefix(path, "live/"))
	if room == "" {
		return "", false
	}
	return room, true
}

func GenerateIngestKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "wrgs_" + hex.EncodeToString(buf), nil
}

func CombineRTMPURL(base, streamKey string) string {
	base = strings.TrimSpace(base)
	streamKey = strings.TrimSpace(streamKey)
	if base == "" || streamKey == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + streamKey
}

func (r ResolvedSettings) Ready() bool {
	if !r.Enabled || r.IngestKey == "" {
		return false
	}
	hasKick := r.KickRTMPURL != "" && r.KickStreamKey != ""
	hasYouTube := r.YouTubeRTMPURL != "" && r.YouTubeStreamKey != ""
	return hasKick || hasYouTube
}

func (r ResolvedSettings) RelayConfig(sourceBase string) RelayConfig {
	sourceBase = strings.TrimRight(strings.TrimSpace(sourceBase), "/")
	cfg := RelayConfig{
		RoomID:    r.RoomID,
		SourceURL: fmt.Sprintf("%s/%s", sourceBase, IngestPathForRoom(r.RoomID)),
	}
	if kick := CombineRTMPURL(r.KickRTMPURL, r.KickStreamKey); kick != "" {
		cfg.Destinations = append(cfg.Destinations, RelayDestination{Label: "kick", URL: kick})
	}
	if yt := CombineRTMPURL(r.YouTubeRTMPURL, r.YouTubeStreamKey); yt != "" {
		cfg.Destinations = append(cfg.Destinations, RelayDestination{Label: "youtube", URL: yt})
	}
	return cfg
}
