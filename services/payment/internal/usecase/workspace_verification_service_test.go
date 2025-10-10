package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// MockWorkspaceVerificationRepository is a mock implementation of WorkspaceVerificationRepository
type MockWorkspaceVerificationRepository struct {
	mock.Mock
}

func (m *MockWorkspaceVerificationRepository) VerifyWorkspaceMembership(ctx context.Context, userID, workspaceID string) (*domainRepo.WorkspaceMember, error) {
	args := m.Called(ctx, userID, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainRepo.WorkspaceMember), args.Error(1)
}

func TestWorkspaceVerificationService_VerifyUserWorkspaceAccess(t *testing.T) {
	logger := zap.NewNop()
	
	tests := []struct {
		name           string
		userID         string
		workspaceID    string
		mockSetup      func(*MockWorkspaceVerificationRepository)
		expectedError  bool
		errorType      string
	}{
		{
			name:        "successful verification",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(&domainRepo.WorkspaceMember{
						ID:          "member-789",
						WorkspaceID: "workspace-456",
						UserID:      "user-123",
						Role:        "admin",
						JoinedAt:    "2024-01-01T00:00:00Z",
						InvitedBy:   "user-000",
						Description: "Test member",
					}, nil)
			},
			expectedError: false,
		},
		{
			name:        "user not member error",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(nil, domainErrors.NewUserNotMemberError("user-123", "workspace-456"))
			},
			expectedError: true,
			errorType:     domainErrors.ErrTypeUserNotMember,
		},
		{
			name:        "workspace not found error",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(nil, domainErrors.NewWorkspaceNotFoundError("user-123", "workspace-456"))
			},
			expectedError: true,
			errorType:     domainErrors.ErrTypeWorkspaceNotFound,
		},
		{
			name:        "supabase connection error",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(nil, domainErrors.NewSupabaseConnectionError("user-123", "workspace-456", 
						errors.New("network timeout")))
			},
			expectedError: true,
			errorType:     domainErrors.ErrTypeSupabaseConnectionFailed,
		},
		{
			name:        "user with empty role",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(&domainRepo.WorkspaceMember{
						ID:          "member-789",
						WorkspaceID: "workspace-456",
						UserID:      "user-123",
						Role:        "", // Empty role should trigger insufficient permissions
						JoinedAt:    "2024-01-01T00:00:00Z",
						InvitedBy:   "user-000",
						Description: "Test member",
					}, nil)
			},
			expectedError: true,
			errorType:     domainErrors.ErrTypeInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockWorkspaceVerificationRepository)
			tt.mockSetup(mockRepo)
			
			service := NewWorkspaceVerificationService(mockRepo, logger)
			
			// Execute
			err := service.VerifyUserWorkspaceAccess(context.Background(), tt.userID, tt.workspaceID)
			
			// Verify
			if tt.expectedError {
				assert.Error(t, err)
				
				if tt.errorType != "" {
					var workspaceErr *domainErrors.WorkspaceError
					if assert.ErrorAs(t, err, &workspaceErr) {
						assert.Equal(t, tt.errorType, workspaceErr.Type)
						assert.Equal(t, tt.userID, workspaceErr.UserID)
						assert.Equal(t, tt.workspaceID, workspaceErr.WorkspaceID)
					}
				}
			} else {
				assert.NoError(t, err)
			}
			
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestWorkspaceVerificationService_GetUserWorkspaceRole(t *testing.T) {
	logger := zap.NewNop()
	
	tests := []struct {
		name         string
		userID       string
		workspaceID  string
		mockSetup    func(*MockWorkspaceVerificationRepository)
		expectedRole string
		expectedError bool
	}{
		{
			name:        "successful role retrieval",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(&domainRepo.WorkspaceMember{
						ID:          "member-789",
						WorkspaceID: "workspace-456",
						UserID:      "user-123",
						Role:        "admin",
						JoinedAt:    "2024-01-01T00:00:00Z",
						InvitedBy:   "user-000",
						Description: "Test member",
					}, nil)
			},
			expectedRole:  "admin",
			expectedError: false,
		},
		{
			name:        "user not member",
			userID:      "user-123",
			workspaceID: "workspace-456",
			mockSetup: func(repo *MockWorkspaceVerificationRepository) {
				repo.On("VerifyWorkspaceMembership", mock.Anything, "user-123", "workspace-456").
					Return(nil, domainErrors.NewUserNotMemberError("user-123", "workspace-456"))
			},
			expectedRole:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockWorkspaceVerificationRepository)
			tt.mockSetup(mockRepo)
			
			service := NewWorkspaceVerificationService(mockRepo, logger)
			
			// Execute
			role, err := service.GetUserWorkspaceRole(context.Background(), tt.userID, tt.workspaceID)
			
			// Verify
			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, role)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
			
			mockRepo.AssertExpectations(t)
		})
	}
}