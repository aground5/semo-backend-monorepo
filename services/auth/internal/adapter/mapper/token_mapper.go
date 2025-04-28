package mapper

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"
)

// TokenToModel 토큰 엔티티를 DB 모델로 변환
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

// TokenFromModel DB 모델을 토큰 엔티티로 변환
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

// TokensFromModels DB 모델 슬라이스를 토큰 엔티티 슬라이스로 변환
func TokensFromModels(models []model.TokenModel) []*entity.Token {
	tokens := make([]*entity.Token, len(models))
	for i, model := range models {
		tokens[i] = TokenFromModel(&model)
	}
	return tokens
}
