package handler

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/voxmesh/pkg/auth"
	slogx "github.com/voxmesh/pkg/log"

	"github.com/voxmesh/ws-gateway/internal/ws"
)

type WSHandler struct {
	hub *ws.Hub
}

func NewWSHandler(onDisconnect func(userID string)) *WSHandler {
	return &WSHandler{hub: ws.NewHub(onDisconnect)}
}

func (h *WSHandler) Upgrade() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		token := c.Query("token")
		slogx.Info("[ws] upgrade request: token_len=%d token_preview=%.10s...", len(token), token)
		if token == "" {
			slogx.Info("[ws] missing token in query params")
			c.Close()
			return
		}

		claims, err := auth.ValidateAccessToken(token)
		if err != nil {
			slogx.Info("[ws] token validation failed: %v", err)
			c.Close()
			return
		}

		slogx.Info("[ws] client authenticated: user=%s id=%s", claims.Username, claims.Subject)

		client := ws.NewClient(claims.Subject, claims.Username, c.Conn, h.hub)
		h.hub.Register <- client

		go client.WritePump()
		go client.ReadPump()

		<-client.Done()
	})
}

func (h *WSHandler) Run() {
	go h.hub.Run()
}
