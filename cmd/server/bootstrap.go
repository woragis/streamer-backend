package main

import (
	"encoding/json"
	"net/http"
	"sync"

	appmw "github.com/woragis/streamer-backend/internal/middleware"
)

// earlyListener binds HTTP before DB/seed init so Railway never sees connection refused.
type earlyListener struct {
	cors    []string
	mu      sync.RWMutex
	handler http.Handler
	initErr error
	ready   bool
}

func newEarlyListener(cors []string) *earlyListener {
	return &earlyListener{cors: cors}
}

func (e *earlyListener) setReady(h http.Handler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handler = h
	e.ready = true
	e.initErr = nil
}

func (e *earlyListener) setError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.initErr = err
}

func (e *earlyListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	appmw.CORS(e.cors)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.RLock()
		ready := e.ready
		handler := e.handler
		initErr := e.initErr
		e.mu.RUnlock()

		if ready && handler != nil {
			handler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path != "/health" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			msg := "starting"
			if initErr != nil {
				msg = initErr.Error()
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"error": msg, "status": "starting"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if initErr != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  initErr.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "starting"})
	})).ServeHTTP(w, r)
}
