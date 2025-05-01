package item

import (
	"context"
	"errors"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/domain/repository"
)

// UseCase 아이템 유스케이스 인터페이스
type UseCase interface {
	// CreateItem 아이템 생성
	CreateItem(ctx context.Context, name, contents string, itemType entity.ItemType, createdBy string, parentID *string) (*entity.Item, error)

	// GetItemByID ID로 아이템 조회
	GetItemByID(ctx context.Context, id string) (*entity.Item, error)

	// GetChildItems 자식 아이템 목록 조회
	GetChildItems(ctx context.Context, parentID string, limit, offset int) ([]*entity.Item, int64, error)

	// UpdateItem 아이템 정보 업데이트
	UpdateItem(ctx context.Context, id, name, contents string) (*entity.Item, error)

	// DeleteItem 아이템 삭제
	DeleteItem(ctx context.Context, id string) error

	// MoveItem 아이템 위치 이동
	MoveItem(ctx context.Context, id string, parentID *string, position float64) error

	// UpdateItemObjective 아이템 목표 업데이트
	UpdateItemObjective(ctx context.Context, id, objective string) error

	// UpdateItemDeliverable 아이템 예상 결과물 업데이트
	UpdateItemDeliverable(ctx context.Context, id, deliverable string) error

	// UpdateItemRole 아이템 역할 업데이트
	UpdateItemRole(ctx context.Context, id, role string) error

	// GetRootItems 루트 아이템 목록 조회
	GetRootItems(ctx context.Context, createdBy string, limit, offset int) ([]*entity.Item, int64, error)

	// GetProjectsByUser 사용자가 생성한 프로젝트 목록 조회
	GetProjectsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Item, int64, error)

	// GetTasksByProject 프로젝트에 속한 태스크 목록 조회
	GetTasksByProject(ctx context.Context, projectID string, limit, offset int) ([]*entity.Item, int64, error)

	// AddDependency 아이템에 의존성 추가
	AddDependency(ctx context.Context, itemID, dependencyID string) error

	// RemoveDependency 아이템에서 의존성 제거
	RemoveDependency(ctx context.Context, itemID, dependencyID string) error

	// GetDependencies 아이템의 의존성 목록 조회
	GetDependencies(ctx context.Context, itemID string) ([]*entity.Item, error)
}

type useCase struct {
	itemRepo     repository.ItemRepository
	activityRepo repository.ActivityRepository
}

// NewUseCase 아이템 유스케이스 생성
func NewUseCase(itemRepo repository.ItemRepository, activityRepo repository.ActivityRepository) UseCase {
	return &useCase{
		itemRepo:     itemRepo,
		activityRepo: activityRepo,
	}
}

// CreateItem 아이템 생성
func (uc *useCase) CreateItem(ctx context.Context, name, contents string, itemType entity.ItemType, createdBy string, parentID *string) (*entity.Item, error) {
	item, err := entity.NewItem(name, contents, itemType, createdBy, parentID)
	if err != nil {
		return nil, err
	}

	if err := uc.itemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	// 활동 기록
	activity := &repository.ActivityInfo{
		Type:        repository.ActivityTypeCreate,
		ItemID:      item.ID,
		ProfileID:   createdBy,
		Description: "아이템 생성",
	}

	if err := uc.activityRepo.Create(ctx, activity); err != nil {
		// 활동 기록 실패는 치명적이지 않으므로 로그만 남기고 계속 진행
		// log.Printf("활동 기록 실패: %v", err)
	}

	return item, nil
}

// GetItemByID ID로 아이템 조회
func (uc *useCase) GetItemByID(ctx context.Context, id string) (*entity.Item, error) {
	return uc.itemRepo.FindByID(ctx, id)
}

