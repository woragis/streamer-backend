package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/platform"
	"github.com/woragis/streamer-backend/internal/store"
)

type PlatformSettingsHandler struct {
	Store *store.Store
}

func (h *PlatformSettingsHandler) ensureRoom(ctx context.Context, w http.ResponseWriter, roomID string) bool {
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

func (h *PlatformSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}

	settings, err := h.Store.GetPlatformSettings(r.Context(), roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	WriteJSON(w, http.StatusOK, settings)
}

func (h *PlatformSettingsHandler) Put(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}

	var in platform.UpdatePlatformSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	settings, err := h.Store.UpdatePlatformSettings(r.Context(), roomID, in)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, settings)
}
