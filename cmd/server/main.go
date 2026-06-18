package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/woragis/streamer-backend/internal/bus"
	"github.com/woragis/streamer-backend/internal/config"
	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/handlers"
	appmw "github.com/woragis/streamer-backend/internal/middleware"
	appredis "github.com/woragis/streamer-backend/internal/redis"
	"github.com/woragis/streamer-backend/internal/dedup"
	"github.com/woragis/streamer-backend/internal/queue"
	"github.com/woragis/streamer-backend/internal/store"
	"github.com/woragis/streamer-backend/internal/worker"
	"github.com/woragis/streamer-backend/internal/ws"
)

func main() {
	cfg := config.Load()
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	log.Printf("state-api boot (port=%s cors=%v)", cfg.Port, cfg.CORSOrigins)

	gate := newEarlyListener(cfg.CORSOrigins)
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      gate,
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

	app, composite, err := buildApp(ctx, cfg)
	if err != nil {
		log.Printf("init failed: %v", err)
		gate.setError(err)
		<-ctx.Done()
		shutdownServer(srv, composite)
		return
	}
	gate.setReady(app)
	log.Printf("state-api ready")

	<-ctx.Done()
	shutdownServer(srv, composite)
}

func shutdownServer(srv *http.Server, composite *bus.CompositeBus) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if composite != nil {
		_ = composite.Close()
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func buildApp(ctx context.Context, cfg config.Config) (http.Handler, *bus.CompositeBus, error) {
	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("database: %w", err)
	}

	redisClient, err := appredis.Connect(cfg.RedisURL)
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("redis: %w", err)
	}
	if cfg.RedisURL != "" {
		log.Printf("redis: %s (instance %s, ingest=%s)", redisClient.Status(), cfg.InstanceID, cfg.IngestMode)
	}

	st := store.New(database)
	if err := st.Seed(ctx); err != nil {
		log.Printf("seed warning: %v (continuing startup)", err)
	}

	hub := ws.NewHub(cfg.CORSOrigins)
	localBus := bus.NewLocal(hub)
	var eventBus bus.Bus = localBus
	var composite *bus.CompositeBus

	if redisClient.Enabled() && redisClient.Status() == "ok" {
		redisBus := bus.NewRedis(redisClient.Raw(), cfg.InstanceID, localBus)
		composite = bus.NewComposite(localBus, redisBus)
		composite.Start(ctx)
		eventBus = composite
	}

	st.SetBus(eventBus)

	ingestQueue := queue.New(redisClient.Raw())
	dedupStore := dedup.New(redisClient.Raw())
	if ingestQueue != nil && ingestQueue.Enabled() {
		st.SetQueue(ingestQueue)
		st.SetDedup(dedupStore)
		if cfg.ConsumerEnabled {
			worker.StartIngestConsumer(ctx, ingestQueue, st, cfg.InstanceID+"-ingest")
		}
	}

	roomHandler := &handlers.RoomHandler{Store: st}
	calHandler := &handlers.CalisthenicsHandler{Store: st}
	lcHandler := &handlers.LeetCodeHandler{Store: st}
	platformHandler := &handlers.PlatformHandler{Store: st, IngestMode: cfg.IngestMode}
	platformSettingsHandler := &handlers.PlatformSettingsHandler{Store: st}
	wsHandler := &handlers.WSHandler{Hub: hub, Token: cfg.StateAPIToken}
	kickWebhookHandler, err := handlers.NewKickWebhookHandler(st)
	if err != nil {
		_ = database.Close()
		_ = redisClient.Close()
		if composite != nil {
			_ = composite.Close()
		}
		return nil, nil, fmt.Errorf("kick webhook: %w", err)
	}
	healthHandler := &handlers.HealthHandler{
		DB:         database,
		Redis:      redisClient,
		Queue:      ingestQueue,
		InstanceID: cfg.InstanceID,
		IngestMode: cfg.IngestMode,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(appmw.CORS(cfg.CORSOrigins))
	r.Use(appmw.BearerAuth(cfg.StateAPIToken))

	r.Get("/health", healthHandler.Check)
	r.Get("/api/v1/rooms/{roomId}/subscribe", wsHandler.Subscribe)
	r.Post("/api/v1/webhooks/kick", kickWebhookHandler.Receive)

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

			r.Get("/leetcode/sessions", lcHandler.ListSessions)
			r.Post("/leetcode/sessions", lcHandler.CreateSession)
			r.Get("/leetcode/sessions/{sessionId}", lcHandler.GetSession)
			r.Patch("/leetcode/sessions/{sessionId}", lcHandler.UpdateSession)

			r.Get("/leetcode/plan", lcHandler.ListPlan)
			r.Post("/leetcode/plan", lcHandler.CreatePlanItem)
			r.Patch("/leetcode/plan/{itemId}", lcHandler.UpdatePlanItem)
			r.Delete("/leetcode/plan/{itemId}", lcHandler.DeletePlanItem)
			r.Post("/leetcode/plan/{itemId}/toggle", lcHandler.TogglePlanItem)

			r.Get("/leetcode/problems", lcHandler.ListProblems)
			r.Post("/leetcode/problems", lcHandler.CreateProblem)
			r.Get("/leetcode/problems/{problemId}", lcHandler.GetProblem)
			r.Patch("/leetcode/problems/{problemId}", lcHandler.UpdateProblem)
			r.Delete("/leetcode/problems/{problemId}", lcHandler.DeleteProblem)
			r.Post("/leetcode/problems/{problemId}/activate", lcHandler.ActivateProblem)
			r.Post("/leetcode/problems/{problemId}/solve", lcHandler.SolveProblem)
			r.Post("/leetcode/problems/{problemId}/skip", lcHandler.SkipProblem)

			r.Get("/leetcode/stats/streak", lcHandler.GetStreak)
			r.Get("/leetcode/stats", lcHandler.GetStats)
			r.Get("/leetcode/attempts", lcHandler.ListAttempts)

			r.Get("/leetcode/timers", lcHandler.GetTimers)
			r.Get("/leetcode/timers/{timerId}", lcHandler.GetTimers)
			r.Put("/leetcode/timers/{timerId}", lcHandler.PutTimer)

			r.Get("/calisthenics/state", roomHandler.GetCalisthenicsState)
			r.Put("/calisthenics/state", roomHandler.PutCalisthenicsState)

			r.Get("/calisthenics/workouts", calHandler.ListWorkouts)
			r.Post("/calisthenics/workouts", calHandler.CreateWorkout)
			r.Get("/calisthenics/workouts/{workoutId}", calHandler.GetWorkout)
			r.Patch("/calisthenics/workouts/{workoutId}", calHandler.UpdateWorkout)
			r.Delete("/calisthenics/workouts/{workoutId}", calHandler.DeleteWorkout)

			r.Get("/calisthenics/workouts/{workoutId}/exercises", calHandler.ListExercises)
			r.Post("/calisthenics/workouts/{workoutId}/exercises", calHandler.CreateExercise)
			r.Patch("/calisthenics/exercises/{exerciseId}", calHandler.UpdateExercise)
			r.Delete("/calisthenics/exercises/{exerciseId}", calHandler.DeleteExercise)
			r.Post("/calisthenics/exercises/{exerciseId}/activate", calHandler.ActivateExercise)

			r.Get("/calisthenics/exercises/{exerciseId}/sets", calHandler.ListSets)
			r.Post("/calisthenics/sets/{setId}/complete", calHandler.CompleteSet)
			r.Post("/calisthenics/sets/{setId}/increment-rep", calHandler.IncrementRep)
			r.Post("/calisthenics/sets/{setId}/skip", calHandler.SkipSet)

			r.Get("/calisthenics/timers", calHandler.GetTimers)
			r.Get("/calisthenics/timers/{timerId}", calHandler.GetTimers)
			r.Put("/calisthenics/timers/{timerId}", calHandler.PutTimer)

			r.Get("/calisthenics/movements/categories", calHandler.ListMovementCategories)
			r.Get("/calisthenics/movements", calHandler.ListMovements)
			r.Post("/calisthenics/movements", calHandler.CreateMovement)
			r.Get("/calisthenics/movements/{movementId}", calHandler.GetMovement)
			r.Patch("/calisthenics/movements/{movementId}", calHandler.UpdateMovement)
			r.Delete("/calisthenics/movements/{movementId}", calHandler.DeleteMovement)
			r.Get("/calisthenics/movements/{movementId}/proficiency", calHandler.GetProficiency)
			r.Put("/calisthenics/movements/{movementId}/proficiency", calHandler.UpdateProficiency)
			r.Get("/calisthenics/movements/{movementId}/history", calHandler.GetMovementHistory)

			r.Get("/calisthenics/acquisitions", calHandler.ListAcquisitions)
			r.Post("/calisthenics/acquisitions", calHandler.CreateAcquisition)
			r.Get("/calisthenics/acquisitions/{acquisitionId}", calHandler.GetAcquisition)
			r.Delete("/calisthenics/acquisitions/{acquisitionId}", calHandler.DeleteAcquisition)
			r.Post("/calisthenics/acquisitions/{acquisitionId}/ack", calHandler.AcknowledgeAcquisition)

			r.Get("/calisthenics/stats", calHandler.GetSkillStats)
			r.Post("/calisthenics/practice-logs", calHandler.CreatePracticeLog)

			r.Post("/chat/ingest", platformHandler.IngestMessage)
			r.Get("/chat/messages", platformHandler.ListMessages)
			r.Delete("/chat/messages/{messageId}", platformHandler.DeleteMessage)

			r.Post("/events/ingest", platformHandler.IngestEvent)
			r.Get("/events", platformHandler.ListEvents)

			r.Get("/rules", platformHandler.ListRules)
			r.Post("/rules", platformHandler.CreateRule)
			r.Patch("/rules/{ruleId}", platformHandler.UpdateRule)
			r.Delete("/rules/{ruleId}", platformHandler.DeleteRule)

			r.Get("/dashboard", platformHandler.GetDashboard)

			r.Get("/platform-settings", platformSettingsHandler.Get)
			r.Put("/platform-settings", platformSettingsHandler.Put)
		})
	})

	return r, composite, nil
}
