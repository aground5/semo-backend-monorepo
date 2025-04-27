# 알림 서비스

이 서비스는 사용자 알림 및 이벤트 처리를 담당합니다.

## 기능

- 실시간 사용자 알림 생성 및 관리
- 이벤트 기반 알림 처리
- 알림 이력 관리 및 조회

## 아키텍처

이 서비스는 클린 아키텍처 원칙을 따릅니다:

- **Domain**: 비즈니스 엔티티 및 비즈니스 규칙
- **Usecase**: 애플리케이션 특정 비즈니스 규칙
- **Adapter**: 외부 시스템과의 인터페이스 (데이터베이스, API 등)
- **Infrastructure**: 기술적 세부 사항 구현 (데이터베이스 연결, 메시징 등)

## 실행 방법

### 로컬에서 실행

```bash
# 서비스 디렉토리로 이동
cd services/notification

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
make docker-notification

# 또는 직접 Docker 명령어 사용
docker build -t notification-service -f deployments/docker/notification.Dockerfile .
docker run -p 8080:8080 notification-service
```

## API 문서

API 문서는 Swagger를 통해 제공됩니다. 서비스 실행 후 다음 URL에서 확인할 수 있습니다:

```
http://localhost:8080/swagger/index.html
```

## 테스트

```bash
# 모든 테스트 실행
go test ./...

# 커버리지 확인
go test -cover ./...
``` 