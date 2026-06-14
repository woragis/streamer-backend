package redis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/woragis/streamer-backend/internal/redis"
)

func TestConnectDisabled(t *testing.T) {
	c, err := redis.Connect("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status() != "disabled" {
		t.Fatalf("expected disabled, got %s", c.Status())
	}
}

func TestConnectOK(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	c, err := redis.Connect("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if c.Status() != "ok" {
		t.Fatalf("expected ok, got %s", c.Status())
	}

	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestConnectDown(t *testing.T) {
	c, err := redis.Connect("redis://127.0.0.1:6399")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if c.Status() != "down" {
		t.Fatalf("expected down, got %s", c.Status())
	}
}
