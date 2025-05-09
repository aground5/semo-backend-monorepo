package logics

import (
	"errors"
	"fmt"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

// TaskResult represents the paginated tasks result
type TaskResult struct {
	Tasks []models.Item `json:"tasks"`
	utils.PaginationResult
}

// TaskService provides business logic for tasks.
type TaskService struct {
	db            *gorm.DB
	cursorManager *utils.CursorManager
	entryService  *EntryService
}

// NewTaskService creates a new TaskService instance.
func NewTaskService(db *gorm.DB, cursorManager *utils.CursorManager, entryService *EntryService) *TaskService {
	return &TaskService{
		db:            db,
		cursorManager: cursorManager,
		entryService:  entryService,
	}
}

// CreateTask creates a new task with proper ordering using left_item_id.
// If leftItemID is provided, the new task's position is calculated based on the left item within the same parent.
// If leftItemID is nil, the new task is placed at the end of the parent group (max position + 1).
func (ts *TaskService) CreateTask(input *models.Item, leftItemID *string) (*models.Item, error) {
	// Validate required fields.
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("task name is required")
	}

	// Force type to be "task"
	input.Type = "task"

	// Generate a unique ID for the new task.
	newID, err := utils.GenerateUniqueID("IT")
	if err != nil {
		return nil, fmt.Errorf("failed to generate task ID: %w", err)
	}
	input.ID = newID

	// Determine the new position.
	if leftItemID != nil {
		newPos, err := recalcNewItemPositionForUpdate(input.ParentID, input.Type, *leftItemID, "")
		if err != nil {
			return nil, err
		}
		input.Position = newPos
	} else {
		// No left_item_id provided → place at the end of the group.
		var maxPos decimal.Decimal
		if input.ParentID == nil || *input.ParentID == "" {
			if err := repositories.DBS.Postgres.
				Model(&models.Item{}).
				Where("parent_id IS NULL AND type = ?", input.Type).
				Select("COALESCE(MAX(position), 0)").
				Scan(&maxPos).Error; err != nil {
				return nil, fmt.Errorf("failed to get last position: %w", err)
			}
		} else {
			if err := repositories.DBS.Postgres.
				Model(&models.Item{}).
				Where("parent_id = ? AND type = ?", *input.ParentID, input.Type).
				Select("COALESCE(MAX(position), 0)").
				Scan(&maxPos).Error; err != nil {
				return nil, fmt.Errorf("failed to get last position: %w", err)
			}
		}
		input.Position = maxPos.Add(decimal.NewFromInt(1))
	}

	input.Color, _ = utils.UniqueIDSvc.GenerateRandomColor()

	// Create the task in the database.
	if err := repositories.DBS.Postgres.Create(&input).Error; err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if input.ParentID == nil || *input.ParentID == "" {
		// 엔트리 생성
		entry := &models.Entry{
			Name:       input.Name,
			TaskID:     input.ID,
			RootTaskID: input.ID,
			CreatedBy:  input.CreatedBy,
			GrantedTo:  input.CreatedBy,
		}

		_, err = ts.entryService.CreateEntry(entry)
	}

	if err != nil {
		// 엔트리 생성 실패 로그 남김 (태스크 생성은 성공했으므로 에러 리턴하지 않음)
		fmt.Printf("Failed to create entry for task %s: %v", input.ID, err)
	}

	return input, nil
}

// UpdateTask updates an existing task using left_item_id for ordering.
// If left_item_id is provided in the update, the new position is recalculated.
func (ts *TaskService) UpdateTask(taskID string, updates models.ItemUpdate) (*models.Item, error) {
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task with id %s not found", taskID)
		}
		return nil, err
	}

	// Build update map for non-ordering fields.
	updateMap := map[string]interface{}{}
	if updates.Name != nil && *updates.Name != "" {
		updateMap["name"] = *updates.Name
	}
	if updates.Contents != nil {
		updateMap["contents"] = *updates.Contents
	}
	if updates.Objective != nil {
		updateMap["objective"] = *updates.Objective
	}
	if updates.Deliverable != nil {
		updateMap["deliverable"] = *updates.Deliverable
	}
	if updates.Role != nil {
		updateMap["role"] = *updates.Role
	}
	if updates.ParentID != nil {
		updateMap["parent_id"] = *updates.ParentID
	}

	// If left_item_id is provided, recalc the position.
	if updates.LeftItemID != nil && *updates.LeftItemID != task.ID {
		// Determine grouping key: if updates.ParentID is provided, use that; else use current task's ParentID.
		var groupID *string
		if updates.ParentID != nil {
			groupID = updates.ParentID
		} else {
			groupID = task.ParentID
		}
		newPos, err := recalcNewItemPositionForUpdate(groupID, task.Type, *updates.LeftItemID, taskID)
		if err != nil {
			return nil, err
		}
		updateMap["position"] = newPos
	} else if updates.ParentID != nil && (task.ParentID == nil || *updates.ParentID != *task.ParentID) {
		// ParentID가 변경되었고, LeftItemID가 제공되지 않은 경우
		// 새 부모 그룹에서 가장 작은 position 값보다 작은 값을 할당
		var minPos decimal.Decimal
		if err := repositories.DBS.Postgres.
			Model(&models.Item{}).
			Where("parent_id = ? AND type = ?", *updates.ParentID, task.Type).
			Select("COALESCE(MIN(position), 1)").
			Scan(&minPos).Error; err != nil {
			return nil, fmt.Errorf("새 부모 그룹의 최소 position 조회 실패: %w", err)
		}
		// 0과 최소값의 중간값을 할당
		updateMap["position"] = decimal.Avg(decimal.Zero, minPos)
	}

	if len(updateMap) == 0 {
		return &task, nil
	}

	if err := repositories.DBS.Postgres.Model(&task).Updates(updateMap).Error; err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	if err := repositories.DBS.Postgres.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve updated task: %w", err)
	}
	return &task, nil
}

// GetTask retrieves a task by its ID.
func (ts *TaskService) GetTask(taskID string) (*models.Item, error) {
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("task with id %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}
	return &task, nil
}

// DeleteTask deletes a task by its ID.
func (ts *TaskService) DeleteTask(taskID string) error {
	var task models.Item
	if err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", taskID, "task").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("task with id %s not found", taskID)
		}
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	// 트랜잭션 시작
	tx := repositories.DBS.Postgres.Begin()
	if tx.Error != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", tx.Error)
	}

	// 해당 taskID를 부모로 하는 모든 자식 태스크들을 재귀적으로 삭제
	if err := ts.deleteChildTasksRecursively(tx, taskID); err != nil {
		tx.Rollback()
		return fmt.Errorf("자식 태스크 삭제 실패: %w", err)
	}

	// 해당 taskID를 가리키는 모든 엔트리들을 삭제
	if err := tx.Where("task_id = ?", taskID).Delete(&models.Entry{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("관련 엔트리 삭제 실패: %w", err)
	}

	// 해당 태스크 삭제
	if err := tx.Delete(&task).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("태스크 삭제 실패: %w", err)
	}

	// 트랜잭션 커밋
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}

	return nil
}

// deleteChildTasksRecursively 태스크의 모든 자식 태스크를 재귀적으로 삭제하는 헬퍼 함수
func (ts *TaskService) deleteChildTasksRecursively(tx *gorm.DB, parentID string) error {
	// 자식 태스크 목록 조회
	var childTasks []models.Item
	if err := tx.Where("parent_id = ? AND type = ?", parentID, "task").Find(&childTasks).Error; err != nil {
		return fmt.Errorf("자식 태스크 조회 실패: %w", err)
	}

	// 각 자식 태스크에 대해 재귀적으로 처리
	for _, childTask := range childTasks {
		// 자식의 자식 태스크들 삭제
		if err := ts.deleteChildTasksRecursively(tx, childTask.ID); err != nil {
			return err
		}

		// 해당 자식 태스크와 관련된 엔트리 삭제
		if err := tx.Where("task_id = ?", childTask.ID).Delete(&models.Entry{}).Error; err != nil {
			return fmt.Errorf("자식 태스크의 엔트리 삭제 실패: %w", err)
		}

		// 자식 태스크 자체 삭제
		if err := tx.Delete(&childTask).Error; err != nil {
			return fmt.Errorf("자식 태스크 삭제 실패: %w", err)
		}
	}

	return nil
}

