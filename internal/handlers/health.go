package handlers

import (
	"context"
	"net/http"
	"time"

	appredis "github.com/woragis/streamer-backend/internal/redis"
)

type HealthHandler struct {
	Redis      *appredis.Client
	InstanceID string
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

	WriteJSON(w, http.StatusOK, map[string]string{
		"status":     status,
		"redis":      redisStatus,
		"instanceId": h.InstanceID,
	})
}
