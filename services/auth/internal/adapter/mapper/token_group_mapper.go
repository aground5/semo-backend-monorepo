package mapper

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
)

// TokenGroupToModel 토큰 그룹 엔티티를 DB 모델로 변환
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

// TokenGroupFromModel DB 모델을 토큰 그룹 엔티티로 변환
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

// TokenGroupsFromModels DB 모델 슬라이스를 토큰 그룹 엔티티 슬라이스로 변환
func TokenGroupsFromModels(models []model.TokenGroupModel) []*entity.TokenGroup {
	tokenGroups := make([]*entity.TokenGroup, len(models))
	for i, model := range models {
		tokenGroups[i] = TokenGroupFromModel(&model)
	}
	return tokenGroups
}
