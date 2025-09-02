package errors

import (
	"fmt"
)

// WorkspaceError represents errors related to workspace operations
type WorkspaceError struct {
	Type    string
	Message string
	UserID  string
	WorkspaceID string
	Cause   error
}

func (e *WorkspaceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (user: %s, workspace: %s) - %v", 
			e.Type, e.Message, e.UserID, e.WorkspaceID, e.Cause)
	}
	return fmt.Sprintf("%s: %s (user: %s, workspace: %s)", 
		e.Type, e.Message, e.UserID, e.WorkspaceID)
}

func (e *WorkspaceError) Unwrap() error {
	return e.Cause
}

// Workspace error types
const (
	ErrTypeWorkspaceNotFound          = "WORKSPACE_NOT_FOUND"
	ErrTypeUserNotMember              = "USER_NOT_MEMBER"
	ErrTypeInsufficientPermissions    = "INSUFFICIENT_PERMISSIONS"
	ErrTypeWorkspaceVerificationFailed = "WORKSPACE_VERIFICATION_FAILED"
	ErrTypeSupabaseConnectionFailed   = "SUPABASE_CONNECTION_FAILED"
)

// NewWorkspaceNotFoundError creates a new workspace not found error
func NewWorkspaceNotFoundError(userID, workspaceID string) *WorkspaceError {
	return &WorkspaceError{
		Type:        ErrTypeWorkspaceNotFound,
		Message:     "workspace not found or user does not have access",
		UserID:      userID,
		WorkspaceID: workspaceID,
	}
}

// NewUserNotMemberError creates a new user not member error
func NewUserNotMemberError(userID, workspaceID string) *WorkspaceError {
	return &WorkspaceError{
		Type:        ErrTypeUserNotMember,
		Message:     "user is not a member of the workspace",
		UserID:      userID,
		WorkspaceID: workspaceID,
	}
}

// NewInsufficientPermissionsError creates a new insufficient permissions error
func NewInsufficientPermissionsError(userID, workspaceID string) *WorkspaceError {
	return &WorkspaceError{
		Type:        ErrTypeInsufficientPermissions,
		Message:     "user does not have sufficient permissions for this workspace",
		UserID:      userID,
		WorkspaceID: workspaceID,
	}
}

// NewWorkspaceVerificationError creates a new workspace verification error
func NewWorkspaceVerificationError(userID, workspaceID string, cause error) *WorkspaceError {
	return &WorkspaceError{
		Type:        ErrTypeWorkspaceVerificationFailed,
		Message:     "workspace verification failed",
		UserID:      userID,
		WorkspaceID: workspaceID,
		Cause:       cause,
	}
}

// NewSupabaseConnectionError creates a new Supabase connection error
func NewSupabaseConnectionError(userID, workspaceID string, cause error) *WorkspaceError {
	return &WorkspaceError{
		Type:        ErrTypeSupabaseConnectionFailed,
		Message:     "failed to connect to Supabase for workspace verification",
		UserID:      userID,
		WorkspaceID: workspaceID,
		Cause:       cause,
	}
}