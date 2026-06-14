package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientActiveLiveChatID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(SearchResponse{
			Items: []struct {
				ID struct {
					VideoID string `json:"videoId"`
				} `json:"id"`
			}{{ID: struct {
				VideoID string `json:"videoId"`
			}{VideoID: "vid123"}}},
		})
	})
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(VideoListResponse{
			Items: []struct {
				ID                 string `json:"id"`
				LiveStreamingDetails struct {
					ActiveLiveChatID string `json:"activeLiveChatId"`
				} `json:"liveStreamingDetails"`
			}{{ID: "vid123", LiveStreamingDetails: struct {
				ActiveLiveChatID string `json:"activeLiveChatId"`
			}{ActiveLiveChatID: "chat456"}}},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient("key", "UC123")
	client.baseURL = srv.URL

	chatID, err := client.ActiveLiveChatID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if chatID != "chat456" {
		t.Fatalf("expected chat456, got %q", chatID)
	}
}

func TestMapChatMessageText(t *testing.T) {
	mapped, ok := MapChatMessage(ChatMessage{
		ID: "msg1",
		Snippet: ChatSnippet{
			Type: "textMessageEvent",
			TextMessageDetails: struct {
				MessageText string `json:"messageText"`
			}{MessageText: "hello world"},
		},
		AuthorDetails: AuthorDetails{DisplayName: "viewer1"},
	})
	if !ok || mapped.Message == nil {
		t.Fatal("expected text message mapping")
	}
	if mapped.Message.Content != "hello world" {
		t.Fatalf("unexpected content: %q", mapped.Message.Content)
	}
	if mapped.Message.ExternalID != "msg1" {
		t.Fatalf("unexpected external id: %q", mapped.Message.ExternalID)
	}
}

func TestMapChatMessageSuperChat(t *testing.T) {
	mapped, ok := MapChatMessage(ChatMessage{
		ID: "sc1",
		Snippet: ChatSnippet{
			Type: "superChatEvent",
			SuperChatDetails: struct {
				AmountDisplayString string `json:"amountDisplayString"`
				AmountMicros          string `json:"amountMicros"`
				Currency              string `json:"currency"`
				UserComment           string `json:"userComment"`
			}{
				AmountDisplayString: "R$ 20,00",
				Currency:            "BRL",
				UserComment:         "ótima live",
			},
		},
		AuthorDetails: AuthorDetails{DisplayName: "donor1"},
	})
	if !ok || mapped.Message == nil || mapped.Event == nil {
		t.Fatal("expected super chat message and event")
	}
	if mapped.Event.Type != "donation" {
		t.Fatalf("expected donation event, got %q", mapped.Event.Type)
	}
}

func TestMapChatMessageNewSponsor(t *testing.T) {
	mapped, ok := MapChatMessage(ChatMessage{
		ID: "sub1",
		Snippet: ChatSnippet{
			Type: "newSponsorEvent",
		},
		AuthorDetails: AuthorDetails{DisplayName: "member1"},
	})
	if !ok || mapped.Event == nil {
		t.Fatal("expected sponsor event")
	}
	if mapped.Event.Type != "subscriber" {
		t.Fatalf("expected subscriber, got %q", mapped.Event.Type)
	}
}
