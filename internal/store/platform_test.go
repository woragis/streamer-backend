package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/defaults"
	"github.com/woragis/streamer-backend/internal/platform"
)

func TestPlatformIngestAndRules(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roomID := defaults.DefaultRoomID

	result, err := st.IngestMessage(ctx, roomID, platform.IngestMessageInput{
		Platform: "youtube",
		Username: "viewer1",
		Content:  "!brb stream break",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "!brb stream break" {
		t.Fatalf("unexpected message: %+v", result.Message)
	}
	if len(result.TriggeredRules) != 1 || !result.TriggeredRules[0].Applied {
		t.Fatalf("expected !brb rule applied, got %+v", result.TriggeredRules)
	}

	doc, err := st.GetDocument(ctx, roomID, DocSession)
	if err != nil {
		t.Fatal(err)
	}
	var session map[string]any
	if err := json.Unmarshal(doc.Data, &session); err != nil {
		t.Fatal(err)
	}
	if session["scene"] != "brb" {
		t.Fatalf("expected scene brb, got %v", session["scene"])
	}

	rules, err := st.ListRules(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) < 3 {
		t.Fatalf("expected seeded rules, got %d", len(rules))
	}

	msgs, err := st.ListMessages(ctx, roomID, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	dash, err := st.GetDashboard(ctx, roomID, "")
	if err != nil {
		t.Fatal(err)
	}
	if dash.Chat.MessageCount != 1 {
		t.Fatalf("expected 1 chat message in dashboard, got %d", dash.Chat.MessageCount)
	}
}

func TestPlatformStreamEvent(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	st := New(database)
	if err := st.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	roomID := defaults.DefaultRoomID

	_, err = st.IngestStreamEvent(ctx, roomID, platform.IngestEventInput{
		Type:     "follower",
		Platform: "kick",
		Username: "newfan",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc, err := st.GetDocument(ctx, roomID, DocSession)
	if err != nil {
		t.Fatal(err)
	}
	var session map[string]any
	if err := json.Unmarshal(doc.Data, &session); err != nil {
		t.Fatal(err)
	}
	events, _ := session["streamEvents"].(map[string]any)
	if events["latestFollower"] != "newfan" {
		t.Fatalf("expected latestFollower newfan, got %v", events["latestFollower"])
	}
}
