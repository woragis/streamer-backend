package config

import "testing"

func TestNormalizeDatabaseURLRailwayPublicSSL(t *testing.T) {
	raw := "postgres://user:pass@containers-us-west-123.railway.app:6543/railway"
	got := NormalizeDatabaseURL(raw)
	if got != raw+"?sslmode=require" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeDatabaseURLRailwayInternal(t *testing.T) {
	raw := "postgres://user:pass@postgres.railway.internal:5432/railway"
	got := NormalizeDatabaseURL(raw)
	if got != raw+"?sslmode=disable" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeDatabaseURLPreservesExistingSSLMode(t *testing.T) {
	raw := "postgres://user:pass@host/db?sslmode=disable"
	if got := NormalizeDatabaseURL(raw); got != raw {
		t.Fatalf("got %q", got)
	}
}
