package youtube

import (
	"encoding/json"
	"testing"
)

func TestMapChatMessageSuperSticker(t *testing.T) {
	mapped, ok := MapChatMessage(ChatMessage{
		ID: "ss1",
		Snippet: ChatSnippet{
			Type: "superStickerEvent",
			SuperStickerDetails: struct {
				AmountDisplayString string `json:"amountDisplayString"`
				AmountMicros          string `json:"amountMicros"`
				Currency              string `json:"currency"`
				SuperStickerMetadata  struct {
					AltText string `json:"altText"`
				} `json:"superStickerMetadata"`
			}{
				AmountDisplayString: "$5.00",
				Currency:            "USD",
				SuperStickerMetadata: struct {
					AltText string `json:"altText"`
				}{AltText: "Cool Sticker"},
			},
		},
		AuthorDetails: AuthorDetails{DisplayName: "sticker_fan"},
	})
	if !ok || mapped.Message == nil || mapped.Event == nil {
		t.Fatal("expected super sticker mapping")
	}
	if mapped.Event.Type != "donation" {
		t.Fatalf("expected donation, got %q", mapped.Event.Type)
	}

	var payload map[string]string
	if err := json.Unmarshal(mapped.Event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["message"] != "Cool Sticker" {
		t.Fatalf("unexpected payload message: %q", payload["message"])
	}
}
