package httpEngine

import (
	"authn-server/configs"
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/boj/redistore"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type Server struct {
	e *echo.Echo
}

// NewServer instantiates Echo, initializes session store, and registers routes
func NewServer() *Server {
	e := echo.New()
	e.IPExtractor = echo.ExtractIPFromRealIPHeader()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://semo.world", "https://www.semo.world", "https://app.semo.world"}, // 특정 출처만 허용
		AllowCredentials: true,                                                                               // 쿠키 전송 허용
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "x-duid"},
	}))

	requestLoggerConfig := initCustomRequestLoggerConfig()
	e.Use(middleware.RequestLoggerWithConfig(*requestLoggerConfig))

	store, err := initSessionStore()
	if err != nil {
		// 에러 발생 시 nil 반환 (실제 서비스에서는 적절한 에러 핸들링 필요)
		return nil
	}
	e.Use(session.Middleware(store))

	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(5)))
	e.Use(middleware.Recover())

	RegisterRoutes(e)

	return &Server{e: e}
}

// Start runs the Echo server on the configured HTTP port.
// 기존에는 Logger.Fatal로 바로 종료했으나, graceful shutdown을 위해 error를 반환합니다.
func (s *Server) Start() error {
	port := configs.Configs.Service.HttpPort
	if port == "" {
		port = "8080"
	}
	return s.e.Start(":" + port)
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.e.Shutdown(ctx)
}

// initSessionStore는 Redis 기반 세션 스토어를 초기화합니다.
func initSessionStore() (store *redistore.RediStore, err error) {
	if len(configs.Configs.Redis.Addresses) == 0 {
		configs.Logger.Fatal("No Redis addresses configured for session store")
	}

	redisAddress := configs.Configs.Redis.Addresses[0]
	secretKey := []byte(configs.Configs.Secrets.SessionSecret)

	// redigo 기반의 redis 연결 풀 생성
	pool := &redis.Pool{
		MaxIdle:     10,
		MaxActive:   0, // 0이면 제한 없음
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// Dial 옵션들을 구성합니다.
			var options []redis.DialOption

			// Username이 설정되어 있다면 Dial 옵션에 추가 (Redis ACL 지원)
			if configs.Configs.Redis.Username != "" {
				options = append(options, redis.DialUsername(configs.Configs.Redis.Username))
			}
			// 비밀번호가 설정되어 있다면 Dial 옵션에 추가
			if configs.Configs.Redis.Password != "" {
				options = append(options, redis.DialPassword(configs.Configs.Redis.Password))
			}
			// TLS 사용 여부에 따라 TLS 옵션 추가
			if configs.Configs.Redis.Tls {
				options = append(options,
					redis.DialUseTLS(true),
					redis.DialTLSConfig(&tls.Config{
						// 필요한 경우 인증서 검증 관련 옵션 설정 (예: InsecureSkipVerify)
						InsecureSkipVerify: true,
					}),
				)
			}
			return redis.Dial("tcp", redisAddress, options...)
		},
	}

	store, err = redistore.NewRediStoreWithPool(pool, secretKey)
	if err != nil {
		configs.Logger.Fatal("Failed to create Redis-based session store", zap.Error(err))
		return nil, err
	}

	store.SetKeyPrefix("session:")

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   configs.Configs.Authn.SessionExpireMin * 60,
		HttpOnly: true,
		Secure:   true,
	}

	configs.Logger.Info("Session store initialized",
		zap.String("redisAddress", redisAddress),
		zap.Int("sessionExpireMin", configs.Configs.Authn.SessionExpireMin),
		zap.Bool("httpOnly", true),
		zap.Bool("secure", true),
	)

	return store, nil
}

func initCustomRequestLoggerConfig() *middleware.RequestLoggerConfig {
	return &middleware.RequestLoggerConfig{
		// 특정 경로(예: /)를 로그에서 제외하고 싶다면 Skipper 사용
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/"
		},
		// 다음 미들웨어나 핸들러가 실행되기 전 실행되는 함수
		BeforeNextFunc: func(c echo.Context) {
			// 예: request 시작 시간이나 custom 값을 context에 담을 수 있음
			c.Set("request-start-time", time.Now())
		},
		// 에러도 글로벌 핸들러에게 넘길지 여부 (원하는 동작에 맞추어 선택)
		HandleError: true,

		// 로그로 남길 항목 설정 (필요한 것들을 true로)
		LogLatency:       true, // 핸들러 체인을 실행한 뒤 소요된 시간 기록
		LogProtocol:      true, // HTTP/1.1, HTTP/2 등 프로토콜 정보
		LogRemoteIP:      true, // 클라이언트 IP (echo.Context.RealIP() 기준)
		LogHost:          true, // Host 정보 (예: example.com)
		LogMethod:        true, // HTTP 메서드 (GET, POST 등)
		LogURI:           true, // 요청 URI (/users?lang=en 등)
		LogURIPath:       true, // 요청 Path 부분 (/users 등)
		LogRoutePath:     true, // echo 라우팅 경로 (/users/:id 등)
		LogRequestID:     true, // X-Request-ID 헤더 또는 자동 생성된 Request ID
		LogReferer:       true, // Referer 정보
		LogUserAgent:     true, // User-Agent 정보
		LogStatus:        true, // 응답 상태 코드
		LogError:         true, // next(...)에서 발생한 에러
		LogContentLength: true, // 요청 헤더의 Content-Length
		LogResponseSize:  true, // 실제 응답의 바이트 수

		// 특정 Header/Query Param/Form Value 등을 추가로 기록하고 싶다면
		// 필요한 이름들을 아래와 같이 배열에 추가
		LogHeaders:     []string{"Content-Type", "Accept-Encoding"},
		LogQueryParams: []string{"lang", "page"},
		LogFormValues:  []string{"username", "email"},

		// 중요: 요청/응답 정보를 실제로 어떻게 로깅할지 결정하는 함수
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			// 예: BeforeNextFunc에서 설정한 정보 가져오기
			startTime, _ := c.Get("request-start-time").(time.Time)
			elapsed := time.Since(startTime).String()

			// 로그 필드를 zap.Field 형태로 구성
			fields := []zap.Field{
				zap.String("request.remote_ip", v.RemoteIP),
				zap.String("request.host", v.Host),
				zap.String("request.protocol", v.Protocol),
				zap.String("request.method", v.Method),
				zap.String("request.uri", v.URI),
				zap.String("request.path", v.URIPath),
				zap.String("request.route", v.RoutePath),
				zap.String("request.user_agent", v.UserAgent),
				zap.String("request.referer", v.Referer),
				zap.Int("response.status", v.Status),
				zap.Duration("response.latency", v.Latency),
				zap.String("response.latency_human", v.Latency.String()),
				zap.String("response.elapsed_since_before_next", elapsed), // BeforeNextFunc 기준 시간
				zap.String("request.request_id", v.RequestID),
				zap.Int64("response.response_size", v.ResponseSize),
				zap.String("request.content_length", v.ContentLength),
			}

			// Header, QueryParam, FormValue 같은 slice 형태 데이터들을 로깅하는 예시
			// (원하는 경우 JSON 형태로 필드를 더 추가할 수도 있습니다)
			if len(v.Headers) > 0 {
				fields = append(fields, zap.Any("request.headers", v.Headers))
			}
			if len(v.QueryParams) > 0 {
				fields = append(fields, zap.Any("request.query_params", v.QueryParams))
			}
			if len(v.FormValues) > 0 {
				fields = append(fields, zap.Any("request.form_values", v.FormValues))
			}

			// 에러가 있는 경우
			if v.Error != nil {
				fields = append(fields, zap.Error(v.Error))
				// 에러이므로 Warn/Error 레벨로 로그를 찍을 수도 있습니다.
				configs.Logger.Error("Request log with error", fields...)
				return nil
			}

			// 정상 응답의 경우 Info 레벨로 기록
			configs.Logger.Info("Request log", fields...)
			return nil
		},
	}
}
