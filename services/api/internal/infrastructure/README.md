# 인프라스트럭처 계층

이 디렉토리는 API 서비스의 인프라스트럭처 계층을 포함하고 있습니다. 인프라스트럭처 계층은 기술적 세부 사항의 구현을 담당합니다.

## 구조

- **db**: 데이터베이스 연결 및 마이그레이션
- **http**: HTTP 서버 설정
- **grpc**: gRPC 서버 설정
- **mail**: 이메일 서비스
- **storage**: 파일 저장소
- **messaging**: 메시징 서비스

## 사용 방법

### 데이터베이스 연결 설정

```go
package db

import (
	"fmt"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewDatabaseConnection 데이터베이스 연결 생성
func NewDatabaseConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		"disable",
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	return db, nil
}
```

### HTTP 서버 설정

```go
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/config"
)

// Server HTTP 서버 구조체
type Server struct {
	router *gin.Engine
	server *http.Server
	config *config.Config
}

// NewServer HTTP 서버 생성
func NewServer(cfg *config.Config) *Server {
	if cfg.Server.HTTP.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	return &Server{
		router: router,
		config: cfg,
	}
}

// Start 서버 시작
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    ":" + s.config.Server.HTTP.Port,
		Handler: s.router,
	}

	return s.server.ListenAndServe()
}

// Shutdown 서버 종료
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Router 라우터 반환
func (s *Server) Router() *gin.Engine {
	return s.router
}
```

### gRPC 서버 설정

```go
package grpc

import (
	"net"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/config"
	"google.golang.org/grpc"
)

// Server gRPC 서버 구조체
type Server struct {
	server *grpc.Server
	config *config.Config
}

// NewServer gRPC 서버 생성
func NewServer(cfg *config.Config) *Server {
	server := grpc.NewServer()

	return &Server{
		server: server,
		config: cfg,
	}
}

// Start 서버 시작
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.config.Server.GRPC.Port)
	if err != nil {
		return err
	}

	return s.server.Serve(lis)
}

// Shutdown 서버 종료
func (s *Server) Shutdown() {
	s.server.GracefulStop()
}

// Server gRPC 서버 반환
func (s *Server) Server() *grpc.Server {
	return s.server
}
```

## 가이드라인

1. 인프라스트럭처 계층은 기술적 구현 세부 사항을 캡슐화합니다.
2. 상위 계층과의 인터페이스는 어댑터 계층을 통해 이루어져야 합니다.
3. 인프라스트럭처 코드는 기술에 특화된 코드만 포함해야 합니다.
4. 인프라스트럭처 구현체는 적절한 어댑터 인터페이스를 구현해야 합니다. 