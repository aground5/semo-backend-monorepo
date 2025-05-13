package controllers

import (
	"fmt"
	"net/http"
	"semo-server/configs"
	"semo-server/internal/logics"
	"semo-server/internal/middlewares"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// KickoffController handles task kickoff operations
type KickoffController struct {
	LlmService     *logics.LLMService
	profileService *logics.ProfileService
}

// NewKickoffController creates a new instance of KickoffController
func NewKickoffController(llmService *logics.LLMService, profileService *logics.ProfileService) *KickoffController {
	return &KickoffController{
		LlmService:     llmService,
		profileService: profileService,
	}
}

// handleSSE sets up and handles a Server-Sent Events stream
func handleSSE(ctx echo.Context, streamHandler func(chan string)) error {
	// 1) Set up SSE headers
	ctx.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	ctx.Response().Header().Set("Cache-Control", "no-cache")
	ctx.Response().Header().Set("Connection", "keep-alive")

	// 2) Check for http.Flusher support
	flusher, ok := ctx.Response().Writer.(http.Flusher)
	if !ok {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Streaming not supported",
		})
	}

	// 3) Create stream channel
	streamChan := make(chan string)

	// 4) Call the handler function in a goroutine
	go func() {
		defer close(streamChan)
		streamHandler(streamChan)
	}()

	// 5) Detect client disconnection
	notify := ctx.Request().Context().Done()

	// 6) Stream the SSE data to client
	for {
		select {
		case <-notify:
			// Client disconnected
			return nil

		case chunk, ok := <-streamChan:
			if !ok {
				// Channel closed, all data sent
				return nil
			}

			// Write chunk and flush
			if _, err := fmt.Fprint(ctx.Response().Writer, chunk+"\n\n"); err != nil {
				// Write failed
				return nil
			}
			flusher.Flush()
		}
	}
}

// GeneratePreview handles the task preview generation endpoint
// It creates a preview of subtasks for a given task
func (kc *KickoffController) GeneratePreview(ctx echo.Context) error {
	// 1) Parse request
	var req logics.GenerateSubtaskRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := kc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Retrieve anonymous id from header
	anonymousId := ctx.Request().Header.Get("x-anonymous-id")
	sessionUUID, err := uuid.Parse(anonymousId)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	if req.Task == "" && req.TaskID == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "Task or TaskID must be provided",
		})
	}

	// Use the common SSE handler
	return handleSSE(ctx, func(streamChan chan string) {
		// Call the subtask generation function with empty parentID and preQuestion
		if err := kc.LlmService.GenerateSubtasks(&req, profile.ID, &sessionUUID, streamChan); err != nil {
			configs.Logger.Error("GenerateSubtasks failed", zap.Error(err))
		}
	})
}

// GeneratePreQuestions handles the pre-question generation endpoint
func (kc *KickoffController) GeneratePreQuestions(ctx echo.Context) error {
	// 1) Parse request - we can reuse the same request structure
	var req logics.GenerateSubtaskRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	if req.TaskID == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "task_id must be provided",
		})
	}

	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := kc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Retrieve anonymous id from header
	anonymousId := ctx.Request().Header.Get("x-anonymous-id")
	sessionUUID, err := uuid.Parse(anonymousId)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Use the common SSE handler
	return handleSSE(ctx, func(streamChan chan string) {
		// Call the pre-question generation function
		if err := kc.LlmService.GeneratePreQuestions(req.TaskID, profile.ID, &sessionUUID, streamChan); err != nil {
			// Handle error (log it)
			configs.Logger.Error("GeneratePreQuestions failed", zap.Error(err))
		}
	})
}

// GenerateDetails handles the details generation endpoint
func (kc *KickoffController) GenerateDetails(ctx echo.Context) error {
	// 1) Parse request
	var req logics.GenerateSubtaskRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	// 미들웨어에서 이메일 가져오기
	email, err := middlewares.GetEmailFromContext(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// 서비스 계층에서 프로필 조회
	profile, err := kc.profileService.GetOrCreateProfile(email)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Retrieve anonymous id from header
	anonymousId := ctx.Request().Header.Get("x-anonymous-id")
	sessionUUID, err := uuid.Parse(anonymousId)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	// Use the common SSE handler
	return handleSSE(ctx, func(streamChan chan string) {
		// Call the details generation function
		if err := kc.LlmService.GenerateDetails(req.TaskID, profile.ID, &sessionUUID, streamChan); err != nil {
			configs.Logger.Error("GenerateDetails failed", zap.Error(err))
		}
	})
}
