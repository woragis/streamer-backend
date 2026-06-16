package platform

type PlatformSettings struct {
	RoomID    string          `json:"roomId"`
	YouTube   YouTubeSettings `json:"youtube"`
	Kick      KickSettings    `json:"kick"`
	UpdatedAt string          `json:"updatedAt"`
}

type YouTubeSettings struct {
	Enabled     bool   `json:"enabled"`
	ChannelID   string `json:"channelId"`
	HasAPIKey   bool   `json:"hasApiKey"`
	IdleSeconds int    `json:"idleSeconds"`
}

type KickSettings struct {
	Enabled           bool   `json:"enabled"`
	ChannelSlug       string `json:"channelSlug"`
	WebhookSkipVerify bool   `json:"webhookSkipVerify"`
}

type UpdatePlatformSettingsInput struct {
	YouTube *UpdateYouTubeSettingsInput `json:"youtube,omitempty"`
	Kick    *UpdateKickSettingsInput    `json:"kick,omitempty"`
}

type UpdateYouTubeSettingsInput struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	APIKey      *string `json:"apiKey,omitempty"`
	ChannelID   *string `json:"channelId,omitempty"`
	IdleSeconds *int    `json:"idleSeconds,omitempty"`
}

type UpdateKickSettingsInput struct {
	Enabled           *bool   `json:"enabled,omitempty"`
	ChannelSlug       *string `json:"channelSlug,omitempty"`
	WebhookSkipVerify *bool   `json:"webhookSkipVerify,omitempty"`
}

// ResolvedSettings is the runtime view used by workers and webhooks (includes secrets).
type ResolvedSettings struct {
	RoomID string

	YouTubeEnabled     bool
	GoogleAPIKey       string
	YouTubeChannelID   string
	YouTubeIdleSeconds int

	KickEnabled           bool
	KickChannelSlug       string
	KickWebhookSkipVerify bool
}

func (r ResolvedSettings) YouTubeReady() bool {
	return r.YouTubeEnabled && r.GoogleAPIKey != "" && r.YouTubeChannelID != ""
}

func (r ResolvedSettings) KickReady() bool {
	return r.KickEnabled
}

func DefaultPlatformSettings(roomID string) PlatformSettings {
	return PlatformSettings{
		RoomID: roomID,
		YouTube: YouTubeSettings{
			Enabled:     false,
			IdleSeconds: 30,
		},
		Kick: KickSettings{
			Enabled: false,
		},
		UpdatedAt: NowISO(),
	}
}

func DefaultResolvedSettings(roomID string) ResolvedSettings {
	return ResolvedSettings{
		RoomID:             roomID,
		YouTubeIdleSeconds: 30,
	}
}
