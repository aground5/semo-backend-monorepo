package repository

import (
	"context"
	"errors"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/domain/repository"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/infrastructure/db/model"

	"gorm.io/gorm"
)

type UserRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository 사용자 레포지토리 구현체 생성
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &UserRepositoryImpl{db: db}
}

// 도메인 엔티티를 DB 모델로 변환
func toUserModel(user *entity.User) *model.UserModel {
	return &model.UserModel{
		ID:                user.ID,
		Username:          user.Username,
		Name:              user.Name,
		Email:             user.Email,
		Password:          user.Password,
		Hash:              user.Hash,
		EmailVerified:     user.EmailVerified,
		AccountStatus:     user.AccountStatus,
		LastLoginAt:       user.LastLoginAt,
		LastLoginIP:       user.LastLoginIP,
		FailedLoginCount:  user.FailedLoginCount,
		PasswordChangedAt: user.PasswordChangedAt,
	}
}

// DB 모델을 도메인 엔티티로 변환
func toUserEntity(model *model.UserModel) *entity.User {
	return &entity.User{
		ID:                model.ID,
		Username:          model.Username,
		Name:              model.Name,
		Email:             model.Email,
		Password:          model.Password,
		Hash:              model.Hash,
		EmailVerified:     model.EmailVerified,
		AccountStatus:     model.AccountStatus,
		LastLoginAt:       model.LastLoginAt,
		LastLoginIP:       model.LastLoginIP,
		FailedLoginCount:  model.FailedLoginCount,
		PasswordChangedAt: model.PasswordChangedAt,
	}
}

// FindByID ID로 사용자 조회
func (r *UserRepositoryImpl) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var userModel model.UserModel

	if err := r.db.WithContext(ctx).First(&userModel, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 사용자를 찾지 못함
		}
		return nil, err
	}

	return toUserEntity(&userModel), nil
}

// FindByEmail 이메일로 사용자 조회
func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var userModel model.UserModel

	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&userModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return toUserEntity(&userModel), nil
}

// FindByUsername 사용자명으로 사용자 조회
func (r *UserRepositoryImpl) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var userModel model.UserModel

	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&userModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return toUserEntity(&userModel), nil
}

// Create 새 사용자 생성
func (r *UserRepositoryImpl) Create(ctx context.Context, user *entity.User) error {
	userModel := toUserModel(user)

	if err := r.db.WithContext(ctx).Create(userModel).Error; err != nil {
		return err
	}

	// ID가 DB에서 생성된 경우 엔티티에 반영
	user.ID = userModel.ID
	return nil
}

// Update 사용자 정보 업데이트
func (r *UserRepositoryImpl) Update(ctx context.Context, user *entity.User) error {
	userModel := toUserModel(user)

	return r.db.WithContext(ctx).Save(userModel).Error
}

// Delete 사용자 삭제
func (r *UserRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.UserModel{}, "id = ?", id).Error
}
