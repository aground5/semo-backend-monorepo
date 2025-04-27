# API 서비스

이 서비스는 외부 API 요청을 처리하고 클라이언트와의 통신을 담당합니다.

## 기능

- RESTful API 엔드포인트 제공
- 다른 서비스와의 통합 및 조정
- API 요청 검증 및 처리

## 아키텍처

이 서비스는 클린 아키텍처 원칙을 따릅니다:

- **Domain**: 비즈니스 엔티티 및 비즈니스 규칙
- **Usecase**: 애플리케이션 특정 비즈니스 규칙
- **Adapter**: 외부 시스템과의 인터페이스 (데이터베이스, API 등)
- **Infrastructure**: 기술적 세부 사항 구현 (데이터베이스 연결, HTTP 서버 등)

## 실행 방법

### 로컬에서 실행

```bash
# 서비스 디렉토리로 이동
cd services/api

# 의존성 설치
go mod download

# 개발 모드로 실행 (라이브 리로딩)
make dev

# 또는 직접 실행
go run cmd/server/main.go
```

### Docker로 실행

```bash
# 루트 디렉토리에서
make docker-api

# 또는 직접 Docker 명령어 사용
docker build -t api-service -f deployments/docker/api.Dockerfile .
docker run -p 8081:8081 api-service
```

## API 문서

API 문서는 Swagger를 통해 제공됩니다. 서비스 실행 후 다음 URL에서 확인할 수 있습니다:

```
http://localhost:8081/swagger/index.html
```

## 테스트

```bash
# 모든 테스트 실행
go test ./...

# 커버리지 확인
go test -cover ./...
``` 