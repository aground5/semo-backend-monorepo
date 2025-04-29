// File: /Users/k2zoo/Documents/growingup/ox-hr/authn/internal/logics/public_key_service.go
package logics

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"semo-server/configs"
	pb "semo-server/proto/publickey"
)

// PublicKeyService는 gRPC 클라이언트를 이용해 공개키를 가져오는 기능을 제공합니다.
type PublicKeyService struct {
	client pb.PublicKeyServiceClient
	conn   *grpc.ClientConn
}

// NewPublicKeyService는 gRPC 서버와 연결된 PublicKeyService 인스턴스를 생성합니다.
func NewPublicKeyService(name string) (*PublicKeyService, error) {
	// gRPC 서버 포트를 설정합니다.
	var microservice configs.MicroserviceConfig

	for i := range configs.Configs.Microservices {
		if configs.Configs.Microservices[i].Name == name {
			microservice = configs.Configs.Microservices[i]
			break
		}
	}

	grpcAddress := microservice.Address
	grpcPort := microservice.GrpcPort
	if grpcPort == "" {
		grpcPort = "9090"
	}
	target := fmt.Sprintf("%s:%s", grpcAddress, grpcPort)

	// gRPC 서버에 Insecure 연결 (실제 환경에서는 TLS 사용 권장)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server at %s: %w", target, err)
	}

	client := pb.NewPublicKeyServiceClient(conn)
	return &PublicKeyService{
		client: client,
		conn:   conn,
	}, nil
}

// Close는 gRPC 연결을 종료합니다.
func (s *PublicKeyService) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// GetPublicKey는 gRPC 서버에 요청하여 공개키를 가져옵니다.
func (s *PublicKeyService) GetPublicKey() (string, error) {
	// 5초 타임아웃 컨텍스트 생성
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// proto에서 정의된 Empty 메시지를 전송 (요청 페이로드 없음)
	req := &pb.Empty{}
	resp, err := s.client.GetPublicKey(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to get public key from gRPC server: %w", err)
	}

	return resp.PublicKey, nil
}
