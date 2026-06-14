package kick

import (
	"testing"
)

func TestMapWebhookChatMessage(t *testing.T) {
	body := []byte(`{
		"message_id": "msg-123",
		"content": "hello kick",
		"sender": {"username": "viewer1", "channel_slug": "viewer1"},
		"broadcaster": {"username": "streamer", "channel_slug": "mystream"}
	}`)

	mapped, ok, err := MapWebhook(EventChatMessage, body, "mystream", "")
	if err != nil || !ok || mapped.Message == nil {
		t.Fatalf("expected chat mapping, ok=%v err=%v mapped=%+v", ok, err, mapped)
	}
	if mapped.Message.Content != "hello kick" {
		t.Fatalf("unexpected content: %q", mapped.Message.Content)
	}
	if mapped.Message.ExternalID != "msg-123" {
		t.Fatalf("unexpected external id: %q", mapped.Message.ExternalID)
	}
}

func TestMapWebhookChatMessageWrongChannel(t *testing.T) {
	body := []byte(`{
		"message_id": "msg-123",
		"content": "hello",
		"sender": {"username": "viewer1"},
		"broadcaster": {"channel_slug": "other"}
	}`)

	_, ok, err := MapWebhook(EventChatMessage, body, "mystream", "")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected event to be ignored for other channel")
	}
}

func TestMapWebhookKicksGifted(t *testing.T) {
	body := []byte(`{
		"message_id": "gift-1",
		"broadcaster": {"channel_slug": "mystream"},
		"sender": {"username": "tipper"},
		"gift": {"amount": 500, "name": "Rage Quit", "message": "nice stream"}
	}`)

	mapped, ok, err := MapWebhook(EventKicksGifted, body, "mystream", "hdr-gift-1")
	if err != nil || !ok {
		t.Fatalf("expected kicks gifted mapping: ok=%v err=%v", ok, err)
	}
	if mapped.Event == nil || mapped.Event.Type != "donation" {
		t.Fatalf("expected donation event, got %+v", mapped.Event)
	}
	if mapped.Message == nil {
		t.Fatal("expected chat message for kicks gift")
	}
}

func TestMapWebhookFollower(t *testing.T) {
	body := []byte(`{
		"broadcaster": {"channel_slug": "mystream"},
		"follower": {"username": "newfan"}
	}`)

	mapped, ok, err := MapWebhook(EventChannelFollowed, body, "mystream", "hdr-follow-1")
	if err != nil || !ok || mapped.Event == nil {
		t.Fatalf("expected follower event: ok=%v err=%v", ok, err)
	}
	if mapped.Event.Type != "follower" {
		t.Fatalf("expected follower, got %q", mapped.Event.Type)
	}
}

func TestDefaultPublicKeyParsed(t *testing.T) {
	pub, err := DefaultPublicKeyParsed()
	if err != nil {
		t.Fatal(err)
	}
	if pub == nil {
		t.Fatal("expected public key")
	}
}
