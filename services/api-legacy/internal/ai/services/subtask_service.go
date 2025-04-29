package services

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"semo-server/internal/ai/executor"
	"semo-server/internal/ai/models"
	"semo-server/internal/ai/parsers/stream"
	"semo-server/internal/ai/parsers/string"
	"semo-server/internal/ai/streaming"
)

// SubtaskService handles generating subtasks
type SubtaskService struct {
	Executor     *executor.AIExecutor
	Logger       *zap.Logger
	RoleService  *RoleService
	StringParser *stringparser.SubtaskV5Parser
}

// NewSubtaskService creates a new SubtaskService
func NewSubtaskService(exec *executor.AIExecutor, logger *zap.Logger, roleService *RoleService) *SubtaskService {
	return &SubtaskService{
		Executor:     exec,
		Logger:       logger,
		RoleService:  roleService,
		StringParser: stringparser.NewSubtaskV5Parser(),
	}
}

// GenerateSubtasks generates subtasks for a task
func (s *SubtaskService) GenerateSubtasks(ctx context.Context, data map[string]any, streamChan chan<- string) (*models.SubtaskResponse, error) {
	// First, use the role service to get the role information
	roleResponse, err := s.RoleService.GenerateRole(ctx, data, streamChan)
	if err != nil {
		return nil, fmt.Errorf("failed to generate role: %w", err)
	}

	// Update data with role information
	data["Role"] = roleResponse.Role
	data["RoleMessage"] = roleResponse.SystemMessage

	// Create a streaming parser for this request
	streamParser := streamparser.NewSubtaskV5StreamParser(streamChan)
	eventSender := streaming.NewEventSender(streamChan)

	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-breakdown-todo", // Based on the lookup table
		Variables:   data,
		Temperature: 0.0,                       // From the original chain
		Model:       "google:gemini-2.0-flash", // From the original chain
		LineByLine:  true,
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
				// Process the chunk with the streaming parser
				if err := streamParser.ProcessChunk([]byte(chunk + "\n")); err != nil {
					s.Logger.Error("Error processing chunk", zap.Error(err))
				}

				// Also collect the output for later parsing
				outputBuffer.WriteString(chunk)
				outputBuffer.WriteString("\n")
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
				eventSender.SendError(err)
			}
		}

		// Exit when all channels are closed
		if outputCh == nil && errorCh == nil && execErrCh == nil {
			break
		}
	}

	// Finalize the streaming parser
	_ = streamParser.Finalize()

	// Check for execution error
	if execErr != nil {
		return nil, fmt.Errorf("error generating subtasks: %w", execErr)
	}

	// Parse the complete output using the string parser
	results, err := s.StringParser.Parse(outputBuffer.String())
	if err != nil {
		s.Logger.Error("Failed to parse subtask output", zap.Error(err))
		return nil, fmt.Errorf("failed to parse subtask output: %w", err)
	}

	eventSender.SendComplete("Subtask generation completed")

	return results, nil
}

// CheckNeedSubtask determines if a task needs to be broken down into subtasks
// This logic was deleted from chain_need_subtask.go according to the lookup table
func (s *SubtaskService) CheckNeedSubtask(ctx context.Context, data map[string]any, streamChan chan<- string) (bool, error) {
	// Always return true since the original logic was deleted
	if streamChan != nil {
		eventSender := streaming.NewEventSender(streamChan)
		eventSender.Send("need_subtask", fmt.Sprintf("%s|%s|%s",
			data["SelectedTodo"], "yes", "Breaking down the task for detailed analysis."))
	}

	return true, nil
}
