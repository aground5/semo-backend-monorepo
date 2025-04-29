package services

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"semo-server/internal/ai/cache"
	"semo-server/internal/ai/executor"
	"semo-server/internal/ai/models"
	"semo-server/internal/ai/parsers/string"
	"semo-server/internal/ai/streaming"
)

// RoleService handles generating expert roles
type RoleService struct {
	Executor     *executor.AIExecutor
	Logger       *zap.Logger
	StringParser *stringparser.RoleParser
}

// NewRoleService creates a new RoleService
func NewRoleService(exec *executor.AIExecutor, logger *zap.Logger) *RoleService {
	return &RoleService{
		Executor:     exec,
		Logger:       logger,
		StringParser: stringparser.NewRoleParser(),
	}
}

// GenerateRole generates an expert role for a task
func (s *RoleService) GenerateRole(ctx context.Context, data map[string]any, streamChan chan<- string) (*models.RoleResponse, error) {
	// Check cache first
	var cachedResponse executor.AIExecutorResponse
	found, err := cache.Get(ctx, "llm_role", data, &cachedResponse)
	if err == nil && found {
		s.Logger.Info("Cache hit for role generation")

		// Parse the cached response
		roleResponse, err := s.StringParser.Parse(cachedResponse.Output)
		if err != nil {
			s.Logger.Error("Failed to parse cached role response", zap.Error(err))
			// Continue with generating a new role
		} else {
			// Send the role to the stream channel if provided
			if streamChan != nil {
				eventSender := streaming.NewEventSender(streamChan)
				eventSender.Send(models.EventRole, roleResponse.Role)
			}
			return roleResponse, nil
		}
	}

	// Cache miss or parse error, generate a new role
	s.Logger.Info("Cache miss for role generation, calling AI executor")

	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-expert-system-prompt", // Based on the lookup table
		Variables:   data,
		Temperature: 0.3,
		Model:       "openai:gpt-4o-mini", // From the original chain_role.go
		LineByLine:  true,
		UserId:      data["UserId"].(string),
		SessionId:   data["SessionId"].(string),
	}

	// Execute AI request
	var response *executor.AIExecutorResponse
	var execErr error

	if streamChan != nil {
		// If stream channel is provided, use it for streaming
		outputCh, errorCh, execErrCh := executor.NewAIExecutor().Execute(request)

		// Collect the response
		var outputLines []string
		var errors []string

		for {
			select {
			case line, ok := <-outputCh:
				if !ok {
					outputCh = nil
				} else {
					outputLines = append(outputLines, line)
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
				}
			}

			// Exit when all channels are closed
			if outputCh == nil && errorCh == nil && execErrCh == nil {
				break
			}
		}

		// Create the response
		response = &executor.AIExecutorResponse{
			Output:    strings.Join(outputLines, "\n"),
			Errors:    errors,
			ExecError: execErr,
		}
	} else {
		// If no stream channel, use regular execution
		response, execErr = s.Executor.ExecuteAndCollect(request)
		if execErr != nil {
			return nil, fmt.Errorf("error generating role: %w", execErr)
		}
	}

	// Check for errors
	if response.ExecError != nil {
		return nil, fmt.Errorf("error generating role: %w", response.ExecError)
	}

	// Parse the response
	roleResponse, err := s.StringParser.Parse(response.Output)
	if err != nil {
		s.Logger.Error("Failed to parse role response",
			zap.String("output", response.Output),
			zap.Error(err))

		// Return a default response on parse error
		roleResponse = &models.RoleResponse{
			Think:         "",
			Role:          "Expert Task Analyzer",
			SystemMessage: "Unable to parse role response.",
		}
	} else {
		// Cache the successful response
		if err := cache.Set(ctx, "llm_role", data, response); err != nil {
			s.Logger.Warn("Failed to cache role response", zap.Error(err))
		}
	}

	// Send the role to the stream channel if provided
	if streamChan != nil {
		eventSender := streaming.NewEventSender(streamChan)
		eventSender.Send(models.EventRole, roleResponse.Role)
	}

	return roleResponse, nil
}

// ClearRoleCache clears the role cache
func (s *RoleService) ClearRoleCache(ctx context.Context) error {
	return cache.ClearByPrefix(ctx, "llm_role")
}
