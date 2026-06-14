package kick

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/woragis/streamer-backend/internal/platform"
)

type MappedItem struct {
	Message *platform.IngestMessageInput
	Event   *platform.IngestEventInput
}

func MapWebhook(eventType string, body []byte, channelSlug string) (MappedItem, bool, error) {
	switch eventType {
	case EventChatMessage:
		var payload ChatMessagePayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return MappedItem{}, false, err
		}
		if channelSlug != "" && payload.Broadcaster.ChannelSlug != channelSlug {
			return MappedItem{}, false, nil
		}
		username := payload.Sender.Username
		if username == "" {
			username = "anonymous"
		}
		if payload.Content == "" {
			return MappedItem{}, false, nil
		}
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: payload.Sender.Username,
				Content:     payload.Content,
				ExternalID:  payload.MessageID,
			},
		}, true, nil

	case EventChannelFollowed:
		var payload FollowPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return MappedItem{}, false, err
		}
		if channelSlug != "" && payload.Broadcaster.ChannelSlug != channelSlug {
			return MappedItem{}, false, nil
		}
		username := payload.Follower.Username
		if username == "" {
			return MappedItem{}, false, nil
		}
		return MappedItem{
			Event: &platform.IngestEventInput{
				Type:       "follower",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalIDFromBody(eventType, body),
				Payload:    json.RawMessage(body),
			},
		}, true, nil

	case EventSubNew, EventSubRenewal:
		var payload SubscriptionPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return MappedItem{}, false, err
		}
		if channelSlug != "" && payload.Broadcaster.ChannelSlug != channelSlug {
			return MappedItem{}, false, nil
		}
		username := payload.Subscriber.Username
		if username == "" {
			return MappedItem{}, false, nil
		}
		return MappedItem{
			Event: &platform.IngestEventInput{
				Type:       "subscriber",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalIDFromBody(eventType, body),
				Payload:    json.RawMessage(body),
			},
		}, true, nil

	case EventSubGifts:
		var payload SubscriptionGiftsPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return MappedItem{}, false, err
		}
		if channelSlug != "" && payload.Broadcaster.ChannelSlug != channelSlug {
			return MappedItem{}, false, nil
		}
		username := payload.Gifter.Username
		if username == "" && payload.Gifter.IsAnonymous {
			username = "anonymous"
		}
		if username == "" {
			return MappedItem{}, false, nil
		}
		return MappedItem{
			Event: &platform.IngestEventInput{
				Type:       "subscriber",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalIDFromBody(eventType, body),
				Payload:    json.RawMessage(body),
			},
		}, true, nil

	case EventKicksGifted:
		var payload KicksGiftedPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return MappedItem{}, false, err
		}
		if channelSlug != "" && payload.Broadcaster.ChannelSlug != channelSlug {
			return MappedItem{}, false, nil
		}
		username := payload.Sender.Username
		if username == "" {
			return MappedItem{}, false, nil
		}
		amount := strconv.Itoa(payload.Gift.Amount)
		message := payload.Gift.Message
		if message == "" {
			message = payload.Gift.Name
		}
		eventPayload, _ := json.Marshal(map[string]string{
			"amount":  amount,
			"message": message,
			"name":    payload.Gift.Name,
		})
		chatContent := fmt.Sprintf("[%s kicks] %s", amount, message)
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: payload.Sender.Username,
				Content:     chatContent,
				ExternalID:  externalIDFromBody(eventType, body) + ":msg",
			},
			Event: &platform.IngestEventInput{
				Type:       "donation",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalIDFromBody(eventType, body),
				Payload:    eventPayload,
			},
		}, true, nil

	default:
		return MappedItem{}, false, nil
	}
}

func externalIDFromBody(eventType string, body []byte) string {
	var meta struct {
		MessageID string `json:"message_id"`
	}
	_ = json.Unmarshal(body, &meta)
	if meta.MessageID != "" {
		return meta.MessageID
	}
	return eventType + ":" + string(body)
}
