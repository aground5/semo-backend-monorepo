package grpcEngine

import (
	"net"

	"authn-server/configs"
	pb "authn-server/proto/publickey" // proto 패키지 경로는 실제 설정에 맞게 수정
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GRPCServer wraps the gRPC server instance.
type GRPCServer struct {
	server *grpc.Server
}

// NewGRPCServer creates a new GRPCServer instance.
func NewGRPCServer() *GRPCServer {
	return &GRPCServer{
		server: grpc.NewServer(),
	}
}

// Start starts the gRPC server on the configured port.
func (s *GRPCServer) Start() {
	port := configs.Configs.Service.GrpcPort
	if port == "" {
		port = "9090"
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		configs.Logger.Fatal("failed to listen", zap.Error(err))
	}

	// Register PublicKeyService
	pb.RegisterPublicKeyServiceServer(s.server, NewPublicKeyServiceServer())

	configs.Logger.Info("gRPC server started", zap.String("port", port))
	if err := s.server.Serve(listener); err != nil {
		configs.Logger.Fatal("failed to serve", zap.Error(err))
	}
}

// Shutdown gracefully stops the gRPC server.
func (s *GRPCServer) Shutdown() {
	s.server.GracefulStop()
}
