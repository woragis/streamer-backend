package db

import "testing"

func TestRebind(t *testing.T) {
	got := rebind("SELECT * FROM rooms WHERE id = ? AND active_domain = ?")
	want := "SELECT * FROM rooms WHERE id = $1 AND active_domain = $2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
