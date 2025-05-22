package logics

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"semo-server/internal/ai/collectors"
	"semo-server/internal/ai/executor"
	"semo-server/internal/ai/formatters"
	"semo-server/internal/ai/services"
	"semo-server/internal/models"
	"semo-server/internal/repositories"

	"go.uber.org/zap"
)

type GenerateSubtaskRequest struct {
	Task   string `json:"task"`
	TaskID string `json:"task_id"`
	Answer string `json:"answer"`
}

type LLMService struct {
	taskService     *TaskService
	userTestService *UserTestService
	logger          *zap.Logger

	// AI services
	subtaskService    *services.SubtaskService
	dependencyService *services.DependencyService
	questionService   *services.QuestionService
	detailService     *services.DetailService
	// Collectors
	taskCollector     *collectors.TaskCollector
	userDataCollector *collectors.UserDataCollector
	contextCollector  *collectors.ContextCollector
}

func NewLLMService(db *gorm.DB, taskService *TaskService, userTestService *UserTestService, logger *zap.Logger) *LLMService {
	// Create executor
	aiExecutor := executor.NewAIExecutor()

	// Create services
	roleService := services.NewRoleService(aiExecutor, logger)

	return &LLMService{
		taskService:     taskService,
		userTestService: userTestService,

		subtaskService:    services.NewSubtaskService(aiExecutor, logger, roleService),
		dependencyService: services.NewDependencyService(aiExecutor, logger),
		questionService:   services.NewQuestionService(aiExecutor, logger, roleService),
		detailService:     services.NewDetailService(aiExecutor, logger),

		taskCollector:     collectors.NewTaskCollector(),
		userDataCollector: collectors.NewUserDataCollector(),
		contextCollector:  collectors.NewContextCollector(),
	}
}

// sendEvent is a helper function to format and send SSE events
func (cs *LLMService) sendEvent(streamChan chan<- string, eventType string, data string) {
	streamChan <- fmt.Sprintf("event: %s", eventType)

	// Create a map with the data
	wrapper := map[string]interface{}{
		"event": eventType,
		"v":     data,
	}

	// Marshal to JSON with proper encoding
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		// Handle error
		jsonBytes = []byte(`{"v":"Error encoding JSON"}`)
	}

	streamChan <- fmt.Sprintf("data: %s", string(jsonBytes))
}

// sendEvent is a helper function to format and send SSE events
func (cs *LLMService) sendEventWithAdditional(streamChan chan<- string, eventType string, data string, additionalData map[string]interface{}) {
	streamChan <- fmt.Sprintf("event: %s", eventType)

	// Create a map with the data
	wrapper := map[string]interface{}{
		"event": eventType,
		"v":     data,
	}

	// Merge additionalData into wrapper
	for key, value := range additionalData {
		wrapper[key] = value
	}

	// Marshal to JSON with proper encoding
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		// Handle error
		jsonBytes = []byte(`{"v":"Error encoding JSON"}`)
	}

	streamChan <- fmt.Sprintf("data: %s", string(jsonBytes))
}

func (cs *LLMService) sendCrash(streamChan chan<- string) {
	streamChan <- "event: fail"
	streamChan <- fmt.Sprintf("data: [CRASH]")
}

