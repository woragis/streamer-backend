package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/woragis/streamer-backend/internal/config"
	appmw "github.com/woragis/streamer-backend/internal/middleware"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appmw.CORS(cfg.CORSOrigins))
	r.Use(appmw.BearerAuth(cfg.StateAPIToken))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("state-api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
