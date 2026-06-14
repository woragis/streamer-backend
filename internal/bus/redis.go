package bus

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RedisBus struct {
	rdb        *goredis.Client
	instanceID string
	local      Deliverer
}

func NewRedis(rdb *goredis.Client, instanceID string, local Deliverer) *RedisBus {
	return &RedisBus{rdb: rdb, instanceID: instanceID, local: local}
}

func (b *RedisBus) Publish(ctx context.Context, roomID, domain, eventType string, payload any) error {
	if b == nil || b.rdb == nil {
		return nil
	}
	raw, err := marshalEnvelope(b.instanceID, roomID, domain, eventType, payload)
	if err != nil {
		return err
	}
	return b.rdb.Publish(ctx, RoomChannel(roomID), raw).Err()
}

func (b *RedisBus) StartSubscriber(ctx context.Context) {
	if b == nil || b.rdb == nil || b.local == nil {
		return
	}

	go b.runSubscriber(ctx)
}

func (b *RedisBus) runSubscriber(ctx context.Context) {
	backoff := time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		if err := b.subscribeLoop(ctx); err != nil && ctx.Err() == nil {
			log.Printf("redis subscriber: %v; retry in %s", err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (b *RedisBus) subscribeLoop(ctx context.Context) error {
	pubsub := b.rdb.PSubscribe(ctx, PatternChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return goredis.ErrClosed
			}
			b.handleMessage(msg.Payload)
		}
	}
}

func (b *RedisBus) handleMessage(payload string) {
	ev, err := parseEnvelope([]byte(payload))
	if err != nil {
		log.Printf("redis subscriber: %v", err)
		return
	}
	if ev.OriginID == b.instanceID {
		return
	}
	if ev.RoomID == "" || ev.Type == "" {
		return
	}

	var data any
	if len(ev.Data) > 0 && string(ev.Data) != "null" {
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			b.local.Deliver(context.Background(), ev.RoomID, ev.Domain, ev.Type, json.RawMessage(ev.Data))
			return
		}
	}
	b.local.Deliver(context.Background(), ev.RoomID, ev.Domain, ev.Type, data)
}

func (b *RedisBus) Close() error {
	return nil
}

// RoomIDFromChannel extracts room id from streamer:room:{roomId}:events.
func RoomIDFromChannel(channel string) string {
	channel = strings.TrimPrefix(channel, channelPrefix)
	return strings.TrimSuffix(channel, ":events")
}