// GenerateSubtasks processes a task and generates a response using AI
// It streams the response chunks through the provided channel
func (cs *LLMService) GenerateSubtasks(req *GenerateSubtaskRequest, userID string, session *uuid.UUID, streamChan chan<- string) error {
	ctx := context.Background()

	var taskRecord *models.Item
	var data map[string]any

	if req.TaskID != "" {
		if err := repositories.DBS.Postgres.Where("parent_id = ? AND type = ?", req.TaskID, "task").Delete(&models.Item{}).Error; err != nil {
			cs.logger.Error("Failed to delete existing subtasks", zap.String("parentID", req.TaskID), zap.Error(err))
			return fmt.Errorf("failed to delete existing subtasks: %w", err)
		}

		// Get task context data
		var err error
		data, err = cs.contextCollector.CollectContextForTask(req.TaskID, true, userID)
		if err != nil {
			return err
		}

		// If we have an answer, process it
		if req.Answer != "" {
			userTest, err := cs.understandQuestion(req.TaskID, req.Answer, userID, session)
			if err != nil {
				return err
			}
			data["UserData"] = userTest.UserData
		}

	} else if req.Task != "" {
		taskRecord = &models.Item{
			Name:      req.Task,
			Type:      "task",
			CreatedBy: userID,
		}

		var err error
		taskRecord, err = cs.taskService.CreateItemWithOrdering(taskRecord, nil)
		if err != nil {
			return err
		}

		cs.sendEvent(streamChan, "task_id", taskRecord.ID)
		req.TaskID = taskRecord.ID

		data = map[string]any{
			"TodoList":     "# " + req.Task,
			"Language":     "Korean",
			"UserData":     "",
			"SelectedTodo": req.Task,
		}
	} else {
		return fmt.Errorf("either task or taskID must be provided")
	}

	data["UserId"] = userID
	data["SessionId"] = session.String()

	// Generate subtasks
	subtaskResponse, err := cs.subtaskService.GenerateSubtasks(ctx, data, streamChan)
	if err != nil {
		return err
	}

	// Create subtasks in the database
	for _, task := range subtaskResponse.Tasks {
		item := models.Item{
			Name:        task.Title,
			Objective:   task.Objective,
			Deliverable: task.Deliverable,
			Type:        "task",
			CreatedBy:   userID,
			ParentID:    &req.TaskID,
		}
		newItem, err := cs.taskService.CreateItemWithOrdering(&item, nil)
		if err != nil {
			return fmt.Errorf("failed to create item: %w", err)
		}
		cs.sendEvent(streamChan, "task_id_start", strconv.Itoa(task.Number))
		cs.sendEventWithAdditional(streamChan, "task_id", newItem.ID, map[string]interface{}{
			"index": strconv.Itoa(task.Number),
		})
		cs.sendEvent(streamChan, "complete", "Subtask generation completed")
	}

	return nil
}

// GenerateDetails processes a task and generates pre-questions using AI
// It streams the response chunks through the provided channel
func (cs *LLMService) GenerateDetails(taskID, userID string, session *uuid.UUID, streamChan chan<- string) error {
	ctx := context.Background()

	if taskID == "" {
		return fmt.Errorf("taskID must be provided")
	}

	// Get task context data
	data, err := cs.contextCollector.CollectContextForTask(taskID, true, userID)
	if err != nil {
		return err
	}

	data["UserId"] = userID
	data["SessionId"] = session.String()

	// Generate pre-questions
	details, err := cs.detailService.GenerateDetails(ctx, data, streamChan)
	if err != nil {
		return err
	}

	item := models.ItemUpdate{
		Contents: &details,
	}
	_, err = cs.taskService.UpdateTask(taskID, item)
	if err != nil {
		return err
	}

	cs.sendEvent(streamChan, "complete", "Details generation completed")

	return nil
}

// GeneratePreQuestions processes a task and generates pre-questions using AI
// It streams the response chunks through the provided channel
func (cs *LLMService) GeneratePreQuestions(taskID, userID string, session *uuid.UUID, streamChan chan<- string) error {
	ctx := context.Background()

	if taskID == "" {
		return fmt.Errorf("taskID must be provided")
	}

	// Get task context data
	data, err := cs.contextCollector.CollectContextForTask(taskID, true, userID)
	if err != nil {
		return err
	}

	data["UserId"] = userID
	data["SessionId"] = session.String()

	// Generate pre-questions
	questions, err := cs.questionService.GeneratePreQuestions(ctx, data, streamChan)
	if err != nil {
		return err
	}

	// Create user test
	userTest, err := cs.userTestService.CreateUserTest(taskID, questions, userID, cs.logger)
	if err != nil {
		return err
	}
	cs.sendEvent(streamChan, "user_test_id", strconv.Itoa(userTest.ID))

	return nil
}

