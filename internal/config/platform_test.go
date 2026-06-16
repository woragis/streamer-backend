package config

import "testing"

func TestWorkerRoomIDDefault(t *testing.T) {
	t.Setenv("ROOM_ID", "")
	if got := WorkerRoomID(); got != "default" {
		t.Fatalf("expected default, got %q", got)
	}
}

func TestWorkerRoomIDOverride(t *testing.T) {
	t.Setenv("ROOM_ID", "studio")
	if got := WorkerRoomID(); got != "studio" {
		t.Fatalf("expected studio, got %q", got)
	}
}
