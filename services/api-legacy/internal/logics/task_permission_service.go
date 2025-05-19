package logics

import (
	"fmt"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"strings"

	"github.com/google/uuid"
)

// TaskPermissionService provides business logic for task permissions.
type TaskPermissionService struct {
	taskService          *TaskService
	entryService         *EntryService
	shareService         *ShareService
	projectMemberService *ProjectMemberService
}

// NewTaskPermissionService creates a new TaskPermissionService instance.
func NewTaskPermissionService(taskService *TaskService, entryService *EntryService, shareService *ShareService, projectMemberService *ProjectMemberService) *TaskPermissionService {
	return &TaskPermissionService{
		taskService:          taskService,
		entryService:         entryService,
		shareService:         shareService,
		projectMemberService: projectMemberService,
	}
}

// ListPermissions 특정 태스크에 권한이 있는 사용자 목록 조회
func (tps *TaskPermissionService) ListPermissions(taskID string) ([]models.Profile, error) {
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		return nil, fmt.Errorf("태스크를 찾을 수 없음: %w", err)
	}

	var entries []models.Entry
	rootTaskID := tps.taskService.FindRootTaskID(taskID)
	if err := repositories.DBS.Postgres.Where("task_id = ? OR root_task_id = ?", taskID, rootTaskID).Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("엔트리 조회 실패: %w", err)
	}

	// 중복 없이 권한 있는 프로필 ID 수집
	profileIDMap := make(map[string]struct{})
	for _, entry := range entries {
		profileIDMap[entry.CreatedBy] = struct{}{}
	}

	// 권한 있는 프로필 목록 조회
	var profileIDs []string
	for id := range profileIDMap {
		profileIDs = append(profileIDs, id)
	}

	if len(profileIDs) == 0 {
		return []models.Profile{}, nil
	}

	var profiles []models.Profile
	if err := repositories.DBS.Postgres.Where("id IN ?", profileIDs).Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("프로필 조회 실패: %w", err)
	}

	return profiles, nil
}

// GrantPermission 특정 태스크에 대한 권한 부여
func (tps *TaskPermissionService) GrantPermission(taskID, profileID string) error {
	// 태스크 존재 확인
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		return fmt.Errorf("태스크를 찾을 수 없음: %w", err)
	}

	// 프로필 존재 확인
	var profile models.Profile
	if err := repositories.DBS.Postgres.First(&profile, "id = ?", profileID).Error; err != nil {
		return fmt.Errorf("프로필을 찾을 수 없음: %w", err)
	}

	// 이미 권한이 있는지 확인
	hasPermission, err := tps.CheckPermission(taskID, profileID)
	if err != nil {
		return fmt.Errorf("권한 확인 실패: %w", err)
	}
	if hasPermission {
		return nil // 이미 권한 있음
	}

	// root_task_id 확인
	rootTaskID := tps.taskService.FindRootTaskID(taskID)

	// 새 엔트리 생성하여 권한 부여
	entry := &models.Entry{
		Name:       task.Name,
		TaskID:     taskID,
		RootTaskID: rootTaskID,
		CreatedBy:  profileID,
	}

	_, err = tps.entryService.CreateEntry(entry)
	if err != nil {
		return fmt.Errorf("권한 부여 실패: %w", err)
	}

	return nil
}

