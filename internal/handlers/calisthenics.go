package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/calisthenics"
	"github.com/woragis/streamer-backend/internal/store"
)

type CalisthenicsHandler struct {
	Store *store.Store
}

func (h *CalisthenicsHandler) ensureRoom(w http.ResponseWriter, r *http.Request) (string, bool) {
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

func (h *CalisthenicsHandler) GetState(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	state, err := h.Store.GetCalisthenicsState(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	SetETag(w, state.Revision)
	WriteJSON(w, http.StatusOK, state)
}

func (h *CalisthenicsHandler) PutState(w http.ResponseWriter, r *http.Request) {
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
	state, err := h.Store.PutCalisthenicsState(r.Context(), roomID, data, expected)
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

func (h *CalisthenicsHandler) ListWorkouts(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListWorkouts(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.CreateWorkoutInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	wout, err := h.Store.CreateWorkout(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, wout)
}

func (h *CalisthenicsHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	wout, err := h.Store.GetWorkoutByID(r.Context(), roomID, chi.URLParam(r, "workoutId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, wout)
}

func (h *CalisthenicsHandler) UpdateWorkout(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.UpdateWorkoutInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	wout, err := h.Store.UpdateWorkout(r.Context(), roomID, chi.URLParam(r, "workoutId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, wout)
}

func (h *CalisthenicsHandler) DeleteWorkout(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteWorkout(r.Context(), roomID, chi.URLParam(r, "workoutId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalisthenicsHandler) ListExercises(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListExercises(r.Context(), roomID, chi.URLParam(r, "workoutId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.CreateExerciseInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ex, err := h.Store.CreateExercise(r.Context(), roomID, chi.URLParam(r, "workoutId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, ex)
}

func (h *CalisthenicsHandler) UpdateExercise(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.UpdateExerciseInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	ex, err := h.Store.UpdateExercise(r.Context(), roomID, chi.URLParam(r, "exerciseId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, ex)
}

func (h *CalisthenicsHandler) DeleteExercise(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteExercise(r.Context(), roomID, chi.URLParam(r, "exerciseId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalisthenicsHandler) ActivateExercise(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	ex, err := h.Store.ActivateExercise(r.Context(), roomID, chi.URLParam(r, "exerciseId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, ex)
}

func (h *CalisthenicsHandler) ListSets(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListSets(r.Context(), roomID, chi.URLParam(r, "exerciseId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) CompleteSet(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	set, err := h.Store.CompleteSet(r.Context(), roomID, chi.URLParam(r, "setId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, set)
}

func (h *CalisthenicsHandler) IncrementRep(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	set, err := h.Store.IncrementRep(r.Context(), roomID, chi.URLParam(r, "setId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, set)
}

func (h *CalisthenicsHandler) SkipSet(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	set, err := h.Store.SkipSet(r.Context(), roomID, chi.URLParam(r, "setId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, set)
}

func (h *CalisthenicsHandler) GetTimers(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	t, err := h.Store.GetCalTimers(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(t)
}

func (h *CalisthenicsHandler) PutTimer(w http.ResponseWriter, r *http.Request) {
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
	updated, err := h.Store.UpdateCalTimer(r.Context(), roomID, timerID, action.Action, body)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(updated)
}

/* ─── Skills (Phase D) ─── */

func (h *CalisthenicsHandler) ListMovementCategories(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListMovementCategories(r.Context(), roomID)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	items, err := h.Store.ListMovements(r.Context(), roomID, r.URL.Query().Get("level"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) CreateMovement(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.CreateMovementInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	m, err := h.Store.CreateMovement(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, m)
}

func (h *CalisthenicsHandler) GetMovement(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	m, err := h.Store.GetMovement(r.Context(), roomID, chi.URLParam(r, "movementId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, m)
}

func (h *CalisthenicsHandler) UpdateMovement(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.UpdateMovementInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	m, err := h.Store.UpdateMovement(r.Context(), roomID, chi.URLParam(r, "movementId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, m)
}

func (h *CalisthenicsHandler) DeleteMovement(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteMovement(r.Context(), roomID, chi.URLParam(r, "movementId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalisthenicsHandler) GetProficiency(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	p, err := h.Store.GetProficiency(r.Context(), roomID, chi.URLParam(r, "movementId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *CalisthenicsHandler) UpdateProficiency(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.UpdateProficiencyInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	p, err := h.Store.UpdateProficiency(r.Context(), roomID, chi.URLParam(r, "movementId"), in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, p)
}

func (h *CalisthenicsHandler) ListAcquisitions(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	items, err := h.Store.ListAcquisitions(r.Context(), roomID, q.Get("month"), q.Get("liveSessionId"), q.Get("movementId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

func (h *CalisthenicsHandler) CreateAcquisition(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.CreateAcquisitionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	a, err := h.Store.CreateAcquisition(r.Context(), roomID, in)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, a)
}

func (h *CalisthenicsHandler) GetAcquisition(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	a, err := h.Store.GetAcquisition(r.Context(), roomID, chi.URLParam(r, "acquisitionId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, a)
}

func (h *CalisthenicsHandler) DeleteAcquisition(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.DeleteAcquisition(r.Context(), roomID, chi.URLParam(r, "acquisitionId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalisthenicsHandler) AcknowledgeAcquisition(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	if err := h.Store.AcknowledgeAcquisition(r.Context(), roomID, chi.URLParam(r, "acquisitionId")); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CalisthenicsHandler) GetMovementHistory(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	history, err := h.Store.GetMovementHistory(r.Context(), roomID, chi.URLParam(r, "movementId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, history)
}

func (h *CalisthenicsHandler) GetSkillStats(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	stats, err := h.Store.GetSkillStats(r.Context(), roomID, q.Get("month"), q.Get("liveSessionId"))
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, stats)
}

func (h *CalisthenicsHandler) CreatePracticeLog(w http.ResponseWriter, r *http.Request) {
	roomID, ok := h.ensureRoom(w, r)
	if !ok {
		return
	}
	var in calisthenics.CreatePracticeLogInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.Store.CreatePracticeLog(r.Context(), roomID, in); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func writeStoreErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrRevisionConflict):
		WriteError(w, http.StatusConflict, "revision conflict")
	case errors.Is(err, store.ErrCalInvalidSet):
		WriteError(w, http.StatusBadRequest, err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "database error")
	}
}
