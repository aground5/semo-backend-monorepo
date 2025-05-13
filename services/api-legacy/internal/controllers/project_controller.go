package controllers

import (
	"net/http"
	"semo-server/internal/logics"
	"semo-server/internal/models"
	"semo-server/internal/utils"

	"github.com/labstack/echo/v4"
)

// ProjectController 프로젝트 관련 HTTP 요청 처리
type ProjectController struct {
	BaseController
	projectService       *logics.ProjectService
	projectMemberService *logics.ProjectMemberService
}

// NewProjectController 새로운 ProjectController 인스턴스 생성
func NewProjectController(
	projectService *logics.ProjectService,
	projectMemberService *logics.ProjectMemberService,
	profileService *logics.ProfileService,
) *ProjectController {
	return &ProjectController{
		BaseController: NewBaseController(profileService),
		projectService:       projectService,
		projectMemberService: projectMemberService,
	}
}

// ListUserProjects 사용자가 속한 프로젝트 목록 조회
// GET /projects
func (pc *ProjectController) ListUserProjects(c echo.Context) error {
	// 컨텍스트에서 프로필 가져오기
	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 페이지네이션 파라미터 가져오기
	pagination := utils.ExtractCursorPaginationFromContext(c)
	utils.GetPaginationDefaults(&pagination, 20, 100)

	// 사용자의 프로젝트 목록 조회
	projects, err := pc.projectService.ListProjectsPaginated(profile.ID, pagination)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, projects)
}

// GetProject 프로젝트 상세 정보 조회
// GET /projects/:id
func (pc *ProjectController) GetProject(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 읽기 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 조회
	project, err := pc.projectService.GetProject(projectID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, project)
}

// CreateProject 새 프로젝트 생성
// POST /projects
func (pc *ProjectController) CreateProject(c echo.Context) error {
	var input struct {
		models.Item
		LeftItemID *string `json:"left_item_id"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 생성자 ID 설정
	input.CreatedBy = profile.ID

	// 프로젝트 생성
	project, err := pc.projectService.CreateProject(&input.Item, input.LeftItemID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, project)
}

// UpdateProject 프로젝트 정보 업데이트
// PUT /projects/:id
func (pc *ProjectController) UpdateProject(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	var updates models.ItemUpdate
	if err := c.Bind(&updates); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 쓰기 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 업데이트
	project, err := pc.projectService.UpdateProject(projectID, updates, profile.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, project)
}

// DeleteProject 프로젝트 삭제
// DELETE /projects/:id
func (pc *ProjectController) DeleteProject(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 삭제 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 삭제
	if err := pc.projectService.DeleteProject(projectID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "프로젝트가 성공적으로 삭제되었습니다"})
}

// ListProjectTasks 프로젝트 내 태스크 목록 조회
// GET /projects/:id/tasks
func (pc *ProjectController) ListProjectTasks(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 읽기 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 내 태스크 조회
	tasks, err := pc.projectService.GetProjectChildren(projectID, "task")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, tasks)
}

// ListProjectMembers 프로젝트 멤버 목록 조회
// GET /projects/:id/members
func (pc *ProjectController) ListProjectMembers(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 읽기 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 멤버 조회
	members, err := pc.projectMemberService.ListProjectMembers(projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, members)
}

// InviteMemberToProject 프로젝트에 멤버 초대
// POST /projects/:id/members
// 요청 본문: {"user_id": "U123", "role": "member"}
func (pc *ProjectController) InviteMemberToProject(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	var input struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if input.UserID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	}
	if input.Role == "" {
		input.Role = "member" // 기본 역할 설정
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 멤버 관리 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트에 멤버 초대
	if err := pc.projectMemberService.AddMemberToProject(projectID, input.UserID, profile.ID, input.Role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "초대가 성공적으로 전송되었습니다"})
}

// RemoveMemberFromProject 프로젝트에서 멤버 제거
// DELETE /projects/:id/members/:user_id
func (pc *ProjectController) RemoveMemberFromProject(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 본인이 아닌 경우 프로젝트 멤버 관리 권한 확인
	if profile.ID != userID {
		hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
		}
		if !hasPermission {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
		}
	}

	// 프로젝트에서 멤버 제거
	if err := pc.projectMemberService.RemoveMemberFromProject(projectID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "멤버가 성공적으로 제거되었습니다"})
}

// UpdateMemberRole 프로젝트 멤버 역할 변경
// PUT /projects/:id/members/:user_id
// 요청 본문: {"role": "admin"}
func (pc *ProjectController) UpdateMemberRole(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	}

	var input struct {
		Role string `json:"role"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if input.Role == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "역할이 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 멤버 관리 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 멤버 역할 변경
	if err := pc.projectMemberService.UpdateProjectMemberRole(projectID, userID, input.Role); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "멤버 역할이 성공적으로 변경되었습니다"})
}

// GetMemberRole 프로젝트 멤버 역할 조회
// GET /projects/:id/members/:user_id
func (pc *ProjectController) GetMemberRole(c echo.Context) error {
	projectID := c.Param("id")
	if projectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "프로젝트 ID가 필요합니다"})
	}

	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "사용자 ID가 필요합니다"})
	}

	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 읽기 권한 확인
	hasPermission, err := pc.projectMemberService.CheckPermission(projectID, profile.ID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}
	if !hasPermission {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "이 프로젝트에 대한 접근 권한이 없습니다"})
	}

	// 프로젝트 멤버 정보 조회
	member, err := pc.projectMemberService.GetProjectMember(projectID, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, member)
}

// ListProjectInvitations 사용자의 프로젝트 초대 목록 조회
// GET /projects/invitations
func (pc *ProjectController) ListProjectInvitations(c echo.Context) error {
	profile, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 사용자의 프로젝트 초대 목록 조회
	invitations, err := pc.projectMemberService.ListUserProjectInvitations(profile.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, invitations)
}

// AcceptProjectInvitation 프로젝트 초대 수락
// POST /projects/invitations/:id/accept
func (pc *ProjectController) AcceptProjectInvitation(c echo.Context) error {
	membershipID := c.Param("id")
	if membershipID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "초대 ID가 필요합니다"})
	}

	_, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 초대 수락
	if err := pc.projectMemberService.AcceptProjectInvitation(membershipID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "초대가 성공적으로 수락되었습니다"})
}

// RejectProjectInvitation 프로젝트 초대 거절
// POST /projects/invitations/:id/reject
func (pc *ProjectController) RejectProjectInvitation(c echo.Context) error {
	membershipID := c.Param("id")
	if membershipID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "초대 ID가 필요합니다"})
	}

	_, err := pc.GetProfileFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 프로젝트 초대 거절
	if err := pc.projectMemberService.RejectProjectInvitation(membershipID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "초대가 성공적으로 거절되었습니다"})
}
