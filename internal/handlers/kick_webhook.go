package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/woragis/streamer-backend/internal/platforms/kick"
	"github.com/woragis/streamer-backend/internal/store"
)

type KickWebhookHandler struct {
	Store    *store.Store
	verifyFn func(messageID, timestamp string, body []byte, signature string) error
}

func NewKickWebhookHandler(st *store.Store) (*KickWebhookHandler, error) {
	pub, err := kick.DefaultPublicKeyParsed()
	if err != nil {
		return nil, err
	}
	return &KickWebhookHandler{
		Store: st,
		verifyFn: func(messageID, timestamp string, body []byte, signature string) error {
			return kick.VerifySignature(pub, messageID, timestamp, body, signature)
		},
	}, nil
}

func (h *KickWebhookHandler) Receive(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	eventType := r.Header.Get(kick.HeaderEventType)
	messageID := r.Header.Get(kick.HeaderMessageID)
	timestamp := r.Header.Get(kick.HeaderMessageTS)
	signature := r.Header.Get(kick.HeaderSignature)

	broadcasterSlug := peekBroadcasterSlug(body)
	settings, ok, err := h.Store.ResolveKickSettings(r.Context(), broadcasterSlug)
	if err != nil {
		log.Printf("kick webhook: resolve settings: %v", err)
		WriteError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !ok || !settings.KickReady() {
		w.WriteHeader(http.StatusOK)
		return
	}

	if !settings.KickWebhookSkipVerify {
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

	mapped, ok, err := kick.MapWebhook(eventType, body, settings.KickChannelSlug, messageID)
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
	roomID := settings.RoomID

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

func peekBroadcasterSlug(body []byte) string {
	var payload struct {
		Broadcaster struct {
			ChannelSlug string `json:"channel_slug"`
		} `json:"broadcaster"`
	}
	_ = json.Unmarshal(body, &payload)
	return payload.Broadcaster.ChannelSlug
}
