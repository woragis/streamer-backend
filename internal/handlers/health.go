package handlers

import (
	"context"
	"net/http"
	"time"

	appredis "github.com/woragis/streamer-backend/internal/redis"
	"github.com/woragis/streamer-backend/internal/queue"
)

type HealthHandler struct {
	Redis      *appredis.Client
	Queue      *queue.IngestQueue
	InstanceID string
	IngestMode string
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	redisStatus := "disabled"
	if h.Redis != nil {
		redisStatus = h.Redis.Status()
		if h.Redis.Enabled() {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := h.Redis.Ping(ctx); err != nil {
				redisStatus = "down"
				status = "degraded"
			} else {
				redisStatus = "ok"
			}
		}
	}

	out := map[string]any{
		"status":     status,
		"redis":      redisStatus,
		"instanceId": h.InstanceID,
		"ingestMode": h.IngestMode,
	}
	if h.Queue != nil && h.Queue.Enabled() {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if pending, err := h.Queue.PendingCount(ctx); err == nil {
			out["ingestQueuePending"] = pending
		}
	}

	WriteJSON(w, http.StatusOK, out)
}
