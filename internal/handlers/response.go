package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

func ReadRawBody(r *http.Request) (json.RawMessage, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, errors.New("invalid json")
	}
	return json.RawMessage(b), nil
}

func ParseIfMatch(r *http.Request) string {
	return r.Header.Get("If-Match")
}

func SetETag(w http.ResponseWriter, revision int64) {
	w.Header().Set("ETag", strconv.FormatInt(revision, 10))
}
