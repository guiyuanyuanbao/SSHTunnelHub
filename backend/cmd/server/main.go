package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"sshtunnelhub/internal/crypto"
	"sshtunnelhub/internal/db"
	"sshtunnelhub/internal/handler"
	"sshtunnelhub/internal/model"
	"sshtunnelhub/internal/service"
	"sshtunnelhub/internal/tunnel"
	"sshtunnelhub/internal/webui"
)

func main() {
	port := flag.Int("port", 9090, "Port to listen on")
	dataDir := flag.String("data-dir", "./data", "Directory to store database and secret keys")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" && *port == 9090 {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			*port = p
		}
	}
	if envDataDir := os.Getenv("DATA_DIR"); envDataDir != "" && *dataDir == "./data" {
		*dataDir = envDataDir
	}

	log.Println("==================================================")
	log.Println("           SSHTunnelHub Starting...               ")
	log.Println("==================================================")

	// 1. Initialize secret key
	if err := crypto.InitSecretKey(*dataDir); err != nil {
		log.Fatalf("Failed to initialize secret key: %v", err)
	}

	// 2. Initialize database
	database, err := db.InitDB(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 3. Initialize Tunnel Manager
	tunnelMgr := tunnel.NewTunnelManager()

	// 4. Initialize Services
	hostSvc := service.NewHostService()
	tunnelSvc := service.NewTunnelService(tunnelMgr)

	// 5. Auto-start configured tunnels
	var autoTunnels []model.Tunnel
	if err := database.Where("auto_start = ?", true).Find(&autoTunnels).Error; err == nil && len(autoTunnels) > 0 {
		var hosts []model.Host
		if err := database.Find(&hosts).Error; err == nil {
			hostMap := make(map[uint]model.Host, len(hosts))
			for _, h := range hosts {
				hostMap[h.ID] = h
			}
			tunnelMgr.AutoStartAll(autoTunnels, hostMap)
		}
	}

	// 6. Initialize Handlers
	hostHandler := handler.NewHostHandler(hostSvc)
	tunnelHandler := handler.NewTunnelHandler(tunnelSvc)
	logHandler := handler.NewLogHandler()
	wsHub := handler.NewWSHub(tunnelSvc)

	// 7. Setup Web Router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Simple logger middleware
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.URL.Path != "/ws" && c.Request.URL.Path != "/api/ws" {
			log.Printf("[HTTP] %-6s %-25s %3d %v", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
		}
	})

	// CORS middleware for local frontend dev
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// API Routes
	api := router.Group("/api")
	{
		api.GET("/hosts", hostHandler.List)
		api.GET("/hosts/:id", hostHandler.Get)
		api.POST("/hosts", hostHandler.Create)
		api.PUT("/hosts/:id", hostHandler.Update)
		api.DELETE("/hosts/:id", hostHandler.Delete)
		api.POST("/hosts/:id/test", hostHandler.TestSavedHost)
		api.POST("/hosts/test", hostHandler.TestRawHost)

		api.GET("/tunnels", tunnelHandler.List)
		api.GET("/tunnels/:id", tunnelHandler.Get)
		api.POST("/tunnels", tunnelHandler.Create)
		api.PUT("/tunnels/:id", tunnelHandler.Update)
		api.DELETE("/tunnels/:id", tunnelHandler.Delete)
		api.POST("/tunnels/:id/start", tunnelHandler.Start)
		api.POST("/tunnels/:id/stop", tunnelHandler.Stop)
		api.POST("/tunnels/:id/restart", tunnelHandler.Restart)
		api.GET("/dashboard/stats", tunnelHandler.Stats)

		api.GET("/logs", logHandler.List)
		api.DELETE("/logs", logHandler.Clear)

		api.GET("/ws", wsHub.HandleWS)
	}

	router.GET("/ws", wsHub.HandleWS)

	// Register web UI (both embedded FS and filesystem fallback)
	webui.Register(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: router,
	}

	// 8. Run server in goroutine
	go func() {
		log.Printf("[Web] SSHTunnelHub listening on http://127.0.0.1:%d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server ListenAndServe error: %v", err)
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Hub] Shutting down server and closing tunnels...")

	tunnelMgr.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Hub] Server forced to shutdown: %v", err)
	}

	log.Println("[Hub] Server stopped cleanly")
}
