# SEMO 백엔드 모노레포

이 프로젝트는 Go Workspaces를 활용한 모노레포 구조를 사용하여 여러 마이크로서비스를 관리합니다.

## 서비스 구성

- **알림 서버**: 사용자 알림 및 이벤트 처리
- **API 서버**: 외부 API 요청 처리
- **인증 서버**: 사용자 인증 및 권한 관리

## 시작하기

### 필수 조건

- Go 1.18 이상
- Docker와 Docker Compose
- Make

### 개발 환경 설정

```bash
# 프로젝트 클론
git clone https://github.com/wekeepgrowing/semo-backend-monorepo.git
cd semo-backend-monorepo

# 개발 환경 설정
make setup

# 모든 서비스 실행
make run
```

## 프로젝트 구조

```
monorepo/
├── .github/                       # GitHub 관련 설정 (CI/CD 등)
├── .golangci.yml                  # golangci-lint 설정
├── Makefile                       # 공통 명령어 및 스크립트
├── go.work                        # Go 워크스페이스 정의 파일
├── configs/                       # 공통 설정 파일
├── deployments/                   # 배포 관련 파일
├── pkg/                           # 서비스 간 공유 패키지
├── proto/                         # 공유 Protocol Buffers 정의
├── scripts/                       # 유틸리티 스크립트
├── services/                      # 각 서비스 디렉토리
│   ├── notification/              # 알림(이벤트) 서버
│   ├── api/                       # API 서버
│   └── auth/                      # 인증 서버
├── tools/                         # 개발 도구
└── docs/                          # 문서
```

## 주요 명령어

```bash
# 개발 환경 설정
make setup

# 모든 서비스 실행
make run

# 테스트 실행
make test

# 린트 체크
make lint

# 빌드
make build
```
