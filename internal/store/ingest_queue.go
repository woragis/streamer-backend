package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/woragis/streamer-backend/internal/platform"
	"github.com/woragis/streamer-backend/internal/queue"
)

func (s *Store) EnqueueIngestMessage(ctx context.Context, roomID string, in platform.IngestMessageInput) (string, error) {
	if !s.QueueEnabled() {
		return "", queue.ErrDisabled
	}
	if in.Platform == "" || in.Username == "" || in.Content == "" {
		return "", fmt.Errorf("platform, username and content required")
	}
	return s.queue.Enqueue(ctx, queue.JobChatMessage, roomID, in)
}

func (s *Store) EnqueueIngestEvent(ctx context.Context, roomID string, in platform.IngestEventInput) (string, error) {
	if !s.QueueEnabled() {
		return "", queue.ErrDisabled
	}
	if in.Type == "" {
		return "", fmt.Errorf("type required")
	}
	return s.queue.Enqueue(ctx, queue.JobStreamEvent, roomID, in)
}

func (s *Store) ProcessIngestJob(ctx context.Context, job queue.Job) error {
	switch job.JobType {
	case queue.JobChatMessage:
		var in platform.IngestMessageInput
		if err := json.Unmarshal(job.Payload, &in); err != nil {
			return err
		}
		result, err := s.IngestMessage(ctx, job.RoomID, in)
		if err != nil {
			return err
		}
		if result.Duplicate {
			return nil
		}
		return nil
	case queue.JobStreamEvent:
		var in platform.IngestEventInput
		if err := json.Unmarshal(job.Payload, &in); err != nil {
			return err
		}
		_, err := s.IngestStreamEvent(ctx, job.RoomID, in)
		if errors.Is(err, ErrDuplicateIngest) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}
