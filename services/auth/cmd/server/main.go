package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 로거 초기화
	logger := log.New(os.Stdout, "[AUTH] ", log.LstdFlags)
	logger.Println("인증 서비스를 시작합니다...")

	// gRPC 서버 생성
	server := grpc.NewServer()

	// 헬스 체크 서비스 등록
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// 서버 리플렉션 사용 (gRPC CLI 디버깅용)
	reflection.Register(server)

	// 서버 시작
	listener, err := net.Listen("tcp", ":8082")
	if err != nil {
		logger.Fatalf("포트 바인딩 실패: %v", err)
	}

	go func() {
		logger.Printf("gRPC 서버가 %s에서 시작되었습니다", listener.Addr())
		if err := server.Serve(listener); err != nil {
			logger.Fatalf("서버 시작 실패: %v", err)
		}
	}()

	// 그레이스풀 종료를 위한 시그널 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("서버를 종료합니다...")

	// gRPC 서버 정상 종료
	server.GracefulStop()
	logger.Println("서버가 정상적으로 종료되었습니다")
}
