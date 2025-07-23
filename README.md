# SEMO 백엔드 모노레포

## 🏗 아키텍처 개요

이 프로젝트는 Go Workspaces를 활용한 모노레포 구조를 사용하여 여러 마이크로서비스를 관리합니다.
각 서비스는 최소 HTTP API를 제공합니다. 서비스 간 통신은 Protocol Buffers를 사용합니다.

## 📦 서비스 구성

### 현재 구현된 서비스

- **geo**: 위치 정보 서비스
- **payment**: 결제 서비스

## 🚀 시작하기

### 필수 조건

- Go 1.23.6 이상
- Docker 및 Docker Compose
- Make
- PostgreSQL 14

### 빠른 시작

```bash
# 프로젝트 클론
git clone https://github.com/wekeepgrowing/semo-backend-template.git
cd semo-backend-template

# 개발 환경 설정
make setup

# 모든 서비스 실행
make run

# 특정 서비스만 핫 리로드로 실행 (예: geo 서비스)
make air-geo
```

## 📁 프로젝트 구조

```
/
├── bin/                # 컴파일된 바이너리
├── configs/            # 설정 파일
│   ├── dev/           # 개발 환경 설정
│   └── example/       # 설정 템플릿 예제
├── deployments/        # 배포 설정
│   ├── docker/        # 각 서비스별 Dockerfile
│   └── k8s/helm/      # Kubernetes Helm 차트
├── docs/              # 문서
├── pkg/               # 공유 Go 패키지
│   ├── config/        # 공통 설정 처리 (Viper)
│   └── logger/        # 로깅 구현 (Echo, GORM, gRPC, Zap)
├── proto/             # Protocol Buffer 정의
├── scripts/           # 유틸리티 스크립트
├── services/                      # 각 서비스 디렉토리
│   ├── geo/                       # 위치 정보 서비스
│   └── payment/                   # 결제 서비스
├── tools/             # Go 도구 의존성
├── go.work           # Go 워크스페이스 정의
└── Makefile          # 빌드 및 개발 명령어
```

### 서비스 구조 패턴

각 서비스는 다음과 같은 구조를 따릅니다:

```
services/[서비스명]/
├── cmd/server/         # 서비스 진입점
├── internal/
│   ├── adapter/       # 인터페이스 어댑터 (HTTP/gRPC 핸들러, 리포지토리)
│   ├── domain/        # 도메인 엔티티 및 리포지토리 인터페이스
│   ├── infrastructure/# 인프라 구현체
│   ├── usecase/       # 비즈니스 로직 구현
│   └── config/        # 서비스별 설정
```

## 🛠 주요 명령어

### 개발

```bash
make setup          # 개발 환경 설정
make run           # 모든 서비스를 docker-compose로 실행
make air-geo       # geo 서비스를 핫 리로드로 실행
```

### 빌드

```bash
make build         # 모든 서비스 빌드
make docker-geo    # geo 서비스 Docker 이미지 빌드
```

### 코드 생성

```bash
make proto-gen     # protobuf 코드 생성
make mock          # mockery를 사용한 모의 객체 생성
```

### 품질 관리

```bash
make test          # 모든 테스트 실행
make lint          # golangci-lint 실행
make tidy          # 모든 Go 모듈 정리
```

## 🔧 기술 스택

- **언어**: Go 1.23.6
- **웹 프레임워크**: Echo v4
- **RPC**: gRPC
- **로깅**: Zap (구조화된 로깅)
- **설정 관리**: Viper
- **데이터베이스**: PostgreSQL 14
- **컨테이너**: Docker (distroless 이미지 사용)
- **오케스트레이션**: Kubernetes (Helm 사용)
- **개발 도구**: Air (핫 리로드), Mockery (모킹)

## 💡 주요 개발 패턴

1. **클린 아키텍처**: 계층 간 엄격한 관심사 분리
2. **의존성 주입**: 생성자 주입을 통한 인터페이스 기반 설계
3. **에러 처리**: 적절한 전파를 위한 구조화된 에러 타입
4. **설정 관리**: 환경별 YAML 설정 파일
5. **서비스 통신**: Protocol Buffers를 사용한 gRPC
6. **API 설계**: gRPC 엔드포인트와 함께 RESTful HTTP 제공

## 🗄 데이터베이스

- **타입**: PostgreSQL 14
- **패턴**: 서비스당 하나의 데이터베이스 (예: geo_db, auth_db)
- **초기화**: `scripts/create_multiple_dbs.sh`를 사용한 설정
- **설정**: 서비스 YAML 설정 파일에서 데이터베이스 연결 관리

## 📊 로깅

- **로거**: 구조화된 필드를 지원하는 Zap
- **레벨**: debug, info, warn, error, dpanic, panic, fatal
- **형식**: JSON (프로덕션) 또는 콘솔 (개발)
- **기능**:
  - 요청/응답 미들웨어 로깅
  - 개발 환경에서 호출자 정보 표시
  - 컨텍스트 인식 로깅

## 🧪 테스트

- **단위 테스트**: `_test.go` 접미사를 사용하는 Go 규칙 준수
- **모킹**: 모의 객체 생성을 위한 Mockery 사용 (`.mockery.yaml` 참조)
- **테스트 실행**: `make test`
- **예제**: `services/geo/internal/usecase/geo/examples_test.go` 참조

## 🚢 배포

### 로컬 개발

```bash
make run           # docker-compose로 모든 서비스 시작
make air-geo       # geo 서비스 핫 리로드
```

### 프로덕션

- **빌드**: distroless 베이스 이미지를 사용한 멀티 스테이지 Dockerfile
- **배포**: Helm 차트를 사용한 Kubernetes
- **차트 위치**: `/deployments/k8s/helm/`
- **리소스**: Deployment, Service, ConfigMap, Ingress, HPA

### 빌드 변수

서비스는 다음 정보와 함께 빌드됩니다:

- 버전 (git 태그에서)
- 커밋 해시
- 빌드 날짜

## ⚠️ 중요 사항

1. **서비스 독립성**: 각 서비스는 독립적으로 배포 가능해야 합니다
2. **설정 관리**: 절대 시크릿을 커밋하지 마세요. 템플릿으로 예제 설정을 사용하세요
3. **로깅**: 항상 적절한 레벨의 구조화된 로깅을 사용하세요
4. **테스트**: 기존 패턴을 따라 새 기능에 대한 테스트를 작성하세요
5. **Protocol Buffers**: `.proto` 파일 수정 후 `make proto-gen` 실행
6. **의존성**: 새 의존성 추가 후 `make tidy` 실행
