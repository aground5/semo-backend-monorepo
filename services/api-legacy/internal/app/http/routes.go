package httpEngine

import (
	"net/http"
	"semo-server/configs"
	"semo-server/internal/controllers"
	"semo-server/internal/logics"
	"semo-server/internal/logics/config_engine"
	"semo-server/internal/middlewares"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes는 서버의 모든 라우트를 등록합니다.
func RegisterRoutes(e *echo.Echo) {
	// 기본 헬스 체크 엔드포인트 (JWT 미들웨어 없음)
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, from Semo Server!")
	})

	// 서비스 초기화를 위한 공통 의존성
	cursorManager := utils.NewCursorManager(configs.Configs.Secrets.CursorSecret)
	configEngine := config_engine.NewProfileConfigEngine()

	// 기본 서비스 초기화
	profileService := logics.NewProfileService(configEngine)
	teamService := logics.NewTeamService(profileService)

	// 다른 서비스들 초기화
	attributeService := logics.NewAttributeService()
	attributeValueService := logics.NewAttributeValueService()
	userTestService := logics.NewUserTestService()
	fileService := logics.NewFileService(repositories.DBS.S3, configs.Configs.S3.BucketName) // S3 클라이언트 추가 필요
	entryService := logics.NewEntryService(cursorManager)
	shareService := logics.NewShareService(cursorManager)
	taskService := logics.NewTaskService(cursorManager, entryService)
	projectService := logics.NewProjectService(cursorManager)

	//db 사용
	projectMemberService := logics.NewProjectMemberService(profileService, teamService)
	taskPermissionService := logics.NewTaskPermissionService(taskService, entryService, shareService, projectMemberService)
	searchService := logics.NewSearchService(cursorManager, taskPermissionService, projectMemberService, taskService)
	llmService := logics.NewLLMService(repositories.DBS.Postgres, taskService, userTestService)

	// 컨트롤러 초기화 - 필요한 서비스 주입
	attributeController := controllers.NewAttributeController(attributeService, attributeValueService, profileService, taskPermissionService)
	entryController := controllers.NewEntryController(profileService, entryService)
	fileController := controllers.NewFileController(fileService, profileService, taskPermissionService)
	kickoffController := controllers.NewKickoffController(llmService, profileService)
	profileController := controllers.NewProfileController(profileService, searchService)
	projectController := controllers.NewProjectController(projectService, projectMemberService, profileService)
	taskController := controllers.NewTaskController(taskService, profileService, taskPermissionService, projectMemberService)
	taskPermissionController := controllers.NewTaskPermissionController(taskPermissionService, profileService)
	shareController := controllers.NewShareController(taskPermissionService)
	sharePermissionController := controllers.NewSharePermissionController(taskPermissionService, profileService, shareService)
	api := e.Group("/api")
	api.Use(middlewares.JWTMiddleware)
	api.GET("/files/:id", fileController.DownloadFile)
	api.DELETE("/files/:id", fileController.DeleteFile)

	apiV1 := e.Group("/api/v1")
	apiV1.Use(middlewares.JWTMiddleware)

	// 각 컨트롤러의 핸들러 등록
	// 킥오프 관련 엔드포인트
	apiV1.POST("/kickoff/preview", kickoffController.GeneratePreview)
	apiV1.POST("/kickoff/pre-questions", kickoffController.GeneratePreQuestions)
	apiV1.POST("/kickoff/details", kickoffController.GenerateDetails)

	// 태스크 관련 엔드포인트
	apiV1.GET("/tasks/:id", taskController.GetTask)
	apiV1.POST("/tasks", taskController.CreateTask)
	apiV1.PUT("/tasks/:id", taskController.UpdateTask)
	apiV1.DELETE("/tasks/:id", taskController.DeleteTask)
	apiV1.GET("/tasks/:id/children", taskController.GetChildTasks)

	// 태스크 권한 관련 엔드포인트
	apiV1.GET("/tasks/:id/permissions", taskPermissionController.GetPermissions)
	apiV1.POST("/tasks/:id/permissions", taskPermissionController.GrantPermission)
	apiV1.DELETE("/tasks/:id/permissions/:profile_id", taskPermissionController.RevokePermission)
	//apiV1.GET("/tasks/:id/permissions/:profile_id/check", taskPermissionController.CheckPermission)

	// uuid를 통한 태스크 권한 관련 엔드포인트
	apiV1.GET("/tasks/:id/share", sharePermissionController.GetShareUUID)
	apiV1.POST("/tasks/share", sharePermissionController.GrantPermissionFromUUID)
	apiV1.DELETE("/tasks/share/:uuid", sharePermissionController.RevokePermissionFromUUID)

	// 공개 엔드포인트 (JWT 미들웨어 없음)
	e.GET("/api/share/:uuid", shareController.GetSharedTaskFromUUID)

	// 속성 관련 엔드포인트
	apiV1.POST("/attributes", attributeController.CreateAttribute)
	apiV1.PUT("/attributes/:id", attributeController.UpdateAttribute)
	apiV1.POST("/attribute-values", attributeController.EditAttributeValue)
	apiV1.GET("/tasks/:id/attributes", attributeController.GetAttributesOfRootTask)
	apiV1.GET("/tasks/:id/attribute-values", attributeController.GetAttributeValuesOfTask)
	apiV1.DELETE("/attributes/:id", attributeController.DeleteAttribute)

	// 파일 관련 엔드포인트
	apiV1.POST("/items/:item_id/files", fileController.UploadFile)
	apiV1.GET("/items/:item_id/files", fileController.ListFiles)

	// 프로필 관련 엔드포인트
	apiV1.GET("/profile", profileController.GetProfile)
	apiV1.PUT("/profile", profileController.UpdateProfile)
	apiV1.GET("/profile/search", profileController.SearchProfile)
	apiV1.POST("/profiles/invite", profileController.CreateInvitedProfile)

	// 프로젝트 관련 엔드포인트
	apiV1.GET("/projects", projectController.ListUserProjects)
	apiV1.GET("/projects/:id", projectController.GetProject)
	apiV1.POST("/projects", projectController.CreateProject)
	apiV1.PUT("/projects/:id", projectController.UpdateProject)
	apiV1.DELETE("/projects/:id", projectController.DeleteProject)

	// 프로젝트 멤버 관련 엔드포인트
	apiV1.GET("/projects/:id/tasks", projectController.ListProjectTasks)
	apiV1.GET("/projects/:id/members", projectController.ListProjectMembers)
	apiV1.POST("/projects/:id/members", projectController.InviteMemberToProject)
	apiV1.DELETE("/projects/:id/members/:user_id", projectController.RemoveMemberFromProject)
	apiV1.PUT("/projects/:id/members/:user_id", projectController.UpdateMemberRole)
	apiV1.GET("/projects/:id/members/:user_id", projectController.GetMemberRole)

	// 프로젝트 초대 관련 엔드포인트
	apiV1.GET("/projects/invitations", projectController.ListProjectInvitations)
	apiV1.POST("/projects/invitations/:id/accept", projectController.AcceptProjectInvitation)
	apiV1.POST("/projects/invitations/:id/reject", projectController.RejectProjectInvitation)

	// 엔트리 관련 엔드포인트
	apiV1.GET("/entries", entryController.ListEntries)
}
