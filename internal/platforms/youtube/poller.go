package youtube

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/woragis/streamer-backend/internal/store"
)

type Poller struct {
	client      *Client
	store       *store.Store
	roomID      string
	idleSeconds int
}

func NewPoller(client *Client, st *store.Store, roomID string, idleSeconds int) *Poller {
	if idleSeconds <= 0 {
		idleSeconds = 30
	}
	return &Poller{
		client:      client,
		store:       st,
		roomID:      roomID,
		idleSeconds: idleSeconds,
	}
}

func (p *Poller) Run(ctx context.Context) {
	log.Printf("youtube poller started (room=%s channel=%s)", p.roomID, p.client.channelID)

	for {
		if ctx.Err() != nil {
			return
		}
		if err := p.pollLiveChat(ctx); err != nil && ctx.Err() == nil {
			log.Printf("youtube poller: %v", err)
			p.sleep(ctx, time.Duration(p.idleSeconds)*time.Second)
		}
	}
}

func (p *Poller) pollLiveChat(ctx context.Context) error {
	liveChatID, err := p.client.ActiveLiveChatID(ctx)
	if err != nil {
		return err
	}
	if liveChatID == "" {
		log.Printf("youtube: no active live stream for channel %s", p.client.channelID)
		p.sleep(ctx, time.Duration(p.idleSeconds)*time.Second)
		return nil
	}

	log.Printf("youtube: connected to live chat %s", liveChatID)
	pageToken := ""

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		resp, err := p.client.ListChatMessages(ctx, liveChatID, pageToken)
		if err != nil {
			return err
		}

		for _, item := range resp.Items {
			p.ingestItem(ctx, item)
		}

		pageToken = resp.NextPageToken
		interval := resp.PollingIntervalMillis
		if interval <= 0 {
			interval = 5000
		}
		p.sleep(ctx, time.Duration(interval)*time.Millisecond)
	}
}

func (p *Poller) ingestItem(ctx context.Context, item ChatMessage) {
	mapped, ok := MapChatMessage(item)
	if !ok {
		return
	}

	if mapped.Message != nil {
		result, err := p.store.IngestMessage(ctx, p.roomID, *mapped.Message)
		if err != nil {
			log.Printf("youtube ingest message %s: %v", item.ID, err)
			return
		}
		if result.Duplicate {
			return
		}
	}

	if mapped.Event != nil {
		if _, err := p.store.IngestStreamEvent(ctx, p.roomID, *mapped.Event); err != nil {
			if errors.Is(err, store.ErrDuplicateIngest) {
				return
			}
			log.Printf("youtube ingest event %s: %v", item.ID, err)
		}
	}
}

func (p *Poller) sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
