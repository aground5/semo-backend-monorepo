package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// AuthUser represents an authenticated user from JWT
type AuthUser struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// contextKey is used for storing user in context
type contextKey string

const (
	userContextKey contextKey = "authenticated_user"
)

// JWTConfig holds the configuration for JWT middleware
type JWTConfig struct {
	Secret    string
	Logger    *zap.Logger
	SkipPaths []string // Paths to skip JWT validation
}

// JWTMiddleware creates a middleware that validates Supabase JWT tokens
func JWTMiddleware(config JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip JWT validation for certain paths
			path := c.Request().URL.Path
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(path, skipPath) {
					return next(c)
				}
			}

			// Extract token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				config.Logger.Warn("Missing authorization header",
					zap.String("path", path),
					zap.String("method", c.Request().Method))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Authorization header required",
					"code":  "MISSING_AUTH_HEADER",
				})
			}

			// Check Bearer prefix
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				config.Logger.Warn("Invalid authorization header format",
					zap.String("path", path))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Invalid authorization header format. Expected: Bearer <token>",
					"code":  "INVALID_AUTH_FORMAT",
				})
			}

			// Parse and validate JWT token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(config.Secret), nil
			})

			if err != nil {
				config.Logger.Warn("JWT validation failed",
					zap.Error(err),
					zap.String("path", path))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Invalid or expired token",
					"code":  "INVALID_TOKEN",
				})
			}

			// Extract claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// Extract user information from Supabase JWT claims
				userID, ok := claims["sub"].(string)
				if !ok || userID == "" {
					config.Logger.Warn("Missing user ID in JWT claims",
						zap.String("path", path))
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": "Invalid token claims",
						"code":  "INVALID_CLAIMS",
					})
				}

				email, _ := claims["email"].(string)
				role, _ := claims["role"].(string)

				// Create authenticated user
				authUser := &AuthUser{
					UserID: userID,
					Email:  email,
					Role:   role,
				}

				// Store user in request context
				ctx := context.WithValue(c.Request().Context(), userContextKey, authUser)
				c.SetRequest(c.Request().WithContext(ctx))

				c.Set("user_id", userID)

				config.Logger.Debug("User authenticated successfully",
					zap.String("user_id", userID),
					zap.String("email", email),
					zap.String("path", path))

				return next(c)
			}

			config.Logger.Warn("Invalid JWT claims",
				zap.String("path", path))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error": "Invalid token claims",
				"code":  "INVALID_CLAIMS",
			})
		}
	}
}

// GetUserFromContext extracts the authenticated user from the request context
func GetUserFromContext(c echo.Context) (*AuthUser, error) {
	user, ok := c.Request().Context().Value(userContextKey).(*AuthUser)
	if !ok || user == nil {
		return nil, fmt.Errorf("no authenticated user found in context")
	}
	return user, nil
}

// RequireAuth is a helper function to get user or return error response
func RequireAuth(c echo.Context) (*AuthUser, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return nil, c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "Authentication required",
			"code":  "AUTH_REQUIRED",
		})
	}
	return user, nil
}
