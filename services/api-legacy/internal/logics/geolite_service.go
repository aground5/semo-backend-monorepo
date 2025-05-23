package logics

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"semo-server/configs-legacy"
	pb "semo-server/proto/geolite" // proto 파일의 go_package 옵션에 맞게 경로를 설정합니다.
)

// GeoIPResponse는 ox‑geolite 서비스에서 반환하는 지리정보를 담는 구조체입니다.
type GeoIPResponse struct {
	CountryISO string  `json:"country_iso"`
	Country    string  `json:"country"`
	City       string  `json:"city"`
	TimeZone   string  `json:"time_zone"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// GetGeoIP는 주어진 IP에 대해 ox‑geolite gRPC 서비스에 요청을 보내고, 응답을 파싱하여 반환합니다.
func GetGeoIP(ip string) (*GeoIPResponse, error) {
	// Microservices 설정에서 "ox-geolite" 서비스 찾기
	var geoServiceConfig *configs.MicroserviceConfig
	for _, ms := range configs.Configs.Microservices {
		if ms.Name == "ox-geolite" {
			geoServiceConfig = &ms
			break
		}
	}
	if geoServiceConfig == nil {
		return nil, fmt.Errorf("ox‑geolite service configuration not found")
	}

	// gRPC 주소 구성: 주소와 gRPC 포트 사용 (포트가 없으면 기본 9090 사용)
	address := geoServiceConfig.Address
	port := geoServiceConfig.GrpcPort
	if port == "" {
		port = "9090"
	}
	target := fmt.Sprintf("%s:%s", address, port)

	// gRPC 서버 연결 생성 (Insecure 연결 – 실제 운영환경에서는 TLS 설정 필요)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial ox‑geolite gRPC service: %w", err)
	}
	defer conn.Close()

	client := pb.NewGeoLiteServiceClient(conn)

	// 타임아웃이 있는 context 생성 (5초)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// gRPC 요청 전송
	req := &pb.GeoIPRequest{Ip: ip}
	resp, err := client.GetGeoIP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC call to ox‑geolite failed: %w", err)
	}

	// 응답 변환 (pb.GeoIPResponse → GeoIPResponse)
	result := &GeoIPResponse{
		CountryISO: resp.GetCountryIso(),
		Country:    resp.GetCountry(),
		City:       resp.GetCity(),
		TimeZone:   resp.GetTimeZone(),
		Latitude:   resp.GetLatitude(),
		Longitude:  resp.GetLongitude(),
	}
	return result, nil
}
