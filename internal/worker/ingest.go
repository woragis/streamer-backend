package worker

import (
	"context"
	"log"

	"github.com/woragis/streamer-backend/internal/queue"
	"github.com/woragis/streamer-backend/internal/store"
)

func StartIngestConsumer(ctx context.Context, q *queue.IngestQueue, st *store.Store, consumerID string) {
	if q == nil || !q.Enabled() {
		return
	}
	if err := q.EnsureGroup(ctx); err != nil {
		log.Printf("ingest queue: ensure group: %v", err)
		return
	}
	log.Printf("ingest consumer started as %s", consumerID)
	q.RunConsumer(ctx, consumerID, func(ctx context.Context, job queue.Job) error {
		if err := st.ProcessIngestJob(ctx, job); err != nil {
			log.Printf("ingest job %s (%s): %v", job.ID, job.JobType, err)
			return err
		}
		return nil
	})
}
