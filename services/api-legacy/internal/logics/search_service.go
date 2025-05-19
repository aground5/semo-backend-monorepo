package logics

import (
	"errors"
	"fmt"
	"semo-server/internal/models"
	"semo-server/internal/repositories"
	"semo-server/internal/utils"
	"strings"

	"gorm.io/gorm"
)

// ProfileSearchResult 프로필 검색 결과를 담는 구조체
type ProfileSearchResult struct {
	Profile *models.Profile `json:"profile"`
	utils.PaginationResult
}

// ItemSearchResult 프로젝트 및 태스크 검색 결과를 담는 구조체
type ItemSearchResult struct {
	Items []models.Item `json:"items"`
	utils.PaginationResult
}

// SearchService 검색 관련 기능을 제공하는 서비스
type SearchService struct {
	cursorManager         *utils.CursorManager
	taskPermissionService *TaskPermissionService
	projectMemberService  *ProjectMemberService
	taskService           *TaskService
}

// NewSearchService 새로운 SearchService 인스턴스를 생성합니다.
func NewSearchService(cursorManager *utils.CursorManager, taskPermissionService *TaskPermissionService, projectMemberService *ProjectMemberService, taskService *TaskService) *SearchService {
	return &SearchService{
		cursorManager:         cursorManager,
		taskPermissionService: taskPermissionService,
		projectMemberService:  projectMemberService,
		taskService:           taskService,
	}
}

// SearchProfiles 키워드로 프로필을 검색합니다.
// 이메일을 기준으로 검색합니다.
func (ss *SearchService) SearchProfile(email string) (*ProfileSearchResult, error) {
	var profile models.Profile
	if err := repositories.DBS.Postgres.Model(&models.Profile{}).
		Where("email = ?", email).
		First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ProfileSearchResult{
				Profile: nil,
				PaginationResult: utils.PaginationResult{
					NextCursor: "",
					HasMore:    false,
				},
			}, nil
		}
		return nil, fmt.Errorf("프로필 검색 실패: %w", err)
	}

	return &ProfileSearchResult{
		Profile: &profile,
		PaginationResult: utils.PaginationResult{
			NextCursor: "",
			HasMore:    false,
		},
	}, nil
}

// SearchItems 키워드로 프로젝트 및 태스크를 검색합니다.
// 유형, 이름, 내용 등을 기준으로 검색합니다.
// 사용자는 권한이 있는 프로젝트와 태스크만 조회할 수 있습니다.
func (ss *SearchService) SearchItems(userID string, keyword string, itemType string, pagination utils.CursorPagination) (*ItemSearchResult, error) {
	// 페이지네이션 기본값 설정
	utils.GetPaginationDefaults(&pagination, 20, 100)

	// 검색어 전처리
	searchTerm := "%" + strings.ToLower(keyword) + "%"

	// 초기 로드 배수 설정 (필터링 후에도 충분한 결과를 보장하기 위함)
	multiplier := 3
	initialLimit := pagination.Limit * multiplier

	// 쿼리 준비
	query := repositories.DBS.Postgres.Model(&models.Item{}).
		Where("LOWER(name) LIKE ? OR LOWER(contents) LIKE ? OR LOWER(objective) LIKE ? OR LOWER(deliverable) LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm)

	// 아이템 유형 필터링 (프로젝트 또는 태스크)
	if itemType != "" {
		query = query.Where("type = ?", strings.ToLower(itemType))
	}

	// 커서가 제공된 경우 적용
	if pagination.Cursor != "" {
		cursorData, err := ss.cursorManager.DecodeCursor(pagination.Cursor)
		if err != nil {
			return nil, fmt.Errorf("잘못된 커서: %w", err)
		}

		// 커서 조건 적용
		query = query.Where("(updated_at < ? OR (updated_at = ? AND id < ?))",
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID)
	}

	// 초기에 더 많은 아이템을 가져옴
	query = query.Order("updated_at DESC").Order("id DESC").Limit(initialLimit)

	var allItems []models.Item
	if err := query.Find(&allItems).Error; err != nil {
		return nil, fmt.Errorf("아이템 검색 실패: %w", err)
	}

	// 권한에 기반한 필터링
	var authorizedItems []models.Item
	for _, item := range allItems {
		// 프로젝트인 경우
		if item.Type == "project" {
			hasPermission, err := ss.projectMemberService.CheckPermission(item.ID, userID)
			if err != nil {
				continue // 에러 발생 시 해당 항목 스킵
			}

			if hasPermission {
				authorizedItems = append(authorizedItems, item)
			}
		} else if item.Type == "task" {
			// 태스크인 경우
			hasPermission, err := ss.taskPermissionService.CheckPermission(item.ID, userID)
			if err != nil {
				// 직접 권한 확인 실패 시, 태스크의 프로젝트에 권한이 있는지 확인
				projectID, pErr := ss.taskService.FindProjectID(item.ID)
				if pErr == nil {
					// 프로젝트 ID를 찾았으면 프로젝트 권한 확인
					hasProjectPermission, pErr := ss.projectMemberService.CheckPermission(projectID, userID)
					if pErr == nil && hasProjectPermission {
						authorizedItems = append(authorizedItems, item)
					}
				}
			} else if hasPermission {
				authorizedItems = append(authorizedItems, item)
			}
		}

		// 이미 충분한 권한 있는 항목을 찾았으면 중단
		if len(authorizedItems) >= pagination.Limit+1 {
			break
		}
	}

	// 권한 필터링 후 충분한 아이템이 없으면, 더 많은 아이템 로드 시도
	if len(authorizedItems) < pagination.Limit && len(allItems) == initialLimit {
		// 마지막 항목의 커서 가져오기
		lastItem := allItems[len(allItems)-1]
		nextCursor := ss.cursorManager.EncodeCursor(lastItem.UpdatedAt, lastItem.ID)

		// 재귀적으로 더 많은 항목 로드
		newPagination := utils.CursorPagination{
			Cursor: nextCursor,
			Limit:  pagination.Limit,
		}

		// 현재 검색 결과에 추가 결과 병합
		additionalResult, err := ss.SearchItems(userID, keyword, itemType, newPagination)
		if err == nil && len(additionalResult.Items) > 0 {
			authorizedItems = append(authorizedItems, additionalResult.Items...)
		}
	}

	// 더 많은 아이템이 있는지 확인
	hasMore := false
	var items []models.Item

	if len(authorizedItems) > pagination.Limit {
		hasMore = true
		items = authorizedItems[:pagination.Limit] // 요청된 수만큼만 반환
	} else {
		items = authorizedItems
	}

	// 더 많은 아이템이 있으면 다음 커서 생성
	nextCursor := ""
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		nextCursor = ss.cursorManager.EncodeCursor(lastItem.UpdatedAt, lastItem.ID)
	}

	return &ItemSearchResult{
		Items: items,
		PaginationResult: utils.PaginationResult{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

// SearchProjects 프로젝트만 검색하는 편의 함수입니다.
func (ss *SearchService) SearchProjects(userID string, keyword string, pagination utils.CursorPagination) (*ItemSearchResult, error) {
	return ss.SearchItems(userID, keyword, "project", pagination)
}

// SearchTasks 태스크만 검색하는 편의 함수입니다.
func (ss *SearchService) SearchTasks(userID string, keyword string, pagination utils.CursorPagination) (*ItemSearchResult, error) {
	return ss.SearchItems(userID, keyword, "task", pagination)
}
