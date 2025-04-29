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

// DependencyService handles generating task dependencies
type DependencyService struct {
	Executor     *executor.AIExecutor
	Logger       *zap.Logger
	StringParser *stringparser.DependencyParser
}

// NewDependencyService creates a new DependencyService
func NewDependencyService(exec *executor.AIExecutor, logger *zap.Logger) *DependencyService {
	return &DependencyService{
		Executor:     exec,
		Logger:       logger,
		StringParser: stringparser.NewDependencyParser(),
	}
}

// GenerateTaskDependency generates dependencies between tasks
func (s *DependencyService) GenerateTaskDependency(ctx context.Context, data map[string]any, streamChan chan<- string) (*models.TaskDependencyResponse, error) {
	// Check cache first
	var cachedResponse executor.AIExecutorResponse
	found, err := cache.Get(ctx, "task_dependency", data, &cachedResponse)
	if err == nil && found {
		s.Logger.Info("Cache hit for task dependency generation")

		// Parse the cached response
		dependencies, err := s.StringParser.Parse(cachedResponse.Output)
		if err != nil {
			s.Logger.Error("Failed to parse cached dependency response", zap.Error(err))
			// Continue with generating new dependencies
		} else {
			// Send events to the stream channel if provided
			if streamChan != nil {
				eventSender := streaming.NewEventSender(streamChan)
				eventSender.Send("dependency_start", "Starting task dependency analysis")

				for _, task := range dependencies.Tasks {
					dependencyStr := strings.Join(task.Dependencies, ", ")
					if dependencyStr == "" {
						dependencyStr = "None"
					}

					eventSender.Send("dependency_task", fmt.Sprintf("%s|%s|%s",
						task.ID, task.Name, dependencyStr))
				}

				eventSender.Send("dependency_complete", "Task dependency analysis completed")
			}

			return dependencies, nil
		}
	}

	// Cache miss or parse error, generate new dependencies
	s.Logger.Info("Cache miss for task dependency generation, calling AI executor")

	// Create AI executor request
	request := executor.AIExecutorRequest{
		PromptName:  "semo-generate-dependency", // Based on the lookup table
		Variables:   data,
		Temperature: 0.2,                       // From the original chain
		Model:       "google:gemini-2.0-flash", // From the original chain
		UserId:      data["UserId"].(string),
		SessionId:   data["SessionId"].(string),
	}

	// Initialize event sender if stream channel is provided
	var eventSender *streaming.EventSender
	if streamChan != nil {
		eventSender = streaming.NewEventSender(streamChan)
		eventSender.Send("dependency_start", "Starting task dependency analysis")
	}

	// Execute AI request
	response, err := executor.NewAIExecutor().ExecuteAndCollect(request)
	if err != nil {
		if eventSender != nil {
			eventSender.Send("dependency_error", fmt.Sprintf("Error analyzing task dependencies: %v", err))
		}
		return nil, fmt.Errorf("error generating task dependencies: %w", err)
	}

	// Parse the response
	dependencies, err := s.StringParser.Parse(response.Output)
	if err != nil {
		if eventSender != nil {
			eventSender.Send("dependency_error", fmt.Sprintf("Error parsing dependency response: %v", err))
		}
		return nil, fmt.Errorf("failed to parse task dependency response: %w", err)
	}

	// Cache the successful response
	if err := cache.Set(ctx, "task_dependency", data, response); err != nil {
		s.Logger.Warn("Failed to cache task dependency response", zap.Error(err))
	}

	// Send events to the stream channel if provided
	if eventSender != nil {
		for _, task := range dependencies.Tasks {
			dependencyStr := strings.Join(task.Dependencies, ", ")
			if dependencyStr == "" {
				dependencyStr = "None"
			}

			eventSender.Send("dependency_task", fmt.Sprintf("%s|%s|%s",
				task.ID, task.Name, dependencyStr))
		}

		eventSender.Send("dependency_complete", "Task dependency analysis completed")
	}

	return dependencies, nil
}

// ClearTaskDependencyCache clears the task dependency cache
func (s *DependencyService) ClearTaskDependencyCache(ctx context.Context) error {
	return cache.ClearByPrefix(ctx, "task_dependency")
}
