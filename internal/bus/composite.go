package bus

import (
	"context"
)

type CompositeBus struct {
	local  *LocalBus
	redis  *RedisBus
	cancel context.CancelFunc
}

func NewComposite(local *LocalBus, redis *RedisBus) *CompositeBus {
	return &CompositeBus{local: local, redis: redis}
}

func (b *CompositeBus) Start(ctx context.Context) {
	if b == nil || b.redis == nil {
		return
	}
	subCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	b.redis.StartSubscriber(subCtx)
}

func (b *CompositeBus) Publish(ctx context.Context, roomID, domain, eventType string, payload any) error {
	if b == nil || b.local == nil {
		return nil
	}
	b.local.Deliver(ctx, roomID, domain, eventType, payload)
	if b.redis != nil {
		return b.redis.Publish(ctx, roomID, domain, eventType, payload)
	}
	return nil
}

func (b *CompositeBus) Close() error {
	if b.cancel != nil {
		b.cancel()
	}
	if b.redis != nil {
		return b.redis.Close()
	}
	return nil
}
