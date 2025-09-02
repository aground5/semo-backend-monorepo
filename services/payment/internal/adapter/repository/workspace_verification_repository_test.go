package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

func TestSupabaseWorkspaceVerificationRepository_VerifyWorkspaceMembership(t *testing.T) {
	logger := zap.NewNop()
	
	tests := []struct {
		name               string
		userID             string
		workspaceID        string
		mockServerResponse func(w http.ResponseWriter, r *http.Request)
		expectedMember     *domainRepo.WorkspaceMember
		expectedError      bool
		expectedErrorType  string
	}{
		{
			name:        "successful verification",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				// Verify request parameters
				assert.Equal(t, "/rest/v1/workspace_members", r.URL.Path)
				assert.Equal(t, "eq.user-123", r.URL.Query().Get("user_id"))
				assert.Equal(t, "eq.workspace-456", r.URL.Query().Get("workspace_id"))
				assert.Equal(t, "*", r.URL.Query().Get("select"))
				
				// Verify headers
				assert.Equal(t, "test-api-key", r.Header.Get("apikey"))
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				
				members := []domainRepo.WorkspaceMember{
					{
						ID:          "member-789",
						WorkspaceID: "workspace-456",
						UserID:      "user-123",
						Role:        "admin",
						JoinedAt:    "2024-01-01T00:00:00Z",
						InvitedBy:   "user-000",
						Description: "Test member",
					},
				}
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(members)
			},
			expectedMember: &domainRepo.WorkspaceMember{
				ID:          "member-789",
				WorkspaceID: "workspace-456",
				UserID:      "user-123",
				Role:        "admin",
				JoinedAt:    "2024-01-01T00:00:00Z",
				InvitedBy:   "user-000",
				Description: "Test member",
			},
			expectedError: false,
		},
		{
			name:        "user not member - empty response",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]domainRepo.WorkspaceMember{})
			},
			expectedMember:    nil,
			expectedError:     true,
			expectedErrorType: domainErrors.ErrTypeUserNotMember,
		},
		{
			name:        "supabase API unauthorized",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "unauthorized"}`))
			},
			expectedMember:    nil,
			expectedError:     true,
			expectedErrorType: domainErrors.ErrTypeSupabaseConnectionFailed,
		},
		{
			name:        "workspace not found",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "table not found"}`))
			},
			expectedMember:    nil,
			expectedError:     true,
			expectedErrorType: domainErrors.ErrTypeWorkspaceNotFound,
		},
		{
			name:        "supabase API server error",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			expectedMember:    nil,
			expectedError:     true,
			expectedErrorType: domainErrors.ErrTypeSupabaseConnectionFailed,
		},
		{
			name:        "invalid JSON response",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockServerResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{invalid json`))
			},
			expectedMember:    nil,
			expectedError:     true,
			expectedErrorType: domainErrors.ErrTypeSupabaseConnectionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(tt.mockServerResponse))
			defer server.Close()
			
			// Create repository with mock server URL
			repo := NewSupabaseWorkspaceVerificationRepository(
				server.URL,
				"test-api-key",
				"test-jwt-secret",
				logger,
			)
			
			// Execute
			member, err := repo.VerifyWorkspaceMembership(context.Background(), tt.userID, tt.workspaceID)
			
			// Verify
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, member)
				
				if tt.expectedErrorType != "" {
					var workspaceErr *domainErrors.WorkspaceError
					if assert.ErrorAs(t, err, &workspaceErr) {
						assert.Equal(t, tt.expectedErrorType, workspaceErr.Type)
						assert.Equal(t, tt.userID, workspaceErr.UserID)
						assert.Equal(t, tt.workspaceID, workspaceErr.WorkspaceID)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, member)
				assert.Equal(t, tt.expectedMember.ID, member.ID)
				assert.Equal(t, tt.expectedMember.WorkspaceID, member.WorkspaceID)
				assert.Equal(t, tt.expectedMember.UserID, member.UserID)
				assert.Equal(t, tt.expectedMember.Role, member.Role)
				assert.Equal(t, tt.expectedMember.JoinedAt, member.JoinedAt)
				assert.Equal(t, tt.expectedMember.InvitedBy, member.InvitedBy)
				assert.Equal(t, tt.expectedMember.Description, member.Description)
			}
		})
	}
}

func TestSupabaseWorkspaceVerificationRepository_RequestTimeout(t *testing.T) {
	logger := zap.NewNop()
	
	// Setup a server that never responds to simulate timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond, causing a timeout
		select {}
	}))
	defer server.Close()
	
	repo := &SupabaseWorkspaceVerificationRepository{
		client:    &http.Client{Timeout: 1}, // 1 nanosecond timeout
		baseURL:   server.URL,
		apiKey:    "test-api-key",
		jwtSecret: "test-jwt-secret",
		logger:    logger,
	}
	
	member, err := repo.VerifyWorkspaceMembership(context.Background(), "user-123", "workspace-456")
	
	assert.Error(t, err)
	assert.Nil(t, member)
	
	var workspaceErr *domainErrors.WorkspaceError
	if assert.ErrorAs(t, err, &workspaceErr) {
		assert.Equal(t, domainErrors.ErrTypeSupabaseConnectionFailed, workspaceErr.Type)
	}
}

func TestSupabaseWorkspaceVerificationRepository_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	
	// Setup a server that takes time to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()
	
	repo := NewSupabaseWorkspaceVerificationRepository(
		server.URL,
		"test-api-key",
		"test-jwt-secret",
		logger,
	)
	
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	member, err := repo.VerifyWorkspaceMembership(ctx, "user-123", "workspace-456")
	
	assert.Error(t, err)
	assert.Nil(t, member)
	
	var workspaceErr *domainErrors.WorkspaceError
	if assert.ErrorAs(t, err, &workspaceErr) {
		assert.Equal(t, domainErrors.ErrTypeSupabaseConnectionFailed, workspaceErr.Type)
	}
}