package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/woragis/streamer-backend/internal/platform"
	"github.com/woragis/streamer-backend/internal/platforms/youtube"
	"github.com/woragis/streamer-backend/internal/store"
)

const platformPollInterval = 15 * time.Second

type youtubeRunner struct {
	cancel context.CancelFunc
	key    string
}

// StartPlatformSupervisor reloads YouTube pollers from database settings.
func StartPlatformSupervisor(ctx context.Context, st *store.Store) {
	go runPlatformSupervisor(ctx, st)
}

func runPlatformSupervisor(ctx context.Context, st *store.Store) {
	log.Printf("platform supervisor started (poll every %s)", platformPollInterval)

	var mu sync.Mutex
	runners := map[string]youtubeRunner{}

	syncOnce := func() {
		rooms, err := st.ListYouTubeReadyRooms(ctx)
		if err != nil {
			log.Printf("platform supervisor: list rooms: %v", err)
			return
		}

		desired := make(map[string]platform.ResolvedSettings, len(rooms))
		for _, room := range rooms {
			desired[room.RoomID] = room
		}

		mu.Lock()
		defer mu.Unlock()

		for roomID, runner := range runners {
			room, ok := desired[roomID]
			if !ok || settingsKey(room) != runner.key {
				runner.cancel()
				delete(runners, roomID)
				log.Printf("platform supervisor: stopped youtube poller for room %s", roomID)
			}
		}

		for roomID, room := range desired {
			key := settingsKey(room)
			if runner, ok := runners[roomID]; ok && runner.key == key {
				continue
			}
			if runner, ok := runners[roomID]; ok {
				runner.cancel()
			}

			pollerCtx, cancel := context.WithCancel(ctx)
			client := youtube.NewClient(room.GoogleAPIKey, room.YouTubeChannelID)
			poller := youtube.NewPoller(client, st, room.RoomID, room.YouTubeIdleSeconds)
			runners[roomID] = youtubeRunner{cancel: cancel, key: key}
			go poller.Run(pollerCtx)
			log.Printf("platform supervisor: started youtube poller for room %s", roomID)
		}
	}

	syncOnce()
	ticker := time.NewTicker(platformPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for roomID, runner := range runners {
				runner.cancel()
				delete(runners, roomID)
			}
			mu.Unlock()
			return
		case <-ticker.C:
			syncOnce()
		}
	}
}

func settingsKey(room platform.ResolvedSettings) string {
	return fmt.Sprintf("%s|%s|%d", room.GoogleAPIKey, room.YouTubeChannelID, room.YouTubeIdleSeconds)
}
