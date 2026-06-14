package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/woragis/streamer-backend/internal/config"
	"github.com/woragis/streamer-backend/internal/platforms/kick"
	"github.com/woragis/streamer-backend/internal/store"
)

type KickWebhookHandler struct {
	Store       *store.Store
	PlatformCfg config.PlatformConfig
	verifyFn    func(messageID, timestamp string, body []byte, signature string) error
}

func NewKickWebhookHandler(st *store.Store, platformCfg config.PlatformConfig) (*KickWebhookHandler, error) {
	pub, err := kick.DefaultPublicKeyParsed()
	if err != nil {
		return nil, err
	}
	return &KickWebhookHandler{
		Store:       st,
		PlatformCfg: platformCfg,
		verifyFn: func(messageID, timestamp string, body []byte, signature string) error {
			return kick.VerifySignature(pub, messageID, timestamp, body, signature)
		},
	}, nil
}

func (h *KickWebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	if !h.PlatformCfg.KickEnabled {
		WriteError(w, http.StatusServiceUnavailable, "kick webhooks disabled")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	eventType := r.Header.Get(kick.HeaderEventType)
	messageID := r.Header.Get(kick.HeaderMessageID)
	timestamp := r.Header.Get(kick.HeaderMessageTS)
	signature := r.Header.Get(kick.HeaderSignature)

	if !h.PlatformCfg.KickWebhookSkipVerify {
		if messageID == "" || timestamp == "" || signature == "" {
			WriteError(w, http.StatusUnauthorized, "missing kick signature headers")
			return
		}
		if err := h.verifyFn(messageID, timestamp, body, signature); err != nil {
			log.Printf("kick webhook: signature verification failed: %v", err)
			WriteError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	mapped, ok, err := kick.MapWebhook(eventType, body, h.PlatformCfg.KickChannelSlug)
	if err != nil {
		log.Printf("kick webhook: map %s: %v", eventType, err)
		WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()
	roomID := h.PlatformCfg.RoomID

	if mapped.Message != nil {
		result, err := h.Store.IngestMessage(ctx, roomID, *mapped.Message)
		if err != nil {
			log.Printf("kick webhook ingest message: %v", err)
			WriteError(w, http.StatusInternalServerError, "ingest failed")
			return
		}
		if result.Duplicate {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	if mapped.Event != nil {
		if _, err := h.Store.IngestStreamEvent(ctx, roomID, *mapped.Event); err != nil {
			if errors.Is(err, store.ErrDuplicateIngest) {
				w.WriteHeader(http.StatusOK)
				return
			}
			log.Printf("kick webhook ingest event: %v", err)
			WriteError(w, http.StatusInternalServerError, "ingest failed")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
