package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/woragis/streamer-backend/internal/config"
	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/handlers"
	appmw "github.com/woragis/streamer-backend/internal/middleware"
	"github.com/woragis/streamer-backend/internal/store"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	st := store.New(database)
	if err := st.Seed(context.Background()); err != nil {
		log.Fatalf("seed: %v", err)
	}

	roomHandler := &handlers.RoomHandler{Store: st}

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

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/rooms/{roomId}", func(r chi.Router) {
			r.Get("/session", roomHandler.GetSession)
			r.Put("/session", roomHandler.PutSession)

			r.Get("/branding", roomHandler.GetBranding)
			r.Put("/branding", roomHandler.PutBranding)

			r.Get("/timers/stream", roomHandler.GetStreamTimer)
			r.Put("/timers/stream", roomHandler.PutStreamTimer)

			r.Get("/leetcode/state", roomHandler.GetLeetCodeState)
			r.Put("/leetcode/state", roomHandler.PutLeetCodeState)

			r.Get("/calisthenics/state", roomHandler.GetCalisthenicsState)
			r.Put("/calisthenics/state", roomHandler.PutCalisthenicsState)
		})
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("state-api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