// GetChildItems 자식 아이템 목록 조회
func (uc *useCase) GetChildItems(ctx context.Context, parentID string, limit, offset int) ([]*entity.Item, int64, error) {
	items, err := uc.itemRepo.FindByParentID(ctx, parentID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := uc.itemRepo.CountByParentID(ctx, parentID)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

// UpdateItem 아이템 정보 업데이트
func (uc *useCase) UpdateItem(ctx context.Context, id, name, contents string) (*entity.Item, error) {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, errors.New("아이템을 찾을 수 없습니다")
	}

	item.Update(name, contents)

	if err := uc.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	// 활동 기록
	activity := &repository.ActivityInfo{
		Type:        repository.ActivityTypeUpdate,
		ItemID:      item.ID,
		ProfileID:   item.CreatedBy, // 실제로는 현재 사용자 ID를 사용해야 함
		Description: "아이템 업데이트",
	}

	if err := uc.activityRepo.Create(ctx, activity); err != nil {
		// 활동 기록 실패는 치명적이지 않으므로 로그만 남기고 계속 진행
	}

	return item, nil
}

// DeleteItem 아이템 삭제
func (uc *useCase) DeleteItem(ctx context.Context, id string) error {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("아이템을 찾을 수 없습니다")
	}

	if err := uc.itemRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 활동 기록
	activity := &repository.ActivityInfo{
		Type:        repository.ActivityTypeDelete,
		ItemID:      id,
		ProfileID:   item.CreatedBy, // 실제로는 현재 사용자 ID를 사용해야 함
		Description: "아이템 삭제",
	}

	if err := uc.activityRepo.Create(ctx, activity); err != nil {
		// 활동 기록 실패는 치명적이지 않으므로 로그만 남기고 계속 진행
	}

	return nil
}

// MoveItem 아이템 위치 이동
func (uc *useCase) MoveItem(ctx context.Context, id string, parentID *string, position float64) error {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("아이템을 찾을 수 없습니다")
	}

	item.MoveToParent(parentID)
	item.SetPosition(position)

	return uc.itemRepo.Update(ctx, item)
}

// UpdateItemObjective 아이템 목표 업데이트
func (uc *useCase) UpdateItemObjective(ctx context.Context, id, objective string) error {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("아이템을 찾을 수 없습니다")
	}

	item.SetObjective(objective)

	return uc.itemRepo.Update(ctx, item)
}

// UpdateItemDeliverable 아이템 예상 결과물 업데이트
func (uc *useCase) UpdateItemDeliverable(ctx context.Context, id, deliverable string) error {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("아이템을 찾을 수 없습니다")
	}

	item.SetDeliverable(deliverable)

	return uc.itemRepo.Update(ctx, item)
}

// UpdateItemRole 아이템 역할 업데이트
func (uc *useCase) UpdateItemRole(ctx context.Context, id, role string) error {
	item, err := uc.itemRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if item == nil {
		return errors.New("아이템을 찾을 수 없습니다")
	}

	item.SetRole(role)

	return uc.itemRepo.Update(ctx, item)
}

// GetRootItems 루트 아이템 목록 조회
func (uc *useCase) GetRootItems(ctx context.Context, createdBy string, limit, offset int) ([]*entity.Item, int64, error) {
	items, err := uc.itemRepo.FindRootItems(ctx, createdBy, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := uc.itemRepo.CountRootItems(ctx, createdBy)
	if err != nil {
		return nil, 0, err
	}

	return items, count, nil
}

// GetProjectsByUser 사용자가 생성한 프로젝트 목록 조회
func (uc *useCase) GetProjectsByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Item, int64, error) {
	projects, err := uc.itemRepo.FindProjectsByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// 총 개수를 반환하는 메서드가 없으므로 임시로 프로젝트 수를 반환
	return projects, int64(len(projects)), nil
}

// GetTasksByProject 프로젝트에 속한 태스크 목록 조회
func (uc *useCase) GetTasksByProject(ctx context.Context, projectID string, limit, offset int) ([]*entity.Item, int64, error) {
	tasks, err := uc.itemRepo.FindTasksByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// 총 개수를 반환하는 메서드가 없으므로 임시로 태스크 수를 반환
	return tasks, int64(len(tasks)), nil
}

// AddDependency 아이템에 의존성 추가
func (uc *useCase) AddDependency(ctx context.Context, itemID, dependencyID string) error {
	return uc.itemRepo.AddDependency(ctx, itemID, dependencyID)
}

// RemoveDependency 아이템에서 의존성 제거
func (uc *useCase) RemoveDependency(ctx context.Context, itemID, dependencyID string) error {
	return uc.itemRepo.RemoveDependency(ctx, itemID, dependencyID)
}

// GetDependencies 아이템의 의존성 목록 조회
func (uc *useCase) GetDependencies(ctx context.Context, itemID string) ([]*entity.Item, error) {
	return uc.itemRepo.FindDependencies(ctx, itemID)
}