// FindRootTaskID 최상위 루트 태스크 ID를 찾는 함수
// taskID로부터 시작하여 상위 계층을 거슬러 올라가 루트 태스크 ID를 반환
func (ts *TaskService) FindRootTaskID(taskID string) string {
	var task models.Item
	var currentID string = taskID

	for {
		err := repositories.DBS.Postgres.First(&task, "id = ? AND type = ?", currentID, "task").Error
		if err != nil {
			// 태스크를 찾을 수 없는 경우 현재 ID 반환
			return currentID
		}

		// 상위 태스크가 없으면 현재 태스크가 루트
		if task.ParentID == nil || *task.ParentID == "" {
			return currentID
		}

		// 상위 태스크로 이동
		currentID = *task.ParentID
		task = models.Item{}
	}
}

// FindProjectID 프로젝트 ID를 찾는 함수
// taskID로부터 시작하여 루트 태스크를 찾고, 루트 태스크의 상위 프로젝트 ID를 반환
// 프로젝트가 없는 경우 에러 반환
func (ts *TaskService) FindProjectID(taskID string) (string, error) {
	// 먼저 루트 태스크 ID를 찾음
	rootTaskID := ts.FindRootTaskID(taskID)

	var rootTask models.Item
	err := repositories.DBS.Postgres.First(&rootTask, "id = ? AND type = ?", rootTaskID, "task").Error
	if err != nil {
		// 루트 태스크를 찾을 수 없는 경우 에러 반환
		return "", fmt.Errorf("root task with id %s not found: %w", rootTaskID, err)
	}

	// 루트 태스크의 상위 항목이 있는지 확인
	if rootTask.ParentID != nil && *rootTask.ParentID != "" {
		var parentItem models.Item
		err = repositories.DBS.Postgres.First(&parentItem, "id = ?", *rootTask.ParentID).Error
		if err != nil {
			return "", fmt.Errorf("parent item with id %s not found: %w", *rootTask.ParentID, err)
		}

		if parentItem.Type == "project" {
			// 상위 항목이 프로젝트인 경우 해당 ID 반환
			return parentItem.ID, nil
		}
	}

	// 프로젝트를 찾을 수 없는 경우 에러 반환
	return "", fmt.Errorf("no project found for task with id %s", taskID)
}

// CreateItemWithOrdering creates a new item with proper ordering using left_item_id.
// If leftItemID is provided, the new item's position is calculated based on the left item within the same parent.
// If leftItemID is nil, the new item is placed at the end of the parent group (max position + 1).
func (ts *TaskService) CreateItemWithOrdering(input *models.Item, leftItemID *string) (*models.Item, error) {
	return ts.CreateTask(input, leftItemID)
}

// GetChildTasks 특정 task를 부모로 하는 모든 자식 task를 페이지네이션하여 조회합니다.
func (ts *TaskService) GetChildTasks(parentID string, pagination utils.CursorPagination) (*TaskResult, error) {
	var tasks []models.Item
	var total int64

	// 부모 task가 존재하는지 확인
	var parentTask models.Item
	if err := repositories.DBS.Postgres.First(&parentTask, "id = ? AND type = ?", parentID, "task").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("부모 task ID %s를 찾을 수 없습니다", parentID)
		}
		return nil, fmt.Errorf("부모 task 조회 실패: %w", err)
	}

	// 기본 쿼리 생성
	query := repositories.DBS.Postgres.Model(&models.Item{}).Where("parent_id = ? AND type = ?", parentID, "task")

	// 전체 자식 task 수 카운트
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("자식 task 수 카운트 실패: %w", err)
	}

	// 커서 기반 페이지네이션 처리
	if pagination.Cursor != "" {
		cursorData, err := ts.cursorManager.DecodeCursor(pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("커서 디코딩 실패: %w", err)
		}

		// 커서를 이용하여 다음 페이지 조회
		query = query.Where("(position > (SELECT position FROM items WHERE id = ?)) OR (position = (SELECT position FROM items WHERE id = ?) AND id > ?)",
			cursorData.ID, cursorData.ID, cursorData.ID)
	}

	// 결과 조회
	if err := query.Order("position ASC, id ASC").Limit(pagination.Limit + 1).Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("자식 task 조회 실패: %w", err)
	}

	// 다음 페이지 여부 확인 및 NextCursor 설정
	hasMore := false
	nextCursor := ""

	if len(tasks) > pagination.Limit {
		hasMore = true
		lastItem := tasks[pagination.Limit-1]
		nextCursor = ts.cursorManager.EncodeCursor(time.Now(), lastItem.ID)
		tasks = tasks[:pagination.Limit] // 초과 항목 제거
	}

	// 결과 반환
	result := &TaskResult{
		Tasks: tasks,
		PaginationResult: utils.PaginationResult{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}

	return result, nil
}
