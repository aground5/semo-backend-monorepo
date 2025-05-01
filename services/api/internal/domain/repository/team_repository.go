package repository

import (
	"context"
	"time"
)

// TeamInfo는 팀 정보를 표현합니다
type TeamInfo struct {
	ID          string
	Name        string
	Description string
	ImageURL    string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TeamMemberInfo는 팀 멤버 정보를 표현합니다
type TeamMemberInfo struct {
	TeamID     string
	ProfileID  string
	Role       string
	InvitedBy  string
	InvitedAt  time.Time
	AcceptedAt *time.Time
	RejectedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TeamRepository 팀 관련 저장소 인터페이스
type TeamRepository interface {
	// FindByID ID로 팀 정보 조회
	FindByID(ctx context.Context, id string) (*TeamInfo, error)

	// FindByProfile 사용자가 속한 팀 목록 조회
	FindByProfile(ctx context.Context, profileID string) ([]*TeamInfo, error)

	// Create 새 팀 생성
	Create(ctx context.Context, team *TeamInfo) error

	// Update 팀 정보 업데이트
	Update(ctx context.Context, team *TeamInfo) error

	// Delete 팀 삭제
	Delete(ctx context.Context, id string) error

	// AddMember 팀에 멤버 추가
	AddMember(ctx context.Context, teamID, profileID, role, invitedBy string) error

	// RemoveMember 팀에서 멤버 제거
	RemoveMember(ctx context.Context, teamID, profileID string) error

	// FindMembers 팀 멤버 목록 조회
	FindMembers(ctx context.Context, teamID string) ([]*TeamMemberInfo, error)

	// FindMembersByProfile 사용자의 팀 멤버십 정보 조회
	FindMembersByProfile(ctx context.Context, profileID string) ([]*TeamMemberInfo, error)

	// UpdateMemberRole 팀 멤버 역할 업데이트
	UpdateMemberRole(ctx context.Context, teamID, profileID, role string) error

	// AcceptInvitation 초대 수락
	AcceptInvitation(ctx context.Context, teamID, profileID string) error

	// RejectInvitation 초대 거절
	RejectInvitation(ctx context.Context, teamID, profileID string) error
}
