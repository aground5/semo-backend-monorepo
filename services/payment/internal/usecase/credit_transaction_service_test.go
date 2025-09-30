package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/dto"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/model"
	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
)

// MockCreditTransactionRepository is a mock implementation of CreditTransactionRepository
type MockCreditTransactionRepository struct {
	mock.Mock
}

func (m *MockCreditTransactionRepository) GetTransactions(ctx context.Context, filters dto.TransactionFilters) ([]model.CreditTransaction, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]model.CreditTransaction), args.Error(1)
}

func (m *MockCreditTransactionRepository) CountTransactions(ctx context.Context, filters dto.TransactionFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCreditTransactionRepository) GetCreditBalance(ctx context.Context, universalID uuid.UUID, serviceProvider string) (*model.UserCreditBalance, error) {
	args := m.Called(ctx, universalID, serviceProvider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserCreditBalance), args.Error(1)
}

func TestCreditTransactionService_GetUserTransactionHistory(t *testing.T) {
	logger := zap.NewNop()
	universalID := uuid.New()
	ctx := context.Background()

	t.Run("successful transaction retrieval", func(t *testing.T) {
		// Setup mock repository
		mockRepo := new(MockCreditTransactionRepository)
		service := usecase.NewCreditTransactionService(mockRepo, logger, model.ServiceProviderSemo)

		// Create test data
		now := time.Now()
		transactions := []model.CreditTransaction{
			{
				ID:              1,
				UniversalID:     universalID,
				TransactionType: model.TransactionTypeCreditUsage,
				Amount:          decimal.NewFromFloat(10.00),
				BalanceAfter:    decimal.NewFromFloat(90.00),
				Description:     "API usage for feature X",
				CreatedAt:       now,
			},
			{
				ID:              2,
				UniversalID:     universalID,
				TransactionType: model.TransactionTypeCreditAllocation,
				Amount:          decimal.NewFromFloat(100.00),
				BalanceAfter:    decimal.NewFromFloat(100.00),
				Description:     "Monthly credit allocation",
				CreatedAt:       now.Add(-24 * time.Hour),
			},
		}

		filters := dto.TransactionFilters{
			UserID: universalID,
			Limit:  20,
			Offset: 0,
		}

		mockRepo.On("GetTransactions", ctx, filters).Return(transactions, nil)
		mockRepo.On("CountTransactions", ctx, filters).Return(int64(2), nil)

		// Execute
		result, err := service.GetUserTransactionHistory(ctx, universalID, dto.TransactionFilters{})

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Transactions, 2)
		assert.Equal(t, "credit_usage", result.Transactions[0].TransactionType)
		assert.Equal(t, "-10", result.Transactions[0].Amount)
		assert.Equal(t, "90", result.Transactions[0].BalanceAfter)
		assert.Equal(t, int64(2), result.Pagination.Total)
		assert.False(t, result.Pagination.HasMore)

		mockRepo.AssertExpectations(t)
	})

	t.Run("pagination with has_more", func(t *testing.T) {
		mockRepo := new(MockCreditTransactionRepository)
		service := usecase.NewCreditTransactionService(mockRepo, logger, model.ServiceProviderSemo)

		filters := dto.TransactionFilters{
			UserID: universalID,
			Limit:  10,
			Offset: 0,
		}

		mockRepo.On("GetTransactions", ctx, filters).Return([]model.CreditTransaction{}, nil)
		mockRepo.On("CountTransactions", ctx, filters).Return(int64(25), nil)

		result, err := service.GetUserTransactionHistory(ctx, universalID, dto.TransactionFilters{
			Limit:  10,
			Offset: 0,
		})

		assert.NoError(t, err)
		assert.True(t, result.Pagination.HasMore)
		assert.Equal(t, int64(25), result.Pagination.Total)

		mockRepo.AssertExpectations(t)
	})

	t.Run("truncate long descriptions", func(t *testing.T) {
		mockRepo := new(MockCreditTransactionRepository)
		service := usecase.NewCreditTransactionService(mockRepo, logger, model.ServiceProviderSemo)

		longDescription := "This is a very long description that exceeds fifty characters and should be truncated"
		transactions := []model.CreditTransaction{
			{
				ID:              1,
				UniversalID:     universalID,
				TransactionType: model.TransactionTypeCreditUsage,
				Amount:          decimal.NewFromFloat(5.00),
				BalanceAfter:    decimal.NewFromFloat(95.00),
				Description:     longDescription,
				CreatedAt:       time.Now(),
			},
		}

		filters := dto.TransactionFilters{
			UserID: universalID,
			Limit:  20,
			Offset: 0,
		}

		mockRepo.On("GetTransactions", ctx, filters).Return(transactions, nil)
		mockRepo.On("CountTransactions", ctx, filters).Return(int64(1), nil)

		result, err := service.GetUserTransactionHistory(ctx, universalID, dto.TransactionFilters{})

		assert.NoError(t, err)
		assert.Equal(t, "This is a very long description that exceeds fi...", result.Transactions[0].Description)
		assert.Equal(t, 50, len(result.Transactions[0].Description))

		mockRepo.AssertExpectations(t)
	})
}

func TestCreditTransactionService_GetCreditBalance(t *testing.T) {
	logger := zap.NewNop()
	universalID := uuid.New()
	ctx := context.Background()

	t.Run("successful balance retrieval", func(t *testing.T) {
		mockRepo := new(MockCreditTransactionRepository)
		service := usecase.NewCreditTransactionService(mockRepo, logger, model.ServiceProviderSemo)

		balance := &model.UserCreditBalance{
			UniversalID:    universalID,
			CurrentBalance: decimal.NewFromFloat(150.50),
		}

		mockRepo.On("GetCreditBalance", ctx, universalID, model.ServiceProviderSemo).Return(balance, nil)

		result, err := service.GetCreditBalance(ctx, universalID)

		assert.NoError(t, err)
		assert.Equal(t, "150.5", result)

		mockRepo.AssertExpectations(t)
	})

	t.Run("no balance found returns zero", func(t *testing.T) {
		mockRepo := new(MockCreditTransactionRepository)
		service := usecase.NewCreditTransactionService(mockRepo, logger, model.ServiceProviderSemo)

		mockRepo.On("GetCreditBalance", ctx, universalID, model.ServiceProviderSemo).Return(nil, nil)

		result, err := service.GetCreditBalance(ctx, universalID)

		assert.NoError(t, err)
		assert.Equal(t, "0.00", result)

		mockRepo.AssertExpectations(t)
	})
}
