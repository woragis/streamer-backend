package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/leetcode"
	"github.com/woragis/streamer-backend/internal/store"
)

type LeetCodeHandler struct {
	Store *store.Store
}

func (h *LeetCodeHandler) ensureRoom(w http.ResponseWriter, r *http.Request) (string, bool) {
	roomID := chi.URLParam(r, "roomId")
	ok, err := h.Store.RoomExists(r.Context(), roomID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "database error")
		return "", false
	}
	if !ok {
		WriteError(w, http.StatusNotFound, "room not found")
		return "", false
	}
	return roomID, true
}

func (h *LeetCodeHandler) GetState(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	state, err := h.Store.GetLeetCodeState(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	SetETag(w, state.Revision)
	WriteJSON(w, http.StatusOK, state)
}

func (h *LeetCodeHandler) PutState(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	body, err := ReadRawBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	expected := store.ParseExpectedRevision(ParseIfMatch(r), body)
	data := store.StripRevisionField(body)
	state, err := h.Store.PutLeetCodeState(r.Context(), roomID, data, expected)
	if errors.Is(err, store.ErrRevisionConflict) {
		WriteError(w, http.StatusConflict, "revision conflict")
		return
	}
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	SetETag(w, state.Revision)
	WriteJSON(w, http.StatusOK, state)
}

/* ─── Live Sessions ─── */

func (h *LeetCodeHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListLiveSessions(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *LeetCodeHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in leetcode.CreateLiveSessionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	sess, err := h.Store.CreateLiveSession(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, sess)
}

func (h *LeetCodeHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	sess, err := h.Store.GetLiveSession(r.Context(), roomID, chi.URLParam(r, "sessionId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, sess)
}

func (h *LeetCodeHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in leetcode.UpdateLiveSessionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	sess, err := h.Store.UpdateLiveSession(r.Context(), roomID, chi.URLParam(r, "sessionId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, sess)
}

/* ─── Plan ─── */

func (h *LeetCodeHandler) ListPlan(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListPlanItems(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *LeetCodeHandler) CreatePlanItem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in leetcode.CreatePlanItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	item, err := h.Store.CreatePlanItem(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, item)
}

func (h *LeetCodeHandler) UpdatePlanItem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in leetcode.UpdatePlanItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	item, err := h.Store.UpdatePlanItem(r.Context(), roomID, chi.URLParam(r, "itemId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, item)
}

func (h *LeetCodeHandler) DeletePlanItem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeletePlanItem(r.Context(), roomID, chi.URLParam(r, "itemId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LeetCodeHandler) TogglePlanItem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	item, err := h.Store.TogglePlanItem(r.Context(), roomID, chi.URLParam(r, "itemId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, item)
}

/* ─── Problems ─── */

func (h *LeetCodeHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListProblems(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *LeetCodeHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in leetcode.CreateProblemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	p, err := h.Store.CreateProblem(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, p)
}

func (h *LeetCodeHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	p, err := h.Store.GetProblem(r.Context(), roomID, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *LeetCodeHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	var in leetcode.UpdateProblemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	p, err := h.Store.UpdateProblem(r.Context(), roomID, id, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *LeetCodeHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	if err := h.Store.DeleteProblem(r.Context(), roomID, id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *LeetCodeHandler) ActivateProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	p, err := h.Store.ActivateProblem(r.Context(), roomID, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *LeetCodeHandler) SolveProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	p, err := h.Store.SolveProblem(r.Context(), roomID, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *LeetCodeHandler) SkipProblem(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "problemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid problem id")
		return
	}
	p, err := h.Store.SkipProblem(r.Context(), roomID, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

/* ─── Stats & Timers ─── */

func (h *LeetCodeHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	month := r.URL.Query().Get("month")
	liveSessionID := r.URL.Query().Get("liveSessionId")
	stats, err := h.Store.GetLeetCodeStats(r.Context(), roomID, month, liveSessionID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, stats)
}

func (h *LeetCodeHandler) GetStreak(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	stats, err := h.Store.GetLeetCodeStreak(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, stats)
}

func (h *LeetCodeHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListProblemAttempts(r.Context(), roomID, r.URL.Query().Get("liveSessionId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *LeetCodeHandler) GetTimers(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	t, err := h.Store.GetLCTimers(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(t)
}

func (h *LeetCodeHandler) PutTimer(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	timerID := chi.URLParam(r, "timerId")
	body, err := ReadRawBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	var action struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(body, &action)
	updated, err := h.Store.UpdateLCTimer(r.Context(), roomID, timerID, action.Action, body)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(updated)
}
