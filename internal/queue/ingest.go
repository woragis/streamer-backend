package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	StreamIngest = "streamer:jobs:ingest"
	StreamDLQ    = "streamer:jobs:ingest:dlq"
	GroupIngest  = "ingest-workers"
	MaxAttempts  = 5
)

const (
	JobChatMessage  = "chat.message"
	JobStreamEvent  = "stream.event"
)

var ErrDisabled = errors.New("ingest queue disabled")

type Job struct {
	ID      string
	JobType string
	RoomID  string
	Payload json.RawMessage
	Attempt int
}

type IngestQueue struct {
	rdb *goredis.Client
}

func New(rdb *goredis.Client) *IngestQueue {
	if rdb == nil {
		return nil
	}
	return &IngestQueue{rdb: rdb}
}

func (q *IngestQueue) Enabled() bool {
	return q != nil && q.rdb != nil
}

func (q *IngestQueue) EnsureGroup(ctx context.Context) error {
	if !q.Enabled() {
		return ErrDisabled
	}
	err := q.rdb.XGroupCreateMkStream(ctx, StreamIngest, GroupIngest, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *IngestQueue) Enqueue(ctx context.Context, jobType, roomID string, payload any) (string, error) {
	if !q.Enabled() {
		return "", ErrDisabled
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return q.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: StreamIngest,
		Values: map[string]interface{}{
			"jobType": jobType,
			"roomId":  roomID,
			"payload": string(raw),
			"attempt": 0,
		},
	}).Result()
}

func (q *IngestQueue) PendingCount(ctx context.Context) (int64, error) {
	if !q.Enabled() {
		return 0, nil
	}
	info, err := q.rdb.XPending(ctx, StreamIngest, GroupIngest).Result()
	if err != nil {
		return 0, err
	}
	return info.Count, nil
}

func (q *IngestQueue) RunConsumer(ctx context.Context, consumerID string, handle func(context.Context, Job) error) {
	if !q.Enabled() {
		return
	}

	go func() {
		backoff := time.Second
		for {
			if ctx.Err() != nil {
				return
			}
			if err := q.consumeOnce(ctx, consumerID, handle); err != nil && ctx.Err() == nil {
				time.Sleep(backoff)
				if backoff < 10*time.Second {
					backoff *= 2
				}
				continue
			}
			backoff = time.Second
		}
	}()
}

func (q *IngestQueue) consumeOnce(ctx context.Context, consumerID string, handle func(context.Context, Job) error) error {
	streams, err := q.rdb.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    GroupIngest,
		Consumer: consumerID,
		Streams:  []string{StreamIngest, ">"},
		Count:    10,
		Block:    2 * time.Second,
	}).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil
		}
		return err
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			job, err := parseJob(msg)
			if err != nil {
				_ = q.moveToDLQ(ctx, msg, err.Error())
				_ = q.rdb.XAck(ctx, StreamIngest, GroupIngest, msg.ID).Err()
				continue
			}

			if err := handle(ctx, job); err != nil {
				job.Attempt++
				if job.Attempt >= MaxAttempts {
					_ = q.moveToDLQ(ctx, msg, err.Error())
				} else {
					_, _ = q.rdb.XAdd(ctx, &goredis.XAddArgs{
						Stream: StreamIngest,
						Values: map[string]interface{}{
							"jobType": job.JobType,
							"roomId":  job.RoomID,
							"payload": string(job.Payload),
							"attempt": job.Attempt,
							"lastErr": err.Error(),
						},
					}).Result()
				}
			}
			_ = q.rdb.XAck(ctx, StreamIngest, GroupIngest, msg.ID).Err()
		}
	}
	return nil
}

func (q *IngestQueue) moveToDLQ(ctx context.Context, msg goredis.XMessage, reason string) error {
	values := map[string]interface{}{"reason": reason}
	for k, v := range msg.Values {
		values[k] = v
	}
	_, err := q.rdb.XAdd(ctx, &goredis.XAddArgs{
		Stream: StreamDLQ,
		Values: values,
	}).Result()
	return err
}

func parseJob(msg goredis.XMessage) (Job, error) {
	jobType, _ := msg.Values["jobType"].(string)
	roomID, _ := msg.Values["roomId"].(string)
	payloadStr, _ := msg.Values["payload"].(string)
	if jobType == "" || roomID == "" || payloadStr == "" {
		return Job{}, fmt.Errorf("invalid job fields")
	}
	attempt := 0
	switch v := msg.Values["attempt"].(type) {
	case string:
		_, _ = fmt.Sscan(v, &attempt)
	case int64:
		attempt = int(v)
	}
	return Job{
		ID:      msg.ID,
		JobType: jobType,
		RoomID:  roomID,
		Payload: json.RawMessage(payloadStr),
		Attempt: attempt,
	}, nil
}