// understandQuestion processes a question and answer to analyze its intent and possible sub-questions
func (cs *LLMService) understandQuestion(taskID, answer, userID string, session *uuid.UUID) (*models.UserTests, error) {
	ctx := context.Background()

	// Update the latest user test with the answer
	userTest, err := cs.userTestService.UpdateLatestAnswer(taskID, answer, userID)
	if err != nil {
		return nil, err
	}

	// Prepare data for the AI model
	data := map[string]any{
		"Question":  userTest.Question,
		"Answer":    userTest.Answer,
		"UserId":    userID,
		"SessionId": session.String(),
	}

	// Analyze the question
	analysisOutput, err := cs.questionService.UnderstandQuestion(ctx, data)
	if err != nil {
		cs.logger.Error("Failed to understand question",
			zap.String("question", userTest.Question),
			zap.Error(err))
		return nil, fmt.Errorf("failed to understand question: %w", err)
	}

	// Log success
	cs.logger.Info("Successfully analyzed question",
		zap.String("question", userTest.Question),
		zap.String("response", analysisOutput))

	// Update user test with the analysis
	userTest, err = cs.userTestService.UpdateUserData(userTest.ID, analysisOutput, userID)
	if err != nil {
		return nil, err
	}

	return userTest, nil
}

// GenerateMindmapRequest defines the request structure for mindmap generation
type GenerateMindmapRequest struct {
	TaskID string `json:"task_id"`
	Depth  int    `json:"depth"`
}

// GenerateCompleteMindmap generates a complete mindmap with tasks up to the specified depth
func (cs *LLMService) GenerateCompleteMindmap(req *GenerateMindmapRequest, userID string, streamChan chan<- string) error {
	ctx := context.Background()

	// If depth is not specified or is invalid, default to 1
	if req.Depth < 1 {
		req.Depth = 1
	}

	// Start generating the mindmap
	cs.sendEvent(streamChan, "mindmap_start", fmt.Sprintf("Starting mindmap generation for task %s with depth %d", req.TaskID, req.Depth))

	// Create a queue of tasks to process
	type taskInfo struct {
		ID    string
		Depth int
	}
	queue := []taskInfo{{ID: req.TaskID, Depth: 0}}

	// Process tasks in the queue (breadth-first)
	for len(queue) > 0 {
		// Get the next task from the queue
		currentTask := queue[0]
		queue = queue[1:]

		// Skip if we've reached the max depth
		if currentTask.Depth >= req.Depth {
			continue
		}

		// Get task context
		data, err := cs.contextCollector.CollectContextForTask(currentTask.ID, true, userID)
		if err != nil {
			cs.sendEvent(streamChan, "error", fmt.Sprintf("Failed to collect task context: %v", err))
			continue
		}

		// Check if we need subtasks for this task (always true now)
		needSubtask, err := cs.subtaskService.CheckNeedSubtask(ctx, data, streamChan)
		if err != nil {
			cs.sendEvent(streamChan, "error", fmt.Sprintf("Failed to check if task needs subtasks: %v", err))
			continue
		}

		// If we don't need subtasks, skip this task
		if !needSubtask {
			cs.sendEvent(streamChan, "skip_task", fmt.Sprintf("Skipping subtask generation for %s", currentTask.ID))
			continue
		}

		// Generate subtasks for this task
		subtaskIDs, err := cs.generateSubtasksAndReturnIDs(ctx, currentTask.ID, userID, streamChan)
		if err != nil {
			cs.sendEvent(streamChan, "error", fmt.Sprintf("Failed to generate subtasks: %v", err))
			continue
		}

		// Add the subtasks to the queue for processing
		for _, id := range subtaskIDs {
			queue = append(queue, taskInfo{ID: id, Depth: currentTask.Depth + 1})
		}
	}

	cs.sendEvent(streamChan, "mindmap_complete", "Mindmap generation completed")
	return nil
}

