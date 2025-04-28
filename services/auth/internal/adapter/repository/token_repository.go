package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
	"gorm.io/gorm"
)

// TokenRepositoryImpl는 TokenRepository 인터페이스 구현체입니다.
type TokenRepositoryImpl struct {
	db *gorm.DB
}

// NewTokenRepository는 토큰 저장소 구현체를 생성합니다.
func NewTokenRepository(db *gorm.DB) repository.TokenRepository {
	return &TokenRepositoryImpl{db: db}
}

// ===== 토큰 관련 메서드 =====

// FindByID는 ID로 토큰을 조회합니다.
func (r *TokenRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Token, error) {
	var tokenModel model.TokenModel

	if err := r.db.WithContext(ctx).First(&tokenModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 토큰을 찾지 못함
		}
		return nil, err
	}

	return TokenFromModel(&tokenModel), nil
}

// FindByToken은 토큰 값으로 조회합니다.
func (r *TokenRepositoryImpl) FindByToken(ctx context.Context, token string) (*entity.Token, error) {
	var tokenModel model.TokenModel

	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&tokenModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return TokenFromModel(&tokenModel), nil
}

// FindByGroupID는 그룹 ID로 토큰 목록을 조회합니다.
func (r *TokenRepositoryImpl) FindByGroupID(ctx context.Context, groupID uint) ([]*entity.Token, error) {
	var tokenModels []model.TokenModel

	if err := r.db.WithContext(ctx).Where("group_id = ?", groupID).Find(&tokenModels).Error; err != nil {
		return nil, err
	}

	return TokensFromModels(tokenModels), nil
}

// Create는 새 토큰을 생성합니다.
func (r *TokenRepositoryImpl) Create(ctx context.Context, token *entity.Token) error {
	tokenModel := TokenToModel(token)

	if err := r.db.WithContext(ctx).Create(tokenModel).Error; err != nil {
		return err
	}

	// ID가 DB에서 생성된 경우 엔티티에 반영
	token.ID = tokenModel.ID
	return nil
}

// Update는 토큰 정보를 업데이트합니다.
func (r *TokenRepositoryImpl) Update(ctx context.Context, token *entity.Token) error {
	tokenModel := TokenToModel(token)

	return r.db.WithContext(ctx).Save(tokenModel).Error
}

// Delete는 토큰을 삭제합니다.
func (r *TokenRepositoryImpl) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TokenModel{}, id).Error
}

// ===== 토큰 그룹 관련 메서드 =====

// FindOrCreateTokenGroup은 토큰 그룹을 찾거나 생성합니다.
func (r *TokenRepositoryImpl) FindOrCreateTokenGroup(ctx context.Context, userID string) (*entity.TokenGroup, error) {
	var tokenGroupModel model.TokenGroupModel

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&tokenGroupModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 새 토큰 그룹 생성
			tokenGroupModel = model.TokenGroupModel{
				UserID: userID,
			}
			if createErr := r.db.WithContext(ctx).Create(&tokenGroupModel).Error; createErr != nil {
				return nil, fmt.Errorf("토큰 그룹 생성 실패: %w", createErr)
			}
		} else {
			return nil, err
		}
	}

	return TokenGroupFromModel(&tokenGroupModel), nil
}

// FindGroupByID는 ID로 토큰 그룹을 조회합니다.
func (r *TokenRepositoryImpl) FindGroupByID(ctx context.Context, id uint) (*entity.TokenGroup, error) {
	var tokenGroupModel model.TokenGroupModel

	if err := r.db.WithContext(ctx).First(&tokenGroupModel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 토큰 그룹을 찾지 못함
		}
		return nil, err
	}

	return TokenGroupFromModel(&tokenGroupModel), nil
}

// FindGroupsByUserID는 사용자 ID로 토큰 그룹 목록을 조회합니다.
func (r *TokenRepositoryImpl) FindGroupsByUserID(ctx context.Context, userID string) ([]*entity.TokenGroup, error) {
	var tokenGroupModels []model.TokenGroupModel

	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&tokenGroupModels).Error; err != nil {
		return nil, err
	}

	return TokenGroupsFromModels(tokenGroupModels), nil
}

