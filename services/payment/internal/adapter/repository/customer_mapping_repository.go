package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/entity"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"gorm.io/gorm"
)

type customerMappingRepository struct {
	db *gorm.DB
}

func NewCustomerMappingRepository(db *gorm.DB) repository.CustomerMappingRepository {
	return &customerMappingRepository{
		db: db,
	}
}

// modelToEntity converts a model.CustomerMapping to entity.CustomerMapping
func (r *customerMappingRepository) modelToEntity(m *model.CustomerMapping) *entity.CustomerMapping {
	if m == nil {
		return nil
	}
	return &entity.CustomerMapping{
		ID:                 m.ID,
		Provider:           m.Provider,
		ProviderCustomerID: m.ProviderCustomerID,
		UniversalID:        m.UniversalID.String(),
		Email:              m.CustomerEmail,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

// entityToModel converts an entity.CustomerMapping to model.CustomerMapping
func (r *customerMappingRepository) entityToModel(e *entity.CustomerMapping) (*model.CustomerMapping, error) {
	if e == nil {
		return nil, nil
	}

	userUUID, err := uuid.Parse(e.UniversalID)
	if err != nil {
		return nil, err
	}

	return &model.CustomerMapping{
		ID:                 e.ID,
		Provider:           e.Provider,
		ProviderCustomerID: e.ProviderCustomerID,
		UniversalID:        userUUID,
		CustomerEmail:      e.Email,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}, nil
}

func (r *customerMappingRepository) Create(ctx context.Context, mapping *entity.CustomerMapping) error {
	modelMapping, err := r.entityToModel(mapping)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(modelMapping).Error
}

func (r *customerMappingRepository) GetByProviderCustomerID(ctx context.Context, provider string, providerCustomerID string) (*entity.CustomerMapping, error) {
	var mapping model.CustomerMapping
	err := r.db.WithContext(ctx).Where("provider = ? AND provider_customer_id = ?", provider, providerCustomerID).First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.modelToEntity(&mapping), nil
}

func (r *customerMappingRepository) GetByProviderAndUniversalID(ctx context.Context, provider string, universalID string) (*entity.CustomerMapping, error) {
	// Parse universalID to ensure it's a valid UUID
	universalUUID, err := uuid.Parse(universalID)
	if err != nil {
		return nil, err
	}

	var mapping model.CustomerMapping
	err = r.db.WithContext(ctx).Where("provider = ? AND universal_id = ?", provider, universalUUID).First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.modelToEntity(&mapping), nil
}

func (r *customerMappingRepository) Update(ctx context.Context, mapping *entity.CustomerMapping) error {
	modelMapping, err := r.entityToModel(mapping)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(modelMapping).Error
}
