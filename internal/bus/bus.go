package bus

import (
	"context"
	"encoding/json"
	"fmt"
)

const channelPrefix = "streamer:room:"

// RoomChannel returns the Redis pub/sub channel for a room.
func RoomChannel(roomID string) string {
	return channelPrefix + roomID + ":events"
}

// PatternChannel matches all room event channels.
const PatternChannel = channelPrefix + "*:events"

type Bus interface {
	Publish(ctx context.Context, roomID, domain, eventType string, payload any) error
	Close() error
}

type envelope struct {
	OriginID string          `json:"originId"`
	Type     string          `json:"type"`
	Domain   string          `json:"domain,omitempty"`
	RoomID   string          `json:"roomId"`
	Data     json.RawMessage `json:"data,omitempty"`
}

func marshalEnvelope(originID, roomID, domain, eventType string, payload any) ([]byte, error) {
	var data json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		data = b
	}
	ev := envelope{
		OriginID: originID,
		Type:     eventType,
		Domain:   domain,
		RoomID:   roomID,
		Data:     data,
	}
	return json.Marshal(ev)
}

func parseEnvelope(raw []byte) (envelope, error) {
	var ev envelope
	if err := json.Unmarshal(raw, &ev); err != nil {
		return envelope{}, fmt.Errorf("parse envelope: %w", err)
	}
	return ev, nil
}