// generateSubtasksAndReturnIDs generates subtasks for a task and returns their IDs
func (cs *LLMService) generateSubtasksAndReturnIDs(ctx context.Context, taskID, userID string, streamChan chan<- string) ([]string, error) {
	if err := repositories.DBS.Postgres.Where("parent_id = ? AND type = ?", taskID, "task").Delete(&models.Item{}).Error; err != nil {
		cs.logger.Error("Failed to delete existing subtasks", zap.String("parentID", taskID), zap.Error(err))
		return nil, fmt.Errorf("failed to delete existing subtasks: %w", err)
	}

	// Get task context data
	data, err := cs.contextCollector.CollectContextForTask(taskID, true, userID)
	if err != nil {
		return nil, err
	}

	// Generate subtasks
	subtasks, err := cs.subtaskService.GenerateSubtasks(ctx, data, streamChan)
	if err != nil {
		return nil, err
	}

	// Create subtasks in database and collect their IDs
	var subtaskIDs []string
	for _, task := range subtasks.Tasks {
		// Join all goals into content with newlines
		var content string
		if len(task.Goals) > 0 {
			content = strings.Join(task.Goals, "\n")
		} else if len(task.Goals) == 0 && task.Title != "" {
			content = task.Title // Use title as content if no goals
		}

		item := models.Item{
			Name:      task.Title,
			Contents:  content,
			Type:      "task",
			CreatedBy: userID,
			ParentID:  &taskID,
		}

		newItem, err := cs.taskService.CreateItemWithOrdering(&item, nil)
		if err != nil {
			return subtaskIDs, fmt.Errorf("failed to create item: %w", err)
		}

		cs.sendEvent(streamChan, "task_id_start", strconv.Itoa(task.Number))
		cs.sendEvent(streamChan, "task_id", newItem.ID)

		subtaskIDs = append(subtaskIDs, newItem.ID)
	}

	return subtaskIDs, nil
}

// TaskDependencyRequest represents a request to generate task dependencies
type TaskDependencyRequest struct {
	TaskIDs []string `json:"task_ids"`
}

// GenerateTaskDependency analyzes a set of tasks and generates their logical dependencies
func (cs *LLMService) GenerateTaskDependency(req *TaskDependencyRequest, streamChan chan<- string) error {
	ctx := context.Background()

	if len(req.TaskIDs) == 0 {
		return fmt.Errorf("at least one task ID must be provided")
	}

	// Prepare tasks data for the AI model
	tasksJSON := make([]map[string]interface{}, 0, len(req.TaskIDs))
	for i, taskID := range req.TaskIDs {
		task, err := cs.taskCollector.GetTask(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task with ID %s: %w", taskID, err)
		}

		// Add task to the JSON data
		taskData := map[string]interface{}{
			"id":   taskID,
			"name": task.Name,
			"goal": task.Contents,
		}
		tasksJSON = append(tasksJSON, taskData)

		// Send progress event
		cs.sendEvent(streamChan, "dependency_progress", fmt.Sprintf("Processing task %d of %d", i+1, len(req.TaskIDs)))
	}

	// Format as JSON
	dataFormatter := formatters.NewDataFormatter()
	jsonData, err := dataFormatter.FormatTasksAsJSON(tasksJSON)
	if err != nil {
		return fmt.Errorf("failed to format tasks as JSON: %w", err)
	}

	// Prepare data for the AI model
	data := map[string]any{
		"JsonData": jsonData,
	}

	// Generate dependencies
	_, err = cs.dependencyService.GenerateTaskDependency(ctx, data, streamChan)
	if err != nil {
		return fmt.Errorf("failed to generate task dependencies: %w", err)
	}

	return nil
}
