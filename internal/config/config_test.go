package config

import "testing"

func TestSplitCSVStripsQuotes(t *testing.T) {
	got := splitCSV(`https://streamer.woragis.me,https://streamer-frontend.vercel.app`)
	if len(got) != 2 || got[0] != "https://streamer.woragis.me" {
		t.Fatalf("unexpected origins: %#v", got)
	}

	wrapped := splitCSV(`"https://streamer.woragis.me,https://streamer-frontend.vercel.app"`)
	if len(wrapped) != 2 || wrapped[0] != "https://streamer.woragis.me" {
		t.Fatalf("unexpected wrapped origins: %#v", wrapped)
	}
}
