package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/platform"
	"github.com/woragis/streamer-backend/internal/store"
)

type PlatformHandler struct {
	Store      *store.Store
	IngestMode string
}

func (h *PlatformHandler) ensureRoom(ctx context.Context, w http.ResponseWriter, roomID string) bool {
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

func (h *PlatformHandler) IngestMessage(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	var in platform.IngestMessageInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if h.IngestMode == "queue" {
		if !h.Store.QueueEnabled() {
			WriteError(w, http.StatusServiceUnavailable, "ingest queue unavailable")
			return
		}
		jobID, err := h.Store.EnqueueIngestMessage(r.Context(), roomID, in)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteJSON(w, http.StatusAccepted, platform.IngestResult{Queued: true, JobID: jobID})
		return
	}

	result, err := h.Store.IngestMessage(r.Context(), roomID, in)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if result.Duplicate {
		WriteJSON(w, http.StatusOK, result)
		return
	}
	WriteJSON(w, http.StatusCreated, result)
}

func (h *PlatformHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	var in platform.IngestEventInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if h.IngestMode == "queue" {
		if !h.Store.QueueEnabled() {
			WriteError(w, http.StatusServiceUnavailable, "ingest queue unavailable")
			return
		}
		jobID, err := h.Store.EnqueueIngestEvent(r.Context(), roomID, in)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		WriteJSON(w, http.StatusAccepted, map[string]any{"queued": true, "jobId": jobID})
		return
	}

	ev, err := h.Store.IngestStreamEvent(r.Context(), roomID, in)
	if errors.Is(err, store.ErrDuplicateIngest) {
		WriteJSON(w, http.StatusOK, map[string]bool{"duplicate": true})
		return
	}
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, ev)
}

func (h *PlatformHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	includeDeleted := q.Get("includeDeleted") == "true"
	msgs, err := h.Store.ListMessages(r.Context(), roomID, limit, includeDeleted)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	if msgs == nil {
		msgs = []platform.Message{}
	}
	WriteJSON(w, http.StatusOK, msgs)
}

func (h *PlatformHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	messageID := chi.URLParam(r, "messageId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	if err := h.Store.DeleteMessage(r.Context(), roomID, messageID); errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "message not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlatformHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.Store.ListStreamEvents(r.Context(), roomID, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	if events == nil {
		events = []platform.StreamEvent{}
	}
	WriteJSON(w, http.StatusOK, events)
}

func (h *PlatformHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	rules, err := h.Store.ListRules(r.Context(), roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	if rules == nil {
		rules = []platform.BotRule{}
	}
	WriteJSON(w, http.StatusOK, rules)
}

func (h *PlatformHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	var in platform.CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	rule, err := h.Store.CreateRule(r.Context(), roomID, in)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteJSON(w, http.StatusCreated, rule)
}

func (h *PlatformHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	ruleID := chi.URLParam(r, "ruleId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	var in platform.UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	rule, err := h.Store.UpdateRule(r.Context(), roomID, ruleID, in)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "rule not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	WriteJSON(w, http.StatusOK, rule)
}

func (h *PlatformHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	ruleID := chi.URLParam(r, "ruleId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	if err := h.Store.DeleteRule(r.Context(), roomID, ruleID); errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "rule not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PlatformHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}
	dash, err := h.Store.GetDashboard(r.Context(), roomID, r.URL.Query().Get("month"))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	WriteJSON(w, http.StatusOK, dash)
}
