// File: internal/app/grpc/public_key_service.go
package grpcEngine

import (
	"context"

	"authn-server/configs"
	pb "authn-server/proto/publickey" // proto 패키지 import (패키지명 및 경로는 실제 설정에 맞게 조정)
)

// PublicKeyServiceServer는 proto에 정의된 PublicKeyServiceServer 인터페이스를 구현합니다.
type PublicKeyServiceServer struct {
	pb.UnimplementedPublicKeyServiceServer
}

// NewPublicKeyServiceServer는 PublicKeyServiceServer의 인스턴스를 생성합니다.
func NewPublicKeyServiceServer() *PublicKeyServiceServer {
	return &PublicKeyServiceServer{}
}

// GetPublicKey는 클라이언트 요청에 대해 ECDSA 공개키(PublicKey)를 반환합니다.
func (s *PublicKeyServiceServer) GetPublicKey(ctx context.Context, req *pb.Empty) (*pb.PublicKeyResponse, error) {
	// configs.Configs.Secrets.EcdsaPublicKey에 PEM 인코딩된 공개키가 저장되어 있다고 가정합니다.
	publicKey := configs.Configs.Secrets.EcdsaPublicKey
	return &pb.PublicKeyResponse{
		PublicKey: publicKey,
	}, nil
}
