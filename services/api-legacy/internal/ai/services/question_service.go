package services

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"semo-server/internal/ai/executor"
	"semo-server/internal/ai/streaming"
)

// QuestionService handles generating and understanding questions
type QuestionService struct {
	Executor    *executor.AIExecutor
	Logger      *zap.Logger
	RoleService *RoleService
}

// NewQuestionService creates a new QuestionService
func NewQuestionService(exec *executor.AIExecutor, logger *zap.Logger, roleService *RoleService) *QuestionService {
	return &QuestionService{
		Executor:    exec,
		Logger:      logger,
		RoleService: roleService,
	}
}

// Event type constants for pre-question generation
const (
	EventPreQuestionStart = "pre_question_start"
	EventPreQuestionToken = "pre_question_token"
	EventPreQuestionEnd   = "pre_question_end"
	EventPreQuestionError = "pre_question_error"
)

// GeneratePreQuestions generates pre-questions for a task
func (s *QuestionService) GeneratePreQuestions(ctx context.Context, data map[string]any, streamChan chan<- string) (string, error) {
	// First, use the role service to get the role information
	roleResponse, err := s.RoleService.GenerateRole(ctx, data, streamChan)
	if err != nil {
		return "", fmt.Errorf("failed to generate role: %w", err)
	}

	// Update data with role information
	data["Role"] = roleResponse.Role
	data["RoleMessage"] = roleResponse.SystemMessage

	// Initialize event sender
	eventSender := streaming.NewEventSender(streamChan)
	eventSender.Send(EventPreQuestionStart, "Starting pre-question generation")

	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-question-with-think", // Based on the lookup table
		Variables:   data,
		Temperature: 0.8,                       // From the original chain
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

// UnderstandQuestion analyzes a question to understand its intent
func (s *QuestionService) UnderstandQuestion(ctx context.Context, data map[string]any) (string, error) {
	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-understand-question", // Based on the lookup table
		Variables:   data,
		Temperature: 0.7,                  // From the original chain
		Model:       "openai:gpt-4o-mini", // From the original chain
	}

	// Execute AI request
	response, err := s.Executor.ExecuteAndCollect(request)
	if err != nil {
		return "", fmt.Errorf("error understanding question: %w", err)
	}

	// Check for errors
	if response.ExecError != nil {
		return "", fmt.Errorf("error understanding question: %w", response.ExecError)
	}

	return response.Output, nil
}
