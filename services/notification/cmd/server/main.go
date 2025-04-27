package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 로거 초기화
	logger := log.New(os.Stdout, "[NOTIFICATION] ", log.LstdFlags)
	logger.Println("알림 서비스를 시작합니다...")

	// Echo 서버 생성
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 라우터 설정
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "notification",
		})
	})

	// 서버 시작
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("서버 시작 실패: %v", err)
		}
	}()

	// 그레이스풀 종료를 위한 시그널 처리
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("서버를 종료합니다...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Fatalf("서버 종료 중 오류 발생: %v", err)
	}

	logger.Println("서버가 정상적으로 종료되었습니다")
}
