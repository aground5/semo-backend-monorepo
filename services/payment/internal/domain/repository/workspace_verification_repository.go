package repository

import "context"

// WorkspaceMember represents a member in a workspace
type WorkspaceMember struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Role        string `json:"role"`
	JoinedAt    string `json:"joined_at"`
	InvitedBy   string `json:"invited_by"`
	Description string `json:"description"`
}

// WorkspaceVerificationRepository defines the interface for workspace verification operations
type WorkspaceVerificationRepository interface {
	// VerifyWorkspaceMembership checks if a user belongs to a workspace
	VerifyWorkspaceMembership(ctx context.Context, userID, workspaceID string) (*WorkspaceMember, error)
}