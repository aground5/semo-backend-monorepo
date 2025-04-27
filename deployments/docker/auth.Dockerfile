FROM golang:1.21-alpine AS builder

WORKDIR /app

# 빌드에 필요한 의존성 설치
RUN apk add --no-cache git

# Go 모듈 캐싱을 위한 의존성 복사 및 다운로드
COPY services/auth/go.mod services/auth/go.sum ./
RUN go mod download

# 소스 코드 복사
COPY services/auth/ ./services/auth/
COPY pkg/ ./pkg/
COPY proto/ ./proto/

# 빌드
WORKDIR /app/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/auth-server ./cmd/server/main.go

# 실행 이미지
FROM alpine:3.18

WORKDIR /app

# 필요한 시스템 패키지 설치
RUN apk add --no-cache ca-certificates tzdata

# 실행 파일 복사
COPY --from=builder /app/auth-server /app/auth-server

# 설정 파일 복사
COPY configs/dev /app/configs/dev

# 실행
EXPOSE 8082
CMD ["/app/auth-server"] 