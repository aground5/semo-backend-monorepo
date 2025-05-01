package httpEngine

import (
	"authn-server/internal/controllers"
	"authn-server/internal/middlewares"
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4/middleware"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all the server routes
func RegisterRoutes(e *echo.Echo) {
	// Basic health check
	e.GET("/", func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.String(http.StatusOK, "Hello, from Authn Server!")
	})

	// Configure rate limiters for sensitive endpoints
	loginLimiter := middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      5,             // 5 requests
				Burst:     10,            // Burst of 10 requests
				ExpiresIn: 1 * time.Hour, // Per 1 hour
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			// Rate limiting based on IP and email (if available)
			email := ""
			if req := ctx.Request(); req.Method == "POST" {
				email = ctx.FormValue("email")
			}
			id := ctx.RealIP()
			if email != "" {
				id += ":" + email
			}
			return id, nil
		},
		ErrorHandler: func(ctx echo.Context, err error) error {
			return ctx.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests. Please try again later.",
			})
		},
		DenyHandler: func(ctx echo.Context, identifier string, err error) error {
			return ctx.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests. Please try again later.",
			})
		},
	}

	// Authentication endpoints
	authGroup := e.Group("/authn")
	authGroup.Use(middlewares.SessionMiddleware)
	{
		authGroup.GET("/me", controllers.MeHandler)
		// Magic code authentication
		authGroup.POST("/magic", controllers.MagicHandler, middleware.RateLimiterWithConfig(loginLimiter))
		authGroup.POST("/login", controllers.LoginHandler, middleware.RateLimiterWithConfig(loginLimiter))
		authGroup.POST("/logout", controllers.LogoutHandler)

		// Token management
		authGroup.POST("/auto_login", controllers.AutoLoginHandler, middleware.RateLimiterWithConfig(loginLimiter))
		authGroup.POST("/refresh", controllers.RefreshAccessTokenHandler)

		// Two-factor authentication specific to login flow
		//authGroup.POST("/two_factor_login", controllers.TwoFactorLoginHandler, middleware.RateLimiterWithConfig(loginLimiter))
		authGroup.POST("/generate-tokens-after-2fa", controllers.GenerateTokensAfter2FAHandler)

		// User registration and email verification
		authGroup.POST("/register", controllers.RegisterHandler, middleware.RateLimiterWithConfig(loginLimiter))
		authGroup.POST("/verify-email", controllers.VerifyEmailHandler)
		authGroup.GET("/verify-email", controllers.VerifyEmailHandler) // Also support GET for email links
		authGroup.POST("/resend-verification", controllers.ResendVerificationHandler, middleware.RateLimiterWithConfig(loginLimiter))

		// Password-based authentication
		authGroup.POST("/login-password", controllers.PasswordLoginHandler, middleware.RateLimiterWithConfig(loginLimiter))
	}

	// CAPTCHA endpoints
	captchaGroup := e.Group("/captcha")
	{
		captchaGroup.POST("/generate", controllers.GenerateCaptchaHandler)
		captchaGroup.POST("/verify", controllers.VerifyCaptchaHandler)
		captchaGroup.GET("/required", controllers.CheckCaptchaRequiredHandler)
	}

	// Two-factor authentication management endpoints
	twoFactorGroup := e.Group("/two-factor")
	twoFactorGroup.Use(middlewares.JWTMiddleware)
	{
		twoFactorGroup.POST("/setup", controllers.SetupTwoFactorHandler)
		twoFactorGroup.POST("/verify", controllers.VerifyAndEnableTwoFactorHandler)
		twoFactorGroup.POST("/disable", controllers.DisableTwoFactorHandler)
		twoFactorGroup.GET("/status", controllers.CheckTwoFactorEnabledHandler)
		twoFactorGroup.POST("/challenge", controllers.CreateTwoFactorChallengeHandler)
		twoFactorGroup.POST("/complete-challenge", controllers.CompleteTwoFactorChallengeHandler)
	}

	// Trusted device management endpoints
	trustedDeviceGroup := e.Group("/trusted-devices")
	trustedDeviceGroup.Use(middlewares.JWTMiddleware)
	{
		trustedDeviceGroup.GET("", controllers.GetTrustedDevicesHandler)
		trustedDeviceGroup.POST("", controllers.AddTrustedDeviceHandler)
		trustedDeviceGroup.DELETE("", controllers.RemoveTrustedDeviceHandler)
		trustedDeviceGroup.GET("/check", controllers.CheckDeviceTrustedHandler)
		trustedDeviceGroup.GET("/alerts", controllers.GetUnknownDeviceAlertsHandler)
		trustedDeviceGroup.POST("/confirm", controllers.ConfirmDeviceHandler)
		trustedDeviceGroup.POST("/detect", controllers.DetectUnknownDeviceHandler)
	}

	// Notification management endpoints
	notificationGroup := e.Group("/notifications")
	notificationGroup.Use(middlewares.JWTMiddleware)
	{
		notificationGroup.GET("", controllers.GetNotificationsHandler)
		notificationGroup.POST("/mark-read", controllers.MarkNotificationAsReadHandler)
		notificationGroup.POST("/mark-all-read", controllers.MarkAllNotificationsAsReadHandler)
		notificationGroup.GET("/count", controllers.GetUnreadNotificationCountHandler)
		notificationGroup.GET("/preferences", controllers.GetNotificationPreferencesHandler)
		notificationGroup.PUT("/preferences", controllers.UpdateNotificationPreferencesHandler)
	}

	// Admin API endpoints (requires both JWT and admin role)
	adminGroup := e.Group("/admin")
	adminGroup.Use(middlewares.JWTMiddleware, middlewares.AdminMiddleware)
	{
		adminGroup.GET("/blocked-ips", controllers.ListBlockedIPsHandler)
		adminGroup.POST("/block-ip", controllers.BlockIPHandler)
		adminGroup.POST("/unblock-ip", controllers.UnblockIPHandler)
	}

	// Remote authentication management endpoints
	remoteAuthGroup := e.Group("/remote-auth")
	remoteAuthGroup.Use(middlewares.JWTMiddleware)
	{
		remoteAuthGroup.GET("/activities", controllers.ListActiveActivitiesHandler)
		remoteAuthGroup.POST("/deactivate", controllers.DeactivateActivityHandler)
	}

	// Initial setup endpoints
	initialGroup := e.Group("/initial")
	initialGroup.Use(middlewares.JWTMiddleware)

	initialGroup.GET("/status", controllers.GetInitialStatusHandler)
	initialGroup.GET("/user", controllers.GetUserNameHandler)
	initialGroup.PUT("/user", controllers.UpdateUserNameHandler)
}
