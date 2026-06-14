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
	calHandler := &handlers.CalisthenicsHandler{Store: st}
	lcHandler := &handlers.LeetCodeHandler{Store: st}

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
