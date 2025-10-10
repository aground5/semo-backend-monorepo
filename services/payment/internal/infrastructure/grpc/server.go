package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	config   *config.Config
	logger   *zap.Logger
	server   *grpc.Server
	listener net.Listener
}

func NewServer(cfg *config.Config, logger *zap.Logger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.GRPC.Host, s.config.Server.GRPC.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	s.server = grpc.NewServer()

	// Register payment service here
	// pb.RegisterPaymentServiceServer(s.server, handler)

	s.logger.Info("Starting gRPC server", zap.String("address", addr))

	return s.server.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		s.server.GracefulStop()
	}
	return nil
}
