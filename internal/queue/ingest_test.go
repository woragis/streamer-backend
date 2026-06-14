package queue_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/woragis/streamer-backend/internal/queue"
)

func TestIngestQueueEnqueueAndConsume(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	q := queue.New(rdb)
	ctx := context.Background()

	if err := q.EnsureGroup(ctx); err != nil {
		t.Fatal(err)
	}

	jobID, err := q.Enqueue(ctx, queue.JobChatMessage, "default", map[string]string{
		"platform": "youtube",
		"username": "viewer1",
		"content":  "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if jobID == "" {
		t.Fatal("expected job id")
	}

	var mu sync.Mutex
	var processed []queue.Job
	done := make(chan struct{}, 1)

	consumerCtx, cancel := context.WithCancel(ctx)
	q.RunConsumer(consumerCtx, "test-consumer", func(ctx context.Context, job queue.Job) error {
		mu.Lock()
		processed = append(processed, job)
		mu.Unlock()
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for consumer")
	}
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if len(processed) != 1 {
		t.Fatalf("expected 1 job, got %d", len(processed))
	}
	if processed[0].JobType != queue.JobChatMessage {
		t.Fatalf("unexpected job type: %s", processed[0].JobType)
	}

	var payload map[string]string
	if err := json.Unmarshal(processed[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["content"] != "hello" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestIngestQueueDisabled(t *testing.T) {
	q := queue.New(nil)
	if q.Enabled() {
		t.Fatal("expected disabled queue")
	}
	_, err := q.Enqueue(context.Background(), queue.JobChatMessage, "default", nil)
	if err != queue.ErrDisabled {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}
