package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/woragis/streamer-backend/internal/bus"
	"github.com/woragis/streamer-backend/internal/config"
	"github.com/woragis/streamer-backend/internal/db"
	"github.com/woragis/streamer-backend/internal/dedup"
	appredis "github.com/woragis/streamer-backend/internal/redis"
	"github.com/woragis/streamer-backend/internal/queue"
	"github.com/woragis/streamer-backend/internal/store"
	"github.com/woragis/streamer-backend/internal/worker"
)

func main() {
	cfg := config.Load()
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer database.Close()

	redisClient, err := appredis.Connect(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis config: %v", err)
	}
	defer func() { _ = redisClient.Close() }()

	st := store.New(database)
	if err := st.Seed(ctx); err != nil {
		log.Fatalf("seed: %v", err)
	}

	if redisClient.Enabled() && redisClient.Status() == "ok" {
		redisBus := bus.NewRedis(redisClient.Raw(), cfg.InstanceID, nil)
		st.SetBus(redisBus)
		log.Printf("redis bus publisher enabled (instance %s)", cfg.InstanceID)
	}

	ingestQueue := queue.New(redisClient.Raw())
	dedupStore := dedup.New(redisClient.Raw())
	if ingestQueue != nil && ingestQueue.Enabled() {
		st.SetQueue(ingestQueue)
		st.SetDedup(dedupStore)
		worker.StartIngestConsumer(ctx, ingestQueue, st, cfg.InstanceID+"-ingest")
	} else {
		log.Printf("ingest queue disabled — worker runs platform pollers only")
	}

	log.Printf("platform worker started (room default=%s, settings from database)",
		config.WorkerRoomID())

	worker.StartPlatformSupervisor(ctx, st)

	<-ctx.Done()
	log.Printf("platform worker shutting down")
}