// RevokePermission 특정 태스크에 대한 권한 회수
func (tps *TaskPermissionService) RevokePermission(taskID, profileID string) error {
	rootTaskID := tps.taskService.FindRootTaskID(taskID)
	result := repositories.DBS.Postgres.Where("(task_id = ? OR root_task_id = ?) AND granted_to = ?", taskID, rootTaskID, profileID).
		Delete(&models.Entry{})

	if result.Error != nil {
		return fmt.Errorf("권한 회수 실패: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("해당 사용자에게 권한이 없음")
	}

	return nil
}

// CheckPermission 특정 사용자가 태스크에 대한 권한이 있는지 확인
func (tps *TaskPermissionService) CheckPermission(taskID, profileID string) (bool, error) {
	var count int64
	rootTaskID := tps.taskService.FindRootTaskID(taskID)
	if strings.Contains(rootTaskID, "IP") {
		hasPermission, err := tps.projectMemberService.CheckPermission(rootTaskID, profileID)
		if err != nil {
			return false, fmt.Errorf("권한 확인 실패: %w", err)
		}
		return hasPermission, nil
	}
	fmt.Println("rootTaskID", rootTaskID)
	if err := repositories.DBS.Postgres.Model(&models.Entry{}).
		Where("(task_id = ? OR root_task_id = ?) AND granted_to = ?", taskID, rootTaskID, profileID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("권한 확인 실패: %w", err)
	}

	return count > 0, nil
}

// GrantPermissionWithUUID 프로젝트 ID를 받아 UUID를 생성하고 권한을 부여하는 메서드
func (tps *TaskPermissionService) GrantPermissionWithUUID(taskID, profileID string) (string, error) {
	// 테스크 존재 확인
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		return "", fmt.Errorf("테스크를 찾을 수 없음: %w", err)
	}

	// 프로필 존재 확인
	var profile models.Profile
	if err := repositories.DBS.Postgres.First(&profile, "id = ?", profileID).Error; err != nil {
		return "", fmt.Errorf("프로필을 찾을 수 없음: %w", err)
	}

	// UUID 생성
	uuidStr := uuid.New().String()

	// 테스크의 루트 테스크 ID 조회
	rootTaskID := tps.taskService.FindRootTaskID(taskID)

	// 새 공유 생성
	share := &models.Share{
		ID:         uuidStr,
		RootTaskID: rootTaskID,
		CreatedBy:  profileID,
		GrantedTo:  profileID,
	}

	// 공유 생성
	_, err := tps.shareService.CreateShare(share)
	if err != nil {
		return "", fmt.Errorf("권한 부여 실패: %w", err)
	}

	return uuidStr, nil
}

// RevokePermissionWithUUID UUID를 통해 권한을 회수하는 메서드
func (tps *TaskPermissionService) RevokePermissionWithUUID(uuid string) error {
	return tps.shareService.DeleteShare(uuid)
}

// CheckShareExists checks if a share exists for the given taskID and profileID
func (tps *TaskPermissionService) CheckShareExists(taskID, profileID string) (bool, error) {
	return tps.shareService.CheckShareExists(taskID, profileID)
}

// TaskWithDepth represents a task with its depth information
type TaskWithDepth struct {
	Task  models.Item `json:"task"`
	Depth int         `json:"depth"`
}

// getAllTasksByBFS retrieves all descendant tasks using Breadth-First Search
func (tps *TaskPermissionService) getAllTasksByBFS(rootID string) ([]TaskWithDepth, error) {
	var allTasks []TaskWithDepth
	queue := []string{rootID}
	visited := make(map[string]bool)
	depth := 0

	for len(queue) > 0 {
		// Get the current level size for level-by-level processing
		levelSize := len(queue)

		for i := 0; i < levelSize; i++ {
			currentID := queue[0]
			queue = queue[1:] // Dequeue

			// Skip if already visited
			if visited[currentID] {
				continue
			}
			visited[currentID] = true

			// Get all direct children of current task
			var children []models.Item
			if err := repositories.DBS.Postgres.Where(&models.Item{
				ParentID: &currentID,
				Type:     "task",
			}).Order("position ASC").Find(&children).Error; err != nil {
				return nil, fmt.Errorf("failed to get children for task %s: %w", currentID, err)
			}

			// Add children to result and queue
			for _, child := range children {
				if !visited[child.ID] {
					allTasks = append(allTasks, TaskWithDepth{
						Task:  child,
						Depth: depth + 1,
					})
					queue = append(queue, child.ID)
				}
			}
		}
		depth++
	}

	return allTasks, nil
}

// GetTaskAndChildrenByShareUUID retrieves a task and its children by share UUID
func (tps *TaskPermissionService) GetTaskAndChildrenByShareUUID(uuid string) (*models.Item, []TaskWithDepth, error) {
	// Get share by UUID
	var share models.Share
	if err := repositories.DBS.Postgres.Where("id::text = ?", uuid).First(&share).Error; err != nil {
		return nil, nil, fmt.Errorf("share not found: %w", err)
	}

	// Get task by task_id
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", share.RootTaskID, "task").Error; err != nil {
		return nil, nil, fmt.Errorf("task not found: %w", err)
	}

	// Get all descendant tasks using BFS
	childTasks, err := tps.getAllTasksByBFS(share.RootTaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get child tasks: %w", err)
	}

	return &task, childTasks, nil
}
