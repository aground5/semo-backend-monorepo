package grpc

import (
	// "context"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/usecase"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	usecase *usecase.PaymentUsecase
	logger  *zap.Logger
}

func NewPaymentHandler(usecase *usecase.PaymentUsecase, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{
		usecase: usecase,
		logger:  logger,
	}
}

// gRPC handler methods would be implemented here
// func (h *PaymentHandler) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
//     // Implementation
// }