package services

import (
	"context"
	"fmt"
	"strings"

	"semo-server/internal/ai/executor"
	"semo-server/internal/ai/streaming"

	"go.uber.org/zap"
)

// DetailService handles generating details
type DetailService struct {
	Executor *executor.AIExecutor
	Logger   *zap.Logger
	//RoleService *RoleService
}

// NewDetailService creates a new DetailService
func NewDetailService(exec *executor.AIExecutor, logger *zap.Logger) *DetailService {
	return &DetailService{
		Executor: exec,
		Logger:   logger,
		//RoleService: roleService,
	}
}

// GenerateDetails generates details for a task
func (s *DetailService) GenerateDetails(ctx context.Context, data map[string]any, streamChan chan<- string) (string, error) {
	//// First, use the role service to get the role information
	//roleResponse, err := s.RoleService.GenerateRole(ctx, data, streamChan)
	//if err != nil {
	//	return "", fmt.Errorf("failed to generate role: %w", err)
	//}
	//
	//// Update data with role information
	//data["Role"] = roleResponse.Role
	//data["RoleMessage"] = roleResponse.SystemMessage

	// Create a streaming parser for this request
	eventSender := streaming.NewEventSender(streamChan)
	eventSender.Send(EventPreQuestionStart, "Starting pre-question generation")

	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-detailed-document-of-todo", // Based on the lookup table
		Variables:   data,
		Temperature: 0.7,                       // From the original chain
		Model:       "google:gemini-2.0-flash", // From the original chain
		LineByLine:  false,
		UserId:      data["UserId"].(string),
		SessionId:   data["SessionId"].(string),
	}

	// Execute the request with streaming
	outputCh, errorCh, execErrCh := executor.NewAIExecutor().Execute(request)

	// Process the streaming output
	var outputBuffer strings.Builder
	var errors []string
	var execErr error

	for {
		select {
		case chunk, ok := <-outputCh:
			if !ok {
				outputCh = nil
			} else {
				// Send token event
				eventSender.Send(EventPreQuestionToken, chunk)

				// Also collect the output
				outputBuffer.WriteString(chunk)
			}
		case errLine, ok := <-errorCh:
			if !ok {
				errorCh = nil
			} else {
				errors = append(errors, errLine)
				s.Logger.Warn("Error from AI executor", zap.String("error", errLine))
			}
		case err, ok := <-execErrCh:
			if !ok {
				execErrCh = nil
			} else {
				execErr = err
				s.Logger.Error("AI executor error", zap.Error(err))
				eventSender.Send(EventPreQuestionError, fmt.Sprintf("Error generating pre-questions: %v", err))
			}
		}

		// Exit when all channels are closed
		if outputCh == nil && errorCh == nil && execErrCh == nil {
			break
		}
	}

	// Check for execution error
	if execErr != nil {
		return "", fmt.Errorf("error generating pre-questions: %w", execErr)
	}

	// Send completion event
	eventSender.Send(EventPreQuestionEnd, "Pre-question generation completed")

	return outputBuffer.String(), nil
}
