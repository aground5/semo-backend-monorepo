package usecase

import (
	"context"
	"fmt"
	"time"

	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// WorkspaceVerificationService handles workspace membership verification
type WorkspaceVerificationService struct {
	workspaceRepo domainRepo.WorkspaceVerificationRepository
	logger        *zap.Logger
}

// NewWorkspaceVerificationService creates a new workspace verification service
func NewWorkspaceVerificationService(
	workspaceRepo domainRepo.WorkspaceVerificationRepository,
	logger *zap.Logger,
) *WorkspaceVerificationService {
	return &WorkspaceVerificationService{
		workspaceRepo: workspaceRepo,
		logger:        logger,
	}
}

// VerifyUserWorkspaceAccess checks if a user has access to a specific workspace
func (s *WorkspaceVerificationService) VerifyUserWorkspaceAccess(
	ctx context.Context,
	userID,
	workspaceID string,
) error {
	// Start timing the verification process
	startTime := time.Now()
	requestID := ""
	if ctx.Value("request_id") != nil {
		requestID = ctx.Value("request_id").(string)
	}
	if requestID == "" {
		requestID = fmt.Sprintf("wvs_%d", time.Now().UnixNano())
	}

	s.logger.Info("WorkspaceVerificationService: Starting workspace access verification",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "service_entry"),
		zap.String("status", "started"))

	// Step 1: Input validation
	s.logger.Debug("WorkspaceVerificationService: Step 1 - Validating input parameters",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "input_validation"))

	if userID == "" || workspaceID == "" {
		s.logger.Warn("WorkspaceVerificationService: Invalid input parameters",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("step", "input_validation"),
			zap.String("status", "failed"))
		return fmt.Errorf("invalid parameters: user_id and workspace_id cannot be empty")
	}

	s.logger.Debug("WorkspaceVerificationService: Input validation successful",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "input_validation"),
		zap.String("status", "success"))

	// Step 2: Query workspace membership via repository
	s.logger.Debug("WorkspaceVerificationService: Step 2 - Calling repository for membership verification",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "repository_call"))

	repoCallStart := time.Now()
	member, err := s.workspaceRepo.VerifyWorkspaceMembership(ctx, userID, workspaceID)
	repoCallDuration := time.Since(repoCallStart)

	if err != nil {
		s.logger.Warn("WorkspaceVerificationService: Repository call failed",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("step", "repository_call"),
			zap.String("status", "failed"),
			zap.Duration("repository_call_duration", repoCallDuration),
			zap.Error(err))
		return fmt.Errorf("access denied: %w", err)
	}

	s.logger.Debug("WorkspaceVerificationService: Repository call successful",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "repository_call"),
		zap.String("status", "success"),
		zap.Duration("repository_call_duration", repoCallDuration),
		zap.String("member_role", member.Role))

	// Step 3: Validate member role permissions
	s.logger.Debug("WorkspaceVerificationService: Step 3 - Validating member role permissions",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("member_role", member.Role),
		zap.String("step", "role_validation"))

	if member.Role == "" {
		s.logger.Warn("WorkspaceVerificationService: User has no role in workspace",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("member_role", member.Role),
			zap.String("step", "role_validation"),
			zap.String("status", "failed"))
		return domainErrors.NewInsufficientPermissionsError(userID, workspaceID)
	}

	s.logger.Debug("WorkspaceVerificationService: Role validation successful",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("member_role", member.Role),
		zap.String("step", "role_validation"),
		zap.String("status", "success"))

	// Log successful completion with timing information
	totalDuration := time.Since(startTime)
	s.logger.Info("WorkspaceVerificationService: Workspace access verification completed successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("member_role", member.Role),
		zap.String("member_id", member.ID),
		zap.String("joined_at", member.JoinedAt),
		zap.String("step", "verification_complete"),
		zap.String("status", "success"),
		zap.Duration("total_verification_duration", totalDuration),
		zap.Duration("repository_call_duration", repoCallDuration))

	return nil
}

// GetUserWorkspaceRole retrieves the user's role in a workspace
func (s *WorkspaceVerificationService) GetUserWorkspaceRole(
	ctx context.Context,
	userID,
	workspaceID string,
) (string, error) {
	startTime := time.Now()
	requestID := ""
	if ctx.Value("request_id") != nil {
		requestID = ctx.Value("request_id").(string)
	}
	if requestID == "" {
		requestID = fmt.Sprintf("role_%d", time.Now().UnixNano())
	}

	s.logger.Debug("WorkspaceVerificationService: Getting user workspace role",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "get_role_entry"))

	member, err := s.workspaceRepo.VerifyWorkspaceMembership(ctx, userID, workspaceID)
	if err != nil {
		duration := time.Since(startTime)
		s.logger.Warn("WorkspaceVerificationService: Failed to get user workspace role",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("step", "get_role_complete"),
			zap.String("status", "failed"),
			zap.Duration("duration", duration),
			zap.Error(err))
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	duration := time.Since(startTime)
	s.logger.Debug("WorkspaceVerificationService: Successfully retrieved user workspace role",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("role", member.Role),
		zap.String("step", "get_role_complete"),
		zap.String("status", "success"),
		zap.Duration("duration", duration))

	return member.Role, nil
}