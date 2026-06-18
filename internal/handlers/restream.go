package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/restream"
	"github.com/woragis/streamer-backend/internal/store"
)

type RestreamHandler struct {
	Store              *store.Store
	ObsServer          string
	RelaySourceBase    string
	InternalToken      string
}

type mediamtxAuthRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
	IP       string `json:"ip"`
	Action   string `json:"action"`
	Path     string `json:"path"`
	Protocol string `json:"protocol"`
	ID       string `json:"id"`
	Query    string `json:"query"`
}

func (h *RestreamHandler) ensureRoom(ctx context.Context, w http.ResponseWriter, roomID string) bool {
	ok, err := h.Store.RoomExists(ctx, roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return false
	}
	if !ok {
		WriteError(w, http.StatusNotFound, "room not found")
		return false
	}
	return true
}

func (h *RestreamHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	settings, err := h.Store.GetRestreamSettingsPublic(r.Context(), roomID, h.ObsServer)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	WriteJSON(w, http.StatusOK, settings)
}

func (h *RestreamHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	var in restream.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	settings, err := h.Store.UpdateRestreamSettings(r.Context(), roomID, in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	settings, err = h.Store.GetRestreamSettingsPublic(r.Context(), roomID, h.ObsServer)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	WriteJSON(w, http.StatusOK, settings)
}

func (h *RestreamHandler) RegenerateIngestKey(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	settings, err := h.Store.RegenerateRestreamIngestKey(r.Context(), roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	settings.ObsServer = strings.TrimRight(h.ObsServer, "/")
	WriteJSON(w, http.StatusOK, settings)
}

func (h *RestreamHandler) Auth(w http.ResponseWriter, r *http.Request) {
	var in mediamtxAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid body", http.StatusUnauthorized)
		return
	}
	if in.Action != "publish" {
		w.WriteHeader(http.StatusOK)
		return
	}
	key := parseQueryParam(in.Query, "key")
	if key == "" {
		key = strings.TrimSpace(in.Password)
	}
	ok, err := h.Store.ValidateRestreamPublish(r.Context(), in.Path, key)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RestreamHandler) InternalRelay(w http.ResponseWriter, r *http.Request) {
	if !h.checkInternalToken(r) {
		WriteError(w, http.StatusUnauthorized, "invalid internal token")
		return
	}
	roomID := chi.URLParam(r, "roomId")
	resolved, err := h.Store.GetRestreamSettingsResolved(r.Context(), roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !resolved.Ready() {
		WriteError(w, http.StatusConflict, "restream not configured for room")
		return
	}
	WriteJSON(w, http.StatusOK, resolved.RelayConfig(h.RelaySourceBase))
}

func (h *RestreamHandler) checkInternalToken(r *http.Request) bool {
	if h.InternalToken == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get("X-Restream-Internal-Token"))
	return got != "" && got == h.InternalToken
}

func parseQueryParam(query, key string) string {
	for part := range strings.SplitSeq(query, "&") {
		k, v, ok := strings.Cut(part, "=")
		if ok && k == key {
			return v
		}
	}
	return ""
}
