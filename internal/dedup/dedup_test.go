package dedup_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/woragis/streamer-backend/internal/dedup"
)

func TestMarkIfNew(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	s := dedup.New(rdb)
	ctx := context.Background()

	ok, err := s.MarkIfNew(ctx, "msg", "youtube", "ext-123")
	if err != nil || !ok {
		t.Fatalf("expected new, ok=%v err=%v", ok, err)
	}

	ok, err = s.MarkIfNew(ctx, "msg", "youtube", "ext-123")
	if err != nil || ok {
		t.Fatalf("expected duplicate, ok=%v err=%v", ok, err)
	}

	ok, err = s.MarkIfNew(ctx, "msg", "youtube", "")
	if err != nil || !ok {
		t.Fatalf("empty external id should always pass, ok=%v err=%v", ok, err)
	}
}
