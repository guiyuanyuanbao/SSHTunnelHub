package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"sshtunnelhub/internal/logger"
	"sshtunnelhub/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local hub
	},
}

type WSHub struct {
	tunnelSvc *service.TunnelService
	clients   map[*websocket.Conn]bool
	mu        sync.Mutex
}

func NewWSHub(tunnelSvc *service.TunnelService) *WSHub {
	hub := &WSHub{
		tunnelSvc: tunnelSvc,
		clients:   make(map[*websocket.Conn]bool),
	}

	// Register real-time log listener
	logger.GetLogger().SetListener(func(entry logger.LogEntry) {
		hub.BroadcastLog(entry)
	})

	go hub.broadcastLoop()
	return hub
}

func (h *WSHub) BroadcastLog(entry logger.LogEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients) == 0 {
		return
	}
	payload := gin.H{
		"type": "log",
		"data": entry,
	}
	for client := range h.clients {
		_ = client.WriteJSON(payload)
	}
}

func (h *WSHub) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Upgrade failed: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	// Send initial snapshot immediately
	h.sendSnapshot(conn)

	// Keep reading to detect client disconnection
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *WSHub) broadcastLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		clientCount := len(h.clients)
		h.mu.Unlock()

		if clientCount == 0 {
			continue
		}

		tunnels, err := h.tunnelSvc.ListTunnels()
		if err != nil {
			continue
		}
		stats, err := h.tunnelSvc.GetDashboardStats()
		if err != nil {
			continue
		}

		payload := gin.H{
			"type":    "tick",
			"time":    time.Now().Unix(),
			"tunnels": tunnels,
			"stats":   stats,
		}

		h.mu.Lock()
		for client := range h.clients {
			if err := client.WriteJSON(payload); err != nil {
				client.Close()
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

func (h *WSHub) sendSnapshot(conn *websocket.Conn) {
	tunnels, _ := h.tunnelSvc.ListTunnels()
	stats, _ := h.tunnelSvc.GetDashboardStats()
	payload := gin.H{
		"type":    "snapshot",
		"time":    time.Now().Unix(),
		"tunnels": tunnels,
		"stats":   stats,
	}
	_ = conn.WriteJSON(payload)
}
