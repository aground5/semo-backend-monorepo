package logics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"semo-server/internal/models"
	"semo-server/internal/repositories"

	authzedpb "github.com/authzed/authzed-go/proto/authzed/api/v1"
)

// -----------------------------------------------------------------------------
// 1) ObjectType, Relation, Permission을 enum 형태로 정의
// -----------------------------------------------------------------------------

// ObjectType은 SpiceDB에서 사용될 Object의 종류를 정의합니다.
type ObjectType int

const (
	ObjectTypeProfile ObjectType = iota
	ObjectTypeTeam
	ObjectTypeOrganization
	ObjectTypeItem
)

// String() 메서드: Authzed에 전달할 때 사용할 실제 string
func (ot ObjectType) String() string {
	switch ot {
	case ObjectTypeProfile:
		return "profile"
	case ObjectTypeTeam:
		return "team"
	case ObjectTypeOrganization:
		return "organization"
	case ObjectTypeItem:
		return "item"
	default:
		return "unknown"
	}
}

// ParseObjectType은 입력받은 문자열을 ObjectType enum으로 변환합니다.
func ParseObjectType(s string) (ObjectType, error) {
	switch strings.ToLower(s) {
	case "profile":
		return ObjectTypeProfile, nil
	case "team":
		return ObjectTypeTeam, nil
	case "organization":
		return ObjectTypeOrganization, nil
	case "item":
		return ObjectTypeItem, nil
	default:
		return 0, fmt.Errorf("unknown object type: %s", s)
	}
}

// Relation은 WriteRelationships 시에 사용될 'relation'을 정의합니다.
type Relation int

const (
	RelationOwner Relation = iota
	RelationMember
	RelationManager
	RelationDirectMember
	RelationDirectGuest
	RelationAdministrator
	RelationReader
	RelationParent
)

// String() 메서드: Authzed에 전달할 때 사용할 실제 string
func (r Relation) String() string {
	switch r {
	case RelationOwner:
		return "owner"
	case RelationMember:
		return "member"
	case RelationManager:
		return "manager"
	case RelationDirectMember:
		return "direct_member"
	case RelationDirectGuest:
		return "direct_guest"
	case RelationAdministrator:
		return "administrator"
	case RelationReader:
		return "reader"
	case RelationParent:
		return "parent"
	default:
		return "unknown"
	}
}

// ParseRelation은 입력받은 문자열을 Relation enum으로 변환합니다.
func ParseRelation(s string) (Relation, error) {
	switch strings.ToLower(s) {
	case "owner":
		return RelationOwner, nil
	case "member":
		return RelationMember, nil
	case "manager":
		return RelationManager, nil
	case "direct_member":
		return RelationDirectMember, nil
	case "direct_guest":
		return RelationDirectGuest, nil
	case "administrator":
		return RelationAdministrator, nil
	case "reader":
		return RelationReader, nil
	case "parent":
		return RelationParent, nil
	default:
		return 0, fmt.Errorf("unknown relation: %s", s)
	}
}

// Permission은 CheckPermission/LookupSubjects 시에 사용될 'permission'을 정의합니다.
type Permission int

const (
	PermissionManage Permission = iota
	PermissionWrite
	PermissionRead
	PermissionMember
	PermissionReader
	PermissionAdmin
)

// String() 메서드: Authzed에 전달할 때 사용할 실제 string
func (p Permission) String() string {
	switch p {
	case PermissionManage:
		return "manage"
	case PermissionWrite:
		return "write"
	case PermissionRead:
		return "read"
	case PermissionMember:
		return "member"
	case PermissionReader:
		return "reader"
	case PermissionAdmin:
		return "admin"
	default:
		return "unknown"
	}
}

// ParsePermission은 입력받은 문자열을 Permission enum으로 변환합니다.
func ParsePermission(s string) (Permission, error) {
	switch strings.ToLower(s) {
	case "manage":
		return PermissionManage, nil
	case "write":
		return PermissionWrite, nil
	case "read":
		return PermissionRead, nil
	case "member":
		return PermissionMember, nil
	case "reader":
		return PermissionReader, nil
	case "admin":
		return PermissionAdmin, nil
	default:
		return 0, fmt.Errorf("unknown permission: %s", s)
	}
}

