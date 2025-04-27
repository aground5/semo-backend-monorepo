package grpc

import (
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server gRPC 서버 구조체
type Server struct {
	server  *grpc.Server
	logger  *zap.Logger
	address string
}

// Config gRPC 서버 설정
type Config struct {
	Port    string
	Timeout int
}

// NewServer gRPC 서버 생성
func NewServer(cfg Config, logger *zap.Logger) *Server {
	// gRPC 서버 생성
	server := grpc.NewServer()

	// 헬스 체크 서비스 등록
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// 서버 리플렉션 설정 (개발 환경에서만 사용)
	reflection.Register(server)

	return &Server{
		server:  server,
		logger:  logger,
		address: fmt.Sprintf(":%s", cfg.Port),
	}
}

// RegisterServices gRPC 서비스 등록
func (s *Server) RegisterServices() {
	// TODO: gRPC 서비스 등록
	// 예: authpb.RegisterAuthServiceServer(s.server, &authService{})
}

// Start gRPC 서버 시작
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("gRPC 서버 리스너 생성 실패: %w", err)
	}

	s.logger.Info("gRPC 서버 시작",
		zap.String("address", s.address),
	)

	return s.server.Serve(listener)
}

// Stop gRPC 서버 중지
func (s *Server) Stop() {
	s.logger.Info("gRPC 서버 종료 중...")
	s.server.GracefulStop()
	s.logger.Info("gRPC 서버 종료 완료")
}
