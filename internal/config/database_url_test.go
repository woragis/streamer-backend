package config

import "testing"

func TestNormalizeDatabaseURLRailwaySSL(t *testing.T) {
	raw := "postgres://user:pass@containers-us-west-123.railway.app:6543/railway"
	got := NormalizeDatabaseURL(raw)
	if got != raw+"?sslmode=require" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeDatabaseURLPreservesExistingSSLMode(t *testing.T) {
	raw := "postgres://user:pass@host/db?sslmode=disable"
	if got := NormalizeDatabaseURL(raw); got != raw {
		t.Fatalf("got %q", got)
	}
}