// -----------------------------------------------------------------------------
// 2) AuthzService 구조체 및 생성자
// -----------------------------------------------------------------------------
type AuthzService struct {
	spiceDB authzedpb.PermissionsServiceClient
}

// NewAuthzService는 endpoint(예: "localhost:50051")를 이용해 AuthzService 인스턴스를 생성합니다.
// (실제 클라이언트 생성 대신 여기서는 repositories.DBS.SpiceDB를 사용한다고 가정)
func NewAuthzService(spiceDB authzedpb.PermissionsServiceClient) *AuthzService {
	return &AuthzService{
		spiceDB: spiceDB,
	}
}

// -----------------------------------------------------------------------------
// 3) 범용 WriteRelationship(= 관계 생성/업데이트) 함수
//   - objectType, subjectType : 위에서 정의한 ObjectType enum
//   - relation                : 위에서 정의한 Relation enum
//
// -----------------------------------------------------------------------------
func (a *AuthzService) CreateRelationship(
	objectType ObjectType, objectID string,
	relation Relation,
	subjectType ObjectType, subjectID string,
) error {
	subjectID = fmt.Sprintf("%s", subjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resource := &authzedpb.ObjectReference{
		ObjectType: objectType.String(),
		ObjectId:   objectID,
	}

	subject := &authzedpb.SubjectReference{
		Object: &authzedpb.ObjectReference{
			ObjectType: subjectType.String(),
			ObjectId:   subjectID,
		},
	}

	relationship := &authzedpb.Relationship{
		Resource: resource,
		Relation: relation.String(),
		Subject:  subject,
	}

	req := &authzedpb.WriteRelationshipsRequest{
		Updates: []*authzedpb.RelationshipUpdate{
			{
				Operation:    authzedpb.RelationshipUpdate_OPERATION_TOUCH, // 없으면 생성, 있으면 업데이트
				Relationship: relationship,
			},
		},
	}

	if _, err := a.spiceDB.WriteRelationships(ctx, req); err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// 4) 범용 CheckPermission 함수
//   - objectType, subjectType : 위에서 정의한 ObjectType enum
//   - permission              : 위에서 정의한 Permission enum
//
// -----------------------------------------------------------------------------
func (a *AuthzService) CheckPermission(
	objectType ObjectType, objectID string,
	permission Permission,
	subjectType ObjectType, subjectID string,
) (bool, error) {
	subjectID = fmt.Sprintf("%s", subjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resource := &authzedpb.ObjectReference{
		ObjectType: objectType.String(),
		ObjectId:   objectID,
	}
	subject := &authzedpb.SubjectReference{
		Object: &authzedpb.ObjectReference{
			ObjectType: subjectType.String(),
			ObjectId:   subjectID,
		},
	}

	req := &authzedpb.CheckPermissionRequest{
		Resource:   resource,
		Subject:    subject,
		Permission: permission.String(),
	}

	resp, err := a.spiceDB.CheckPermission(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return resp.Permissionship == authzedpb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION, nil
}

// -----------------------------------------------------------------------------
//  5. 범용 LookupSubjects 함수
//     특정 Object에 대하여, 해당 permission을 갖는 subject들을 찾습니다.
//
// -----------------------------------------------------------------------------
func (a *AuthzService) LookupSubjects(
	objectType ObjectType, objectID string,
	permission Permission,
) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resource := &authzedpb.ObjectReference{
		ObjectType: objectType.String(),
		ObjectId:   objectID,
	}

	req := &authzedpb.LookupSubjectsRequest{
		Resource:   resource,
		Permission: permission.String(),
	}

	stream, err := a.spiceDB.LookupSubjects(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup subjects: %w", err)
	}

	var subjects []string
	for {
		res, err := stream.Recv()
		if err != nil {
			// stream이 끝났거나 에러 발생
			break
		}
		if res.Subject != nil && res.Subject.SubjectObjectId != "" {
			// 예: "user:PABC123" 형태로 반환하고 싶다면 "user" + ":" + ID
			subjects = append(subjects,
				fmt.Sprintf("%s",
					res.Subject.GetSubjectObjectId(),
				),
			)
		}
	}
	return subjects, nil
}

// DeleteRelationship는 대상 object와 subject 간의 특정 relation을 제거합니다.
func (a *AuthzService) DeleteRelationship(
	objectType ObjectType, objectID string,
	relation Relation,
	subjectType ObjectType, subjectID string,
) error {
	// subjectID에 organization ID prefix를 붙이는 규칙을 동일하게 적용합니다.
	subjectID = fmt.Sprintf("%s", subjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resource := &authzedpb.ObjectReference{
		ObjectType: objectType.String(),
		ObjectId:   objectID,
	}

	subject := &authzedpb.SubjectReference{
		Object: &authzedpb.ObjectReference{
			ObjectType: subjectType.String(),
			ObjectId:   subjectID,
		},
	}

	relationship := &authzedpb.Relationship{
		Resource: resource,
		Relation: relation.String(),
		Subject:  subject,
	}

	req := &authzedpb.WriteRelationshipsRequest{
		Updates: []*authzedpb.RelationshipUpdate{
			{
				Operation:    authzedpb.RelationshipUpdate_OPERATION_DELETE,
				Relationship: relationship,
			},
		},
	}

	if _, err := a.spiceDB.WriteRelationships(ctx, req); err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}
	return nil
}

// ReadRelationships 함수의 응답을 표현할 구조체 예시
type AuthzRelationship struct {
	ObjectType  string `json:"object_type"`
	ObjectID    string `json:"object_id"`
	Relation    string `json:"relation"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
}

func (a *AuthzService) ListRelationships(
	objectType ObjectType,
	objectID string,
) ([]AuthzRelationship, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ReadRelationships 요청을 구성합니다.
	req := &authzedpb.ReadRelationshipsRequest{
		RelationshipFilter: &authzedpb.RelationshipFilter{
			ResourceType:       objectType.String(),
			OptionalResourceId: objectID,
			// 특정 relation만 조회하고 싶다면 OptionalRelation를 설정할 수 있습니다.
			// e.g. OptionalRelation: "reader"
		},
	}

	stream, err := repositories.DBS.SpiceDB.ReadRelationships(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read relationships: %w", err)
	}

	var results []AuthzRelationship

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				// 더 이상 결과 없음
				break
			}
			// 그 외 에러
			break
		}
		if rel := resp.GetRelationship(); rel != nil {
			r := AuthzRelationship{
				ObjectType:  rel.Resource.ObjectType,
				ObjectID:    rel.Resource.ObjectId,
				Relation:    rel.Relation,
				SubjectType: rel.Subject.Object.ObjectType,
				SubjectID:   rel.Subject.Object.ObjectId,
			}
			results = append(results, r)
		}
	}

	return results, nil
}

// -----------------------------------------------------------------------------
// 6) 필요하다면, 기존 함수들을 새로운 범용 함수로 감싸서 제공할 수도 있습니다.
//    예) RegisterUserInOrganization == CreateRelationship(organization, orgID, RelationDirectMember, user, userID)
// -----------------------------------------------------------------------------

// RegisterUserInOrganization: 예시 래퍼 함수
func (a *AuthzService) RegisterUserInOrganization(orgID, userID string) error {
	return a.CreateRelationship(
		ObjectTypeOrganization, // "organization"
		orgID,
		RelationDirectMember, // "direct_member"
		ObjectTypeProfile,    // "user"
		userID,
	)
}

// IsUserRegisteredInOrganization: 예시 래퍼 함수
func (a *AuthzService) IsUserRegisteredInOrganization(orgID, userID string) (bool, error) {
	return a.CheckPermission(
		ObjectTypeOrganization,
		orgID,
		PermissionMember, // "member"
		ObjectTypeProfile,
		userID,
	)
}

// SetupItemRelationships sets up the appropriate relationships for a new item
func (a *AuthzService) SetupItemRelationships(item *models.Item, profileID string) error {
	// Set creator as owner
	err := a.CreateRelationship(
		ObjectTypeItem,
		item.ID,
		RelationOwner,
		ObjectTypeProfile,
		profileID,
	)
	if err != nil {
		return fmt.Errorf("failed to set owner relationship: %w", err)
	}

	// If the item has a parent, create parent relation
	if item.ParentID != nil && *item.ParentID != "" {
		err = a.CreateRelationship(
			ObjectTypeItem,
			item.ID,
			RelationParent,
			ObjectTypeItem,
			*item.ParentID,
		)
		if err != nil {
			return fmt.Errorf("failed to set parent relationship: %w", err)
		}
	}

	return nil
}