// CreateGroup은 새 토큰 그룹을 생성합니다.
func (r *TokenRepositoryImpl) CreateGroup(ctx context.Context, tokenGroup *entity.TokenGroup) error {
	tokenGroupModel := TokenGroupToModel(tokenGroup)

	if err := r.db.WithContext(ctx).Create(tokenGroupModel).Error; err != nil {
		return err
	}

	// ID가 DB에서 생성된 경우 엔티티에 반영
	tokenGroup.ID = tokenGroupModel.ID
	return nil
}

// UpdateGroup은 토큰 그룹 정보를 업데이트합니다.
func (r *TokenRepositoryImpl) UpdateGroup(ctx context.Context, tokenGroup *entity.TokenGroup) error {
	tokenGroupModel := TokenGroupToModel(tokenGroup)

	return r.db.WithContext(ctx).Save(tokenGroupModel).Error
}

// DeleteGroup은 토큰 그룹을 삭제합니다.
func (r *TokenRepositoryImpl) DeleteGroup(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TokenGroupModel{}, id).Error
}

// DeleteByGroup은 그룹에 속한 모든 토큰을 삭제합니다.
func (r *TokenRepositoryImpl) DeleteByGroup(ctx context.Context, groupID uint) error {
	return r.db.WithContext(ctx).Where("group_id = ?", groupID).Delete(&model.TokenModel{}).Error
}

// ===== 매퍼 함수 =====

// TokenToModel은 토큰 엔티티를 DB 모델로 변환합니다.
func TokenToModel(token *entity.Token) *model.TokenModel {
	if token == nil {
		return nil
	}

	return &model.TokenModel{
		ID:        token.ID,
		GroupID:   token.GroupID,
		Token:     token.Token,
		TokenType: token.TokenType,
		ExpiresAt: token.ExpiresAt,
	}
}

// TokenFromModel은 DB 모델을 토큰 엔티티로 변환합니다.
func TokenFromModel(model *model.TokenModel) *entity.Token {
	if model == nil {
		return nil
	}

	return &entity.Token{
		ID:        model.ID,
		GroupID:   model.GroupID,
		Token:     model.Token,
		TokenType: model.TokenType,
		ExpiresAt: model.ExpiresAt,
	}
}

// TokensFromModels은 DB 모델 슬라이스를 토큰 엔티티 슬라이스로 변환합니다.
func TokensFromModels(models []model.TokenModel) []*entity.Token {
	tokens := make([]*entity.Token, len(models))
	for i, model := range models {
		tokens[i] = TokenFromModel(&model)
	}
	return tokens
}

// TokenGroupToModel은 토큰 그룹 엔티티를 DB 모델로 변환합니다.
func TokenGroupToModel(tokenGroup *entity.TokenGroup) *model.TokenGroupModel {
	if tokenGroup == nil {
		return nil
	}

	return &model.TokenGroupModel{
		ID:     tokenGroup.ID,
		UserID: tokenGroup.UserID,
		Name:   tokenGroup.Name,
		Device: tokenGroup.Device,
	}
}

// TokenGroupFromModel은 DB 모델을 토큰 그룹 엔티티로 변환합니다.
func TokenGroupFromModel(model *model.TokenGroupModel) *entity.TokenGroup {
	if model == nil {
		return nil
	}

	return &entity.TokenGroup{
		ID:        model.ID,
		UserID:    model.UserID,
		Name:      model.Name,
		Device:    model.Device,
		CreatedAt: model.CreatedAt,
	}
}

// TokenGroupsFromModels은 DB 모델 슬라이스를 토큰 그룹 엔티티 슬라이스로 변환합니다.
func TokenGroupsFromModels(models []model.TokenGroupModel) []*entity.TokenGroup {
	tokenGroups := make([]*entity.TokenGroup, len(models))
	for i, model := range models {
		tokenGroups[i] = TokenGroupFromModel(&model)
	}
	return tokenGroups
}
