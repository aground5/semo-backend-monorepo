package repository

import (
	"context"
	"errors"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/adapter/mapper"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
	"gorm.io/gorm"
)

type TokenGroupRepositoryImpl struct {
	db *gorm.DB
}

// NewTokenGroupRepository 토큰 그룹 저장소 구현체 생성
func NewTokenGroupRepository(db *gorm.DB) repository.TokenGroupRepository {
	return &TokenGroupRepositoryImpl{db: db}
}

// FindByID ID로 토큰 그룹 조회
func (r *TokenGroupRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.TokenGroup, error) {
	var tokenGroupModel model.TokenGroupModel

	if err := r.db.WithContext(ctx).First(&tokenGroupModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 토큰 그룹을 찾지 못함
		}
		return nil, err
	}

	return mapper.TokenGroupFromModel(&tokenGroupModel), nil
}

// FindByUserID 사용자 ID로 토큰 그룹 목록 조회
func (r *TokenGroupRepositoryImpl) FindByUserID(ctx context.Context, userID string) ([]*entity.TokenGroup, error) {
	var tokenGroupModels []model.TokenGroupModel

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&tokenGroupModels).Error; err != nil {
		return nil, err
	}

	return mapper.TokenGroupsFromModels(tokenGroupModels), nil
}

// Create 새 토큰 그룹 생성
func (r *TokenGroupRepositoryImpl) Create(ctx context.Context, tokenGroup *entity.TokenGroup) error {
	tokenGroupModel := mapper.TokenGroupToModel(tokenGroup)

	if err := r.db.WithContext(ctx).Create(tokenGroupModel).Error; err != nil {
		return err
	}

	// ID가 DB에서 생성된 경우 엔티티에 반영
	tokenGroup.ID = tokenGroupModel.ID
	return nil
}

// Update 토큰 그룹 정보 업데이트
func (r *TokenGroupRepositoryImpl) Update(ctx context.Context, tokenGroup *entity.TokenGroup) error {
	tokenGroupModel := mapper.TokenGroupToModel(tokenGroup)

	return r.db.WithContext(ctx).Save(tokenGroupModel).Error
}

// Delete 토큰 그룹 삭제
func (r *TokenGroupRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TokenGroupModel{}, id).Error
}
