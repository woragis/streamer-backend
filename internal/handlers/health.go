package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/woragis/streamer-backend/internal/db"
	appredis "github.com/woragis/streamer-backend/internal/redis"
	"github.com/woragis/streamer-backend/internal/queue"
)

type HealthHandler struct {
	DB         *db.DB
	Redis      *appredis.Client
	Queue      *queue.IngestQueue
	InstanceID string
	IngestMode string
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	dbStatus := "down"
	if h.DB != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.DB.PingContext(ctx); err != nil {
			status = "degraded"
		} else {
			dbStatus = "ok"
		}
	}

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
		"database":   dbStatus,
		"driver":     "postgres",
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
