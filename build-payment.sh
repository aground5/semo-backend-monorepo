#!/bin/bash

# Payment 서비스 빌드 스크립트
# payment-server와 payment-sync 바이너리를 buildfile 폴더에 생성

set -e

# 프로젝트 루트 디렉토리 (스크립트 위치 기준)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 빌드 출력 디렉토리
BUILD_DIR="buildfile"

# 빌드 디렉토리가 없으면 생성
mkdir -p "$BUILD_DIR"

echo "=== Payment 서비스 빌드 시작 ==="

# 빌드 정보
VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS="-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildDate=$BUILD_DATE"

echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Build Date: $BUILD_DATE"
echo ""

# payment-server 빌드
echo "[1/2] Building payment-server..."
go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/payment-server" ./services/payment/cmd/server/main.go
echo "✓ payment-server 빌드 완료"

# payment-sync 빌드
echo "[2/2] Building payment-sync..."
go build -ldflags "$LDFLAGS" -o "$BUILD_DIR/payment-sync" ./services/payment/cmd/sync-plans/main.go
echo "✓ payment-sync 빌드 완료"

echo ""
echo "=== 빌드 완료 ==="
echo "생성된 파일:"
ls -lh "$BUILD_DIR"/payment-*
