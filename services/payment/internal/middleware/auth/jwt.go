package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// WorkspaceVerificationService defines the interface for workspace verification
type WorkspaceVerificationService interface {
	VerifyUserWorkspaceAccess(ctx context.Context, userID, workspaceID string) error
}

// AuthUser represents an authenticated user from JWT
type AuthUser struct {
	UserID      string `json:"user_id"`     // User ID from JWT sub claim
	UniversalID string `json:"universal_id"` // Now stores workspace_id as universal_id
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// contextKey is used for storing user in context
type contextKey string

const (
	userContextKey contextKey = "authenticated_user"
)

// JWTConfig holds the configuration for JWT middleware
type JWTConfig struct {
	Secret                       string
	Logger                       *zap.Logger
	SkipPaths                    []string                        // Paths to skip JWT validation
	WorkspaceVerificationService WorkspaceVerificationService   // Optional workspace verification service
}

// JWTMiddleware creates a middleware that validates Supabase JWT tokens
func JWTMiddleware(config JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Start timing the entire authentication process
			startTime := time.Now()
			path := c.Request().URL.Path
			method := c.Request().Method
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				// Generate a request ID if not provided
				requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
			}

			config.Logger.Debug("JWT middleware: Starting authentication process",
				zap.String("request_id", requestID),
				zap.String("path", path),
				zap.String("method", method),
				zap.String("step", "start"))

			// Skip JWT validation for certain paths
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(path, skipPath) {
					config.Logger.Debug("JWT middleware: Skipping authentication for path",
						zap.String("request_id", requestID),
						zap.String("path", path),
						zap.String("skip_path", skipPath),
						zap.String("step", "skip_auth"))
					return next(c)
				}
			}

			// Step 1: Extract token from Authorization header
			config.Logger.Debug("JWT middleware: Step 1 - Extracting authorization header",
				zap.String("request_id", requestID),
				zap.String("step", "extract_auth_header"))

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				config.Logger.Warn("JWT middleware: Missing authorization header",
					zap.String("request_id", requestID),
					zap.String("path", path),
					zap.String("method", method),
					zap.String("step", "extract_auth_header"),
					zap.String("status", "failed"))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Authorization header required",
					"code":  "MISSING_AUTH_HEADER",
				})
			}

			config.Logger.Debug("JWT middleware: Authorization header found",
				zap.String("request_id", requestID),
				zap.String("step", "extract_auth_header"),
				zap.String("status", "success"))

			// Step 2: Validate Bearer prefix and extract token
			config.Logger.Debug("JWT middleware: Step 2 - Validating Bearer prefix",
				zap.String("request_id", requestID),
				zap.String("step", "validate_bearer_format"))

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				config.Logger.Warn("JWT middleware: Invalid authorization header format",
					zap.String("request_id", requestID),
					zap.String("path", path),
					zap.String("step", "validate_bearer_format"),
					zap.String("status", "failed"))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Invalid authorization header format. Expected: Bearer <token>",
					"code":  "INVALID_AUTH_FORMAT",
				})
			}

			config.Logger.Debug("JWT middleware: Bearer token extracted successfully",
				zap.String("request_id", requestID),
				zap.String("step", "validate_bearer_format"),
				zap.String("status", "success"),
				zap.Int("token_length", len(tokenString)))

			// Step 3: Parse and validate JWT token
			config.Logger.Debug("JWT middleware: Step 3 - Parsing and validating JWT token",
				zap.String("request_id", requestID),
				zap.String("step", "parse_jwt_token"))

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(config.Secret), nil
			})

			if err != nil {
				config.Logger.Warn("JWT middleware: JWT token validation failed",
					zap.String("request_id", requestID),
					zap.Error(err),
					zap.String("path", path),
					zap.String("step", "parse_jwt_token"),
					zap.String("status", "failed"))
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "Invalid or expired token",
					"code":  "INVALID_TOKEN",
				})
			}

			config.Logger.Debug("JWT middleware: JWT token parsed successfully",
				zap.String("request_id", requestID),
				zap.String("step", "parse_jwt_token"),
				zap.String("status", "success"))

			// Step 4: Extract and validate JWT claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				config.Logger.Debug("JWT middleware: Step 4 - Extracting JWT claims",
					zap.String("request_id", requestID),
					zap.String("step", "extract_jwt_claims"),
					zap.String("status", "success"))

				// Extract user_id from JWT sub claim
				userID, ok := claims["sub"].(string)
				if !ok || userID == "" {
					config.Logger.Warn("JWT middleware: Missing or invalid user_id (sub) in JWT claims",
						zap.String("request_id", requestID),
						zap.String("path", path),
						zap.String("step", "extract_user_id"),
						zap.String("status", "failed"))
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": "Invalid token: missing user_id",
						"code":  "MISSING_USER_ID",
					})
				}

				config.Logger.Debug("JWT middleware: User ID extracted from token",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("step", "extract_user_id"),
					zap.String("status", "success"))

				// Step 5: Validate user_id UUID format
				config.Logger.Debug("JWT middleware: Step 5 - Validating user_id UUID format",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("step", "validate_user_id_format"))

				if _, err := uuid.Parse(userID); err != nil {
					config.Logger.Warn("JWT middleware: Invalid user_id format in JWT",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("path", path),
						zap.String("step", "validate_user_id_format"),
						zap.String("status", "failed"),
						zap.Error(err))
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": "Invalid token: user_id must be a valid UUID",
						"code":  "INVALID_USER_ID_FORMAT",
					})
				}

				config.Logger.Debug("JWT middleware: User ID format validation successful",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("step", "validate_user_id_format"),
					zap.String("status", "success"))

				// Step 6: Extract workspace_id from X-Workspace-Id header
				config.Logger.Debug("JWT middleware: Step 6 - Extracting workspace_id from header",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("step", "extract_workspace_id"))

				workspaceID := c.Request().Header.Get("X-Workspace-Id")
				if workspaceID == "" {
					config.Logger.Warn("JWT middleware: Missing X-Workspace-Id header",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("path", path),
						zap.String("step", "extract_workspace_id"),
						zap.String("status", "failed"))
					return c.JSON(http.StatusBadRequest, echo.Map{
						"error": "X-Workspace-Id header required",
						"code":  "MISSING_WORKSPACE_ID",
					})
				}

				config.Logger.Debug("JWT middleware: Workspace ID extracted from header",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("step", "extract_workspace_id"),
					zap.String("status", "success"))

				// Step 7: Validate workspace_id UUID format
				config.Logger.Debug("JWT middleware: Step 7 - Validating workspace_id UUID format",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("step", "validate_workspace_id_format"))

				if _, err := uuid.Parse(workspaceID); err != nil {
					config.Logger.Warn("JWT middleware: Invalid workspace_id format",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("workspace_id", workspaceID),
						zap.String("path", path),
						zap.String("step", "validate_workspace_id_format"),
						zap.String("status", "failed"),
						zap.Error(err))
					return c.JSON(http.StatusBadRequest, echo.Map{
						"error": "X-Workspace-Id must be a valid UUID format",
						"code":  "INVALID_WORKSPACE_ID_FORMAT",
					})
				}

				config.Logger.Debug("JWT middleware: Workspace ID format validation successful",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("step", "validate_workspace_id_format"),
					zap.String("status", "success"))

				// Step 8: Extract optional fields from JWT claims
				config.Logger.Debug("JWT middleware: Step 8 - Extracting optional JWT claims",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("step", "extract_optional_claims"))

				email, _ := claims["email"].(string)
				role, _ := claims["role"].(string)

				config.Logger.Debug("JWT middleware: Optional claims extracted",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("email", email),
					zap.String("role", role),
					zap.String("step", "extract_optional_claims"),
					zap.String("status", "success"))

				// Step 9: Verify workspace membership if workspace verification service is provided
				if config.WorkspaceVerificationService != nil {
					config.Logger.Debug("JWT middleware: Step 9 - Starting workspace verification",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("workspace_id", workspaceID),
						zap.String("step", "workspace_verification"))

					verificationStart := time.Now()
					if err := config.WorkspaceVerificationService.VerifyUserWorkspaceAccess(
						c.Request().Context(),
						userID,
						workspaceID,
					); err != nil {
						verificationDuration := time.Since(verificationStart)
						config.Logger.Warn("JWT middleware: Workspace access verification failed",
							zap.String("request_id", requestID),
							zap.String("user_id", userID),
							zap.String("workspace_id", workspaceID),
							zap.String("path", path),
							zap.String("step", "workspace_verification"),
							zap.String("status", "failed"),
							zap.Duration("verification_duration", verificationDuration),
							zap.Error(err))
						return c.JSON(http.StatusForbidden, echo.Map{
							"error": "Access denied: user is not authorized for this workspace",
							"code":  "WORKSPACE_ACCESS_DENIED",
						})
					}

					verificationDuration := time.Since(verificationStart)
					config.Logger.Info("JWT middleware: Workspace access verification successful",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("workspace_id", workspaceID),
						zap.String("path", path),
						zap.String("step", "workspace_verification"),
						zap.String("status", "success"),
						zap.Duration("verification_duration", verificationDuration))
				} else {
					// Log warning if workspace verification is disabled
					config.Logger.Warn("JWT middleware: Workspace verification is disabled - security risk",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("workspace_id", workspaceID),
						zap.String("path", path),
						zap.String("step", "workspace_verification"),
						zap.String("status", "skipped"))
				}

				// Step 10: Create authenticated user and set context
				config.Logger.Debug("JWT middleware: Step 10 - Creating authenticated user context",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("step", "create_auth_context"))

				// Create authenticated user with workspace_id as universal_id
				authUser := &AuthUser{
					UserID:      userID,      // User ID from JWT sub claim
					UniversalID: workspaceID, // Using workspace_id as universal_id
					Email:       email,
					Role:        role,
				}

				// Store user in request context
				ctx := context.WithValue(c.Request().Context(), userContextKey, authUser)
				c.SetRequest(c.Request().WithContext(ctx))

				// Set universal_id in echo context (actually workspace_id)
				c.Set("universal_id", workspaceID)
				c.Set("workspace_id", workspaceID) // Also set as workspace_id for clarity
				c.Set("request_id", requestID)       // Store request ID for downstream logging

				totalDuration := time.Since(startTime)
				config.Logger.Info("JWT middleware: Authentication completed successfully",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("workspace_id", workspaceID),
					zap.String("email", email),
					zap.String("role", role),
					zap.String("path", path),
					zap.String("method", method),
					zap.String("step", "authentication_complete"),
					zap.String("status", "success"),
					zap.Duration("total_auth_duration", totalDuration))

				return next(c)
			}

			config.Logger.Warn("JWT middleware: Invalid JWT claims",
				zap.String("request_id", requestID),
				zap.String("path", path),
				zap.String("step", "extract_jwt_claims"),
				zap.String("status", "failed"))
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

// GetUniversalID is a helper function to get universal_id from context
func GetUniversalID(c echo.Context) (string, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return "", err
	}
	return user.UniversalID, nil // UniversalID now contains workspace_id
}

// GetWorkspaceID is a helper function to get workspace_id from context (alias for GetUniversalID)
func GetWorkspaceID(c echo.Context) (string, error) {
	return GetUniversalID(c)
}

// GetUserID is a helper function to get user_id from context
func GetUserID(c echo.Context) (string, error) {
	user, err := GetUserFromContext(c)
	if err != nil {
		return "", err
	}
	return user.UserID, nil
}