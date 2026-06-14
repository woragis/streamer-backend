package bus_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/woragis/streamer-backend/internal/bus"
	"github.com/woragis/streamer-backend/internal/ws"
)

type spyDeliverer struct {
	hub *ws.Hub
	ch  chan string
}

func (s *spyDeliverer) Deliver(_ context.Context, roomID, domain, eventType string, payload any) {
	if s.hub != nil {
		s.hub.Broadcast(roomID, domain, eventType, payload)
	}
	select {
	case s.ch <- eventType:
	default:
	}
}

func TestCompositeBusFanOut(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	hubA := ws.NewHub(nil)
	compositeA := bus.NewComposite(bus.NewLocal(hubA), bus.NewRedis(rdb, "instance-a", bus.NewLocal(hubA)))

	spy := &spyDeliverer{hub: ws.NewHub(nil), ch: make(chan string, 1)}
	compositeB := bus.NewComposite(nil, bus.NewRedis(rdb, "instance-b", spy))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	compositeA.Start(ctx)
	compositeB.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	if err := compositeA.Publish(context.Background(), "default", "leetcode", "state.updated", map[string]any{
		"revision": 42,
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case eventType := <-spy.ch:
		if eventType != "state.updated" {
			t.Fatalf("expected state.updated, got %s", eventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for redis fan-out on instance B")
	}
}

func TestRoomChannel(t *testing.T) {
	if got := bus.RoomChannel("default"); got != "streamer:room:default:events" {
		t.Fatalf("unexpected channel: %s", got)
	}
}

func TestRedisDisabledLocalOnly(t *testing.T) {
	hub := ws.NewHub(nil)
	local := bus.NewLocal(hub)
	composite := bus.NewComposite(local, nil)

	if err := composite.Publish(context.Background(), "default", "all", "ping", nil); err != nil {
		t.Fatal(err)
	}
}
