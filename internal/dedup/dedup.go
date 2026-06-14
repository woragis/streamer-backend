package dedup

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const defaultTTL = 48 * time.Hour

type Store struct {
	rdb *goredis.Client
	ttl time.Duration
}

func New(rdb *goredis.Client) *Store {
	if rdb == nil {
		return nil
	}
	return &Store{rdb: rdb, ttl: defaultTTL}
}

func (s *Store) Enabled() bool {
	return s != nil && s.rdb != nil
}

func (s *Store) key(kind, platform, externalID string) string {
	return fmt.Sprintf("streamer:dedup:%s:%s:%s", kind, platform, externalID)
}

// MarkIfNew returns true when the id was not seen before.
func (s *Store) MarkIfNew(ctx context.Context, kind, platform, externalID string) (bool, error) {
	if !s.Enabled() || externalID == "" {
		return true, nil
	}
	ok, err := s.rdb.SetNX(ctx, s.key(kind, platform, externalID), "1", s.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (s *Store) Release(ctx context.Context, kind, platform, externalID string) error {
	if !s.Enabled() || externalID == "" {
		return nil
	}
	return s.rdb.Del(ctx, s.key(kind, platform, externalID)).Err()
}
