package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	"go.uber.org/zap"
)

// MockWorkspaceVerificationService is a mock implementation
type MockWorkspaceVerificationService struct {
	mock.Mock
}

func (m *MockWorkspaceVerificationService) VerifyUserWorkspaceAccess(ctx context.Context, userID, workspaceID string) error {
	args := m.Called(ctx, userID, workspaceID)
	return args.Error(0)
}

func (m *MockWorkspaceVerificationService) GetUserWorkspaceRole(ctx context.Context, userID, workspaceID string) (string, error) {
	args := m.Called(ctx, userID, workspaceID)
	return args.String(0), args.Error(1)
}

func createValidJWT(userID, email, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})

	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

func createValidUUIDs() (userID, workspaceID string) {
	return "550e8400-e29b-41d4-a716-446655440000", "123e4567-e89b-12d3-a456-426614174000"
}

func TestJWTMiddleware_SuccessfulAuthentication(t *testing.T) {
	logger := zap.NewNop()
	mockService := new(MockWorkspaceVerificationService)
	
	userID, workspaceID := createValidUUIDs()
	
	// Mock successful workspace verification
	mockService.On("VerifyUserWorkspaceAccess", mock.Anything, userID, workspaceID).
		Return(nil)
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: mockService,
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	// Test handler that checks if user is authenticated
	handler := middleware(func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, workspaceID, user.UniversalID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "admin", user.Role)
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
	req.Header.Set("X-Workspace-Id", workspaceID)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	
	mockService.AssertExpectations(t)
}

func TestJWTMiddleware_MissingAuthorizationHeader(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil,
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err) // Middleware handles the error response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "MISSING_AUTH_HEADER")
}

func TestJWTMiddleware_MissingWorkspaceIdHeader_UsesUserIdAsFallback(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil,
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	handler := middleware(func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		assert.NoError(t, err)
		// When no X-Workspace-Id header, user_id should be used as UniversalID
		assert.Equal(t, user.UserID, user.UniversalID)
		
		// Test that workspace_id is empty string
		workspaceID, err := GetWorkspaceID(c)
		assert.NoError(t, err)
		assert.Equal(t, "", workspaceID)
		
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	userID, _ := createValidUUIDs()
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
	// No X-Workspace-Id header
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTMiddleware_InvalidUserIdFormat(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil,
		SkipPaths:                    []string{},
	}
	
	// Create JWT with invalid user ID
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "invalid-uuid",
		"email": "test@example.com",
		"role":  "admin",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})
	tokenString, _ := token.SignedString([]byte("test-secret"))
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	req.Header.Set("X-Workspace-Id", "workspace-456")
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "INVALID_USER_ID_FORMAT")
}

func TestJWTMiddleware_WorkspaceVerificationFailed(t *testing.T) {
	logger := zap.NewNop()
	mockService := new(MockWorkspaceVerificationService)
	
	userID, workspaceID := createValidUUIDs()
	
	// Mock workspace verification failure
	mockService.On("VerifyUserWorkspaceAccess", mock.Anything, userID, workspaceID).
		Return(domainErrors.NewUserNotMemberError(userID, workspaceID))
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: mockService,
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
	req.Header.Set("X-Workspace-Id", workspaceID)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "WORKSPACE_ACCESS_DENIED")
	
	mockService.AssertExpectations(t)
}

func TestJWTMiddleware_SkipPaths(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil,
		SkipPaths:                    []string{"/health", "/webhook"},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	handler := middleware(func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	// Test skipped path
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// No Authorization header - should still pass
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTMiddleware_DisabledWorkspaceVerification(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil, // Disabled
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	userID, workspaceID := createValidUUIDs()
	
	handler := middleware(func(c echo.Context) error {
		user, err := GetUserFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, userID, user.UserID)
		// With workspace header, UniversalID should be workspace_id
		assert.Equal(t, workspaceID, user.UniversalID)
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
	req.Header.Set("X-Workspace-Id", workspaceID)
	rec := httptest.NewRecorder()
	
	c := e.NewContext(req, rec)
	
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetUserID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	
	// Test with no user in context
	userID, err := GetUserID(c)
	assert.Error(t, err)
	assert.Empty(t, userID)
	
	// Test with user in context
	userID, workspaceID := createValidUUIDs()
	authUser := &AuthUser{
		UserID:      userID,
		UniversalID: workspaceID, // Using workspace_id as UniversalID in this test
		Email:       "test@example.com",
		Role:        "admin",
	}
	
	ctx := context.WithValue(c.Request().Context(), userContextKey, authUser)
	c.SetRequest(c.Request().WithContext(ctx))
	
	retrievedUserID, err := GetUserID(c)
	assert.NoError(t, err)
	assert.Equal(t, userID, retrievedUserID)
}

func TestUniversalIDPriorityLogic(t *testing.T) {
	logger := zap.NewNop()
	
	config := JWTConfig{
		Secret:                       "test-secret",
		Logger:                       logger,
		WorkspaceVerificationService: nil,
		SkipPaths:                    []string{},
	}
	
	e := echo.New()
	middleware := JWTMiddleware(config)
	
	userID, workspaceID := createValidUUIDs()
	
	t.Run("Priority 1: X-Workspace-Id header present", func(t *testing.T) {
		handler := middleware(func(c echo.Context) error {
			user, err := GetUserFromContext(c)
			assert.NoError(t, err)
			
			// UniversalID should be workspace_id (priority 1)
			assert.Equal(t, workspaceID, user.UniversalID)
			assert.Equal(t, userID, user.UserID)
			
			// Verify context values
			universalID, err := GetUniversalID(c)
			assert.NoError(t, err)
			assert.Equal(t, workspaceID, universalID)
			
			contextWorkspaceID, err := GetWorkspaceID(c)
			assert.NoError(t, err)
			assert.Equal(t, workspaceID, contextWorkspaceID)
			
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
		req.Header.Set("X-Workspace-Id", workspaceID)
		rec := httptest.NewRecorder()
		
		c := e.NewContext(req, rec)
		
		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
	
	t.Run("Priority 2: X-Workspace-Id header missing, fallback to user_id", func(t *testing.T) {
		handler := middleware(func(c echo.Context) error {
			user, err := GetUserFromContext(c)
			assert.NoError(t, err)
			
			// UniversalID should be user_id (priority 2 - fallback)
			assert.Equal(t, userID, user.UniversalID)
			assert.Equal(t, userID, user.UserID)
			
			// Verify context values
			universalID, err := GetUniversalID(c)
			assert.NoError(t, err)
			assert.Equal(t, userID, universalID)
			
			contextWorkspaceID, err := GetWorkspaceID(c)
			assert.NoError(t, err)
			assert.Equal(t, "", contextWorkspaceID) // Should be empty when header not provided
			
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
		
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+createValidJWT(userID, "test@example.com", "admin"))
		// No X-Workspace-Id header
		rec := httptest.NewRecorder()
		
		c := e.NewContext(req, rec)
		
		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}