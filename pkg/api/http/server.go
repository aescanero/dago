package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aescanero/dago/internal/application/orchestrator"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server represents the HTTP API server
type Server struct {
	router      *gin.Engine
	server      *http.Server
	orchestrator *orchestrator.Manager
	logger      *zap.Logger
}

// Config holds HTTP server configuration
type Config struct {
	Port         int
	Orchestrator *orchestrator.Manager
	Logger       *zap.Logger
}

// NewServer creates a new HTTP server
func NewServer(cfg *Config) *Server {
	// Set Gin mode based on logger
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(cfg.Logger))

	s := &Server{
		router:       router,
		orchestrator: cfg.Orchestrator,
		logger:       cfg.Logger,
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	return s
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.handleHealth)

	// Metrics
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Graph endpoints
		v1.POST("/graphs", s.handleSubmitGraph)
		v1.GET("/graphs", s.handleListGraphs)
		v1.GET("/graphs/:id", s.handleGetGraph)
		v1.GET("/graphs/:id/status", s.handleGetStatus)
		v1.GET("/graphs/:id/result", s.handleGetResult)
		v1.POST("/graphs/:id/cancel", s.handleCancelGraph)
	}
}

// SetupWebSocket adds WebSocket handler to the server
func (s *Server) SetupWebSocket(handler interface{}) {
	// Type assert to get the handler
	if wsHandler, ok := handler.(interface {
		HandleGraphStream(*gin.Context)
	}); ok {
		s.router.GET("/api/v1/graphs/:id/ws", wsHandler.HandleGraphStream)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", zap.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info("HTTP server shut down complete")
	return nil
}

// requestLogger is a middleware for request logging
func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)

		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()))
	}
}
