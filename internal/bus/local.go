package bus

import (
	"context"

	"github.com/woragis/streamer-backend/internal/ws"
)

type LocalBus struct {
	hub *ws.Hub
}

func NewLocal(hub *ws.Hub) *LocalBus {
	return &LocalBus{hub: hub}
}

func (b *LocalBus) Deliver(_ context.Context, roomID, domain, eventType string, payload any) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.Broadcast(roomID, domain, eventType, payload)
}

func (b *LocalBus) Publish(ctx context.Context, roomID, domain, eventType string, payload any) error {
	b.Deliver(ctx, roomID, domain, eventType, payload)
	return nil
}

func (b *LocalBus) Close() error {
	return nil
}
