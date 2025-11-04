package main

import (
	"database/sql"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/richxcame/ride-hailing/internal/realtime"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/middleware"
	"github.com/richxcame/ride-hailing/pkg/redis"
	ws "github.com/richxcame/ride-hailing/pkg/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load("realtime")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	port := cfg.Server.Port
	jwtSecret := cfg.JWT.Secret

	// Connect to PostgreSQL
	dsn := cfg.Database.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL database")

	// Connect to Redis
	redisClient, err := redis.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	// Create WebSocket hub
	hub := ws.NewHub()
	go hub.Run()
	log.Println("WebSocket hub started")

	// Create service and handler
	service := realtime.NewService(hub, db, redisClient)
	handler := realtime.NewHandler(service)

	// Set up Gin router
	router := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // TODO: Configure properly for production
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	router.Use(cors.New(config))

	// Health check and metrics (no auth required)
	router.GET("/healthz", handler.HealthCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := router.Group("/api/v1")
	{
		// WebSocket connection (requires authentication)
		api.GET("/ws", middleware.AuthMiddleware(jwtSecret), handler.HandleWebSocket)

		// Chat history
		api.GET("/rides/:ride_id/chat", middleware.AuthMiddleware(jwtSecret), handler.GetChatHistory)

		// Driver location
		api.GET("/drivers/:driver_id/location", middleware.AuthMiddleware(jwtSecret), handler.GetDriverLocation)

		// Stats (admin only - TODO: add admin middleware)
		api.GET("/stats", middleware.AuthMiddleware(jwtSecret), handler.GetStats)

		// Internal endpoints (for other services to broadcast)
		internal := api.Group("/internal")
		{
			internal.POST("/broadcast/ride", handler.BroadcastRideUpdate)
			internal.POST("/broadcast/user", handler.BroadcastToUser)
		}
	}

	// Start server
	addr := ":" + port
	log.Printf("Real-time service starting on port %s", port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
