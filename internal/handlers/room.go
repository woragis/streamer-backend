package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/store"
)

type RoomHandler struct {
	Store *store.Store
}

func (h *RoomHandler) ensureRoom(ctx context.Context, w http.ResponseWriter, roomID string) bool {
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

func (h *RoomHandler) getDoc(w http.ResponseWriter, r *http.Request, key string) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}

	doc, err := h.Store.GetDocument(r.Context(), roomID, key)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	SetETag(w, doc.Revision)
	out := store.MergeRevision(doc.Data, doc.Revision)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (h *RoomHandler) putDoc(w http.ResponseWriter, r *http.Request, key string) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}

	body, err := ReadRawBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	expected := store.ParseExpectedRevision(ParseIfMatch(r), body)
	data := store.StripRevisionField(body)

	doc, err := h.Store.PutDocument(r.Context(), roomID, key, data, expected)
	if errors.Is(err, store.ErrRevisionConflict) {
		WriteError(w, http.StatusConflict, "revision conflict")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeDoc(w, doc)
}

func writeDoc(w http.ResponseWriter, doc store.Document) {
	SetETag(w, doc.Revision)
	out := store.MergeRevision(doc.Data, doc.Revision)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (h *RoomHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	h.getDoc(w, r, store.DocSession)
}

func (h *RoomHandler) PutSession(w http.ResponseWriter, r *http.Request) {
	h.putDoc(w, r, store.DocSession)
}

func (h *RoomHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	h.getDoc(w, r, store.DocBranding)
}

func (h *RoomHandler) PutBranding(w http.ResponseWriter, r *http.Request) {
	h.putDoc(w, r, store.DocBranding)
}

func (h *RoomHandler) GetStreamTimer(w http.ResponseWriter, r *http.Request) {
	h.getDoc(w, r, store.DocStreamTimer)
}

func (h *RoomHandler) PutStreamTimer(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if !h.ensureRoom(r.Context(), w, roomID) {
		return
	}

	body, err := ReadRawBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	var action struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(body, &action)

	if action.Action != "" {
		h.putStreamTimerAction(w, r, roomID, action.Action)
		return
	}

	expected := store.ParseExpectedRevision(ParseIfMatch(r), body)
	data := store.StripRevisionField(body)

	doc, err := h.Store.PutDocument(r.Context(), roomID, store.DocStreamTimer, data, expected)
	if errors.Is(err, store.ErrRevisionConflict) {
		WriteError(w, http.StatusConflict, "revision conflict")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeDoc(w, doc)
}

func (h *RoomHandler) putStreamTimerAction(w http.ResponseWriter, r *http.Request, roomID, action string) {
	doc, err := h.Store.GetDocument(r.Context(), roomID, store.DocStreamTimer)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}

	var timer map[string]any
	if err := json.Unmarshal(doc.Data, &timer); err != nil {
		WriteError(w, http.StatusInternalServerError, "invalid timer data")
		return
	}

	nowMs := time.Now().UnixMilli()
	if err := applyTimerAction(timer, action, nowMs); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := json.Marshal(timer)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "encode error")
		return
	}

	expected := &doc.Revision
	newDoc, err := h.Store.PutDocument(r.Context(), roomID, store.DocStreamTimer, updated, expected)
	if errors.Is(err, store.ErrRevisionConflict) {
		WriteError(w, http.StatusConflict, "revision conflict")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeDoc(w, newDoc)
}

func applyTimerAction(timer map[string]any, action string, nowMs int64) error {
	switch action {
	case "start":
		if running, _ := timer["running"].(bool); running {
			return nil
		}
		timer["running"] = true
		timer["startedAt"] = nowMs
		if mode, _ := timer["mode"].(string); mode == "countdown" {
			dur := toInt(timer["durationSeconds"])
			acc := toInt(timer["accumulatedSeconds"])
			remaining := dur - acc
			if remaining < 0 {
				remaining = 0
			}
			timer["endsAt"] = nowMs + int64(remaining)*1000
		}
	case "pause":
		if running, _ := timer["running"].(bool); !running {
			return nil
		}
		startedAt := toInt64(timer["startedAt"])
		if startedAt > 0 {
			elapsed := (nowMs - startedAt) / 1000
			timer["accumulatedSeconds"] = toInt(timer["accumulatedSeconds"]) + int(elapsed)
		}
		timer["running"] = false
		timer["startedAt"] = nil
		timer["endsAt"] = nil
	case "reset":
		timer["running"] = false
		timer["startedAt"] = nil
		timer["endsAt"] = nil
		timer["accumulatedSeconds"] = 0
	default:
		return errors.New("unknown action; use start, pause, or reset")
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func (h *RoomHandler) GetLeetCodeState(w http.ResponseWriter, r *http.Request) {
	h.getDoc(w, r, store.DocLeetCode)
}

func (h *RoomHandler) PutLeetCodeState(w http.ResponseWriter, r *http.Request) {
	h.putDoc(w, r, store.DocLeetCode)
}

func (h *RoomHandler) GetCalisthenicsState(w http.ResponseWriter, r *http.Request) {
	h.getDoc(w, r, store.DocCalisthenics)
}

func (h *RoomHandler) PutCalisthenicsState(w http.ResponseWriter, r *http.Request) {
	h.putDoc(w, r, store.DocCalisthenics)
}
