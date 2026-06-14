package youtube

import (
	"encoding/json"
	"fmt"

	"github.com/woragis/streamer-backend/internal/platform"
)

type MappedItem struct {
	Message *platform.IngestMessageInput
	Event   *platform.IngestEventInput
}

func MapChatMessage(item ChatMessage) (MappedItem, bool) {
	username := item.AuthorDetails.DisplayName
	if username == "" {
		username = item.AuthorDetails.ChannelID
	}
	if username == "" {
		username = "anonymous"
	}

	externalID := item.ID
	displayName := item.AuthorDetails.DisplayName

	switch item.Snippet.Type {
	case "textMessageEvent":
		content := item.Snippet.TextMessageDetails.MessageText
		if content == "" {
			content = item.Snippet.DisplayMessage
		}
		if content == "" {
			return MappedItem{}, false
		}
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: displayName,
				Content:     content,
				ExternalID:  externalID,
			},
		}, true

	case "superChatEvent":
		details := item.Snippet.SuperChatDetails
		content := details.UserComment
		if content == "" {
			content = item.Snippet.DisplayMessage
		}
		if content == "" {
			content = fmt.Sprintf("Super Chat %s", details.AmountDisplayString)
		} else {
			content = fmt.Sprintf("[%s] %s", details.AmountDisplayString, content)
		}
		payload, _ := json.Marshal(map[string]string{
			"amount":   details.AmountDisplayString,
			"currency": details.Currency,
			"message":  details.UserComment,
		})
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: displayName,
				Content:     content,
				ExternalID:  externalID + ":msg",
			},
			Event: &platform.IngestEventInput{
				Type:       "donation",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalID,
				Payload:    payload,
			},
		}, true

	case "superStickerEvent":
		details := item.Snippet.SuperStickerDetails
		sticker := details.SuperStickerMetadata.AltText
		if sticker == "" {
			sticker = "Super Sticker"
		}
		content := fmt.Sprintf("[%s] %s", details.AmountDisplayString, sticker)
		payload, _ := json.Marshal(map[string]string{
			"amount":   details.AmountDisplayString,
			"currency": details.Currency,
			"message":  sticker,
		})
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: displayName,
				Content:     content,
				ExternalID:  externalID + ":msg",
			},
			Event: &platform.IngestEventInput{
				Type:       "donation",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalID,
				Payload:    payload,
			},
		}, true

	case "newSponsorEvent":
		payload, _ := json.Marshal(map[string]string{
			"message": "New channel member",
		})
		return MappedItem{
			Event: &platform.IngestEventInput{
				Type:       "subscriber",
				Platform:   PlatformName,
				Username:   username,
				ExternalID: externalID,
				Payload:    payload,
			},
		}, true

	case "memberMilestoneChatEvent":
		content := item.Snippet.DisplayMessage
		if content == "" {
			return MappedItem{}, false
		}
		return MappedItem{
			Message: &platform.IngestMessageInput{
				Platform:    PlatformName,
				Username:    username,
				DisplayName: displayName,
				Content:     content,
				ExternalID:  externalID,
			},
		}, true

	default:
		return MappedItem{}, false
	}
}
