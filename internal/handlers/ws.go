package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/woragis/streamer-backend/internal/ws"
)

type WSHandler struct {
	Hub   *ws.Hub
	Token string
}

func (h *WSHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		WriteError(w, http.StatusBadRequest, "room id required")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if h.Token != "" && token != h.Token {
		WriteError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		domain = "all"
	}

	conn, err := h.Hub.Upgrade(w, r)
	if err != nil {
		return
	}

	client := h.Hub.ClientConn(conn, roomID, domain)
	defer h.Hub.RemoveClient(client)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
