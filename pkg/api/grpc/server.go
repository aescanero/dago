package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/aescanero/dago/internal/application/orchestrator"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server represents the gRPC API server
type Server struct {
	server       *grpc.Server
	listener     net.Listener
	orchestrator *orchestrator.Manager
	logger       *zap.Logger
}

// Config holds gRPC server configuration
type Config struct {
	Port         int
	Orchestrator *orchestrator.Manager
	Logger       *zap.Logger
}

// NewServer creates a new gRPC server
func NewServer(cfg *Config) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	grpcServer := grpc.NewServer()

	s := &Server{
		server:       grpcServer,
		listener:     listener,
		orchestrator: cfg.Orchestrator,
		logger:       cfg.Logger,
	}

	// Register services here
	// For MVP, service registration is placeholder
	// RegisterOrchestratorServiceServer(grpcServer, s)

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.logger.Info("starting gRPC server", zap.String("addr", s.listener.Addr().String()))

	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down gRPC server")

	s.server.GracefulStop()

	s.logger.Info("gRPC server shut down complete")
	return nil
}
