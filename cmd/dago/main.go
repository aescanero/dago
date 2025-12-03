package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aescanero/dago/internal/application/orchestrator"
	"github.com/aescanero/dago/internal/application/workers"
	"github.com/aescanero/dago/internal/config"
	"github.com/aescanero/dago/pkg/adapters/events/redis"
	"github.com/aescanero/dago/pkg/adapters/llm"
	"github.com/aescanero/dago/pkg/adapters/metrics/prometheus"
	redisstorage "github.com/aescanero/dago/pkg/adapters/storage/redis"
	"github.com/aescanero/dago/pkg/api/grpc"
	"github.com/aescanero/dago/pkg/api/http"
	"github.com/aescanero/dago/pkg/api/websocket"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version is set by build flags
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting DA Orchestrator",
		zap.String("version", Version),
		zap.String("build_time", BuildTime))

	// Initialize Redis client
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to connect to Redis", zap.Error(err))
	}
	logger.Info("connected to Redis", zap.String("addr", cfg.Redis.Addr))

	// Initialize adapters
	eventBus, err := redis.NewStreamsEventBus(
		redisClient,
		"dago-workers",
		fmt.Sprintf("dago-%d", os.Getpid()),
		logger,
	)
	if err != nil {
		logger.Fatal("failed to create event bus", zap.Error(err))
	}

	stateStorage := redisstorage.NewStateStorage(
		redisClient,
		24*time.Hour, // 24 hour TTL for states
		logger,
	)

	llmClient, err := llm.NewClient(&llm.Config{
		Provider: cfg.LLM.Provider,
		APIKey:   cfg.LLM.APIKey,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to create LLM client", zap.Error(err))
	}

	metricsCollector := prometheus.NewCollector()

	// Initialize application components
	validator := orchestrator.NewValidator()

	orchestratorMgr := orchestrator.NewManager(
		eventBus,
		stateStorage,
		metricsCollector,
		validator,
		logger,
		cfg.Timeouts.GraphExecutionTimeout,
		cfg.Timeouts.NodeExecutionTimeout,
	)

	workerPool := workers.NewPool(
		cfg.Workers.PoolSize,
		eventBus,
		stateStorage,
		llmClient,
		metricsCollector,
		logger,
		cfg.Workers.HealthCheckInterval,
	)

	// Start worker pool
	if err := workerPool.Start(); err != nil {
		logger.Fatal("failed to start worker pool", zap.Error(err))
	}

	// Initialize API servers
	httpServer := http.NewServer(&http.Config{
		Port:         cfg.HTTPPort,
		Orchestrator: orchestratorMgr,
		Logger:       logger,
	})

	// Add WebSocket handler to HTTP server
	wsHandler := websocket.NewHandler(eventBus, logger)
	httpServer.SetupWebSocket(wsHandler)

	grpcServer, err := grpc.NewServer(&grpc.Config{
		Port:         cfg.GRPCPort,
		Orchestrator: orchestratorMgr,
		Logger:       logger,
	})
	if err != nil {
		logger.Fatal("failed to create gRPC server", zap.Error(err))
	}

	// Start servers
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	go func() {
		if err := grpcServer.Start(); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	logger.Info("DA Orchestrator started",
		zap.Int("http_port", cfg.HTTPPort),
		zap.Int("grpc_port", cfg.GRPCPort),
		zap.Int("worker_pool_size", cfg.Workers.PoolSize))

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.Info("received shutdown signal")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.ShutdownTimeout)
	defer cancel()

	// Shutdown components
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("gRPC server shutdown error", zap.Error(err))
	}

	if err := workerPool.Shutdown(shutdownCtx); err != nil {
		logger.Error("worker pool shutdown error", zap.Error(err))
	}

	if err := orchestratorMgr.Shutdown(shutdownCtx); err != nil {
		logger.Error("orchestrator shutdown error", zap.Error(err))
	}

	if err := redisClient.Close(); err != nil {
		logger.Error("Redis close error", zap.Error(err))
	}

	logger.Info("DA Orchestrator shut down complete")
}

// initLogger initializes the logger based on log level
func initLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	return logger
}
