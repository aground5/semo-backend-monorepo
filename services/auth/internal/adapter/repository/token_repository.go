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

type TokenRepositoryImpl struct {
	db *gorm.DB
}

// NewTokenRepository 토큰 저장소 구현체 생성
func NewTokenRepository(db *gorm.DB) repository.TokenRepository {
	return &TokenRepositoryImpl{db: db}
}

// FindByID ID로 토큰 조회
func (r *TokenRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Token, error) {
	var tokenModel model.TokenModel

	if err := r.db.WithContext(ctx).First(&tokenModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 토큰을 찾지 못함
		}
		return nil, err
	}

	return mapper.TokenFromModel(&tokenModel), nil
}

// FindByToken 토큰 값으로 조회
func (r *TokenRepositoryImpl) FindByToken(ctx context.Context, token string) (*entity.Token, error) {
	var tokenModel model.TokenModel

	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&tokenModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return mapper.TokenFromModel(&tokenModel), nil
}

// FindByGroupID 그룹 ID로 토큰 목록 조회
func (r *TokenRepositoryImpl) FindByGroupID(ctx context.Context, groupID uint) ([]*entity.Token, error) {
	var tokenModels []model.TokenModel

	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&tokenModels).Error; err != nil {
		return nil, err
	}

	return mapper.TokensFromModels(tokenModels), nil
}

// Create 새 토큰 생성
func (r *TokenRepositoryImpl) Create(ctx context.Context, token *entity.Token) error {
	tokenModel := mapper.TokenToModel(token)

	if err := r.db.WithContext(ctx).Create(tokenModel).Error; err != nil {
		return err
	}

	// ID가 DB에서 생성된 경우 엔티티에 반영
	token.ID = tokenModel.ID
	return nil
}

// Update 토큰 정보 업데이트
func (r *TokenRepositoryImpl) Update(ctx context.Context, token *entity.Token) error {
	tokenModel := mapper.TokenToModel(token)

	return r.db.WithContext(ctx).Save(tokenModel).Error
}

// Delete 토큰 삭제
func (r *TokenRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TokenModel{}, id).Error
}
