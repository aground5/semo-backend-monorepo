package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"semo-server/configs"
	"semo-server/internal/repositories"
)

const (
	// Maximum number of LLM requests allowed per day
	maxDailyRequests = 1000

	// Redis key prefix for LLM usage
	llmUsageKeyPrefix = "llm_usage:"

	// MongoDB collection for LLM usage logs
	llmLogsCollection = "llm_usage_logs"
)

// TODO: logger 주입이 이루어져야 함, 현재 연결된 부분이 없어서 주입이 안되고 있음

// LLMUsageMiddleware checks and limits LLM API usage per user
func LLMUsageMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Extract user ID from JWT context
		userID, err := GetUserIDFromContext(c)
		if err != nil {
			// If user is not authenticated, continue without tracking
			configs.Logger.Warn("LLM usage not tracked: user not authenticated", zap.Error(err))
			return next(c)
		}

		// Check if user has exceeded daily limit
		redisKey := llmUsageKeyPrefix + userID

		// Get current count from Redis
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get current usage count
		count, err := repositories.DBS.Redis.Get(ctx, redisKey).Int()
		if err != nil && err.Error() != "redis: nil" {
			configs.Logger.Error("Failed to get LLM usage count from Redis", zap.Error(err))
			// Continue anyway to not disrupt service (fail open)
			count = 0
		}

		// Check if limit is exceeded
		if count >= maxDailyRequests {
			configs.Logger.Warn("User exceeded LLM usage limit",
				zap.String("userID", userID),
				zap.Int("count", count))
			return c.JSON(http.StatusTooManyRequests, echo.Map{
				"error": "Daily LLM usage limit exceeded. Please try again tomorrow.",
			})
		}

		// Capture request body for MongoDB logging
		var requestBody map[string]interface{}
		if err := c.Bind(&requestBody); err != nil {
			// If we can't bind the body, still continue with the request
			configs.Logger.Warn("Failed to bind request body for LLM logging", zap.Error(err))
		}

		// Create a copy of the request body to avoid modification
		requestBodyCopy := make(map[string]interface{})
		for k, v := range requestBody {
			requestBodyCopy[k] = v
		}

		// Store the original body back in the context
		c.Set("requestBody", requestBodyCopy)

		// Store start time for duration measurement
		startTime := time.Now()

		// Process the request
		if err := next(c); err != nil {
			return err
		}

		// After processing, increment usage count in Redis
		_, redisErr := repositories.DBS.Redis.Incr(ctx, redisKey).Result()
		if redisErr != nil {
			configs.Logger.Error("Failed to increment LLM usage count in Redis", zap.Error(redisErr))
		}

		// Set expiry to end of day if this is a new key
		if count == 0 && redisErr == nil {
			// Calculate time until midnight
			now := time.Now()
			midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			ttl := midnight.Sub(now)

			// Set expiry to midnight
			repositories.DBS.Redis.Expire(ctx, redisKey, ttl)
		}

		// Log usage to MongoDB
		logEntry := bson.M{
			"user_id":      userID,
			"request_path": c.Path(),
			"request_body": requestBodyCopy,
			"timestamp":    time.Now(),
			"duration_ms":  time.Since(startTime).Milliseconds(),
			"content_type": c.Request().Header.Get("Content-Type"),
			"method":       c.Request().Method,
			"user_agent":   c.Request().UserAgent(),
			"ip":           c.RealIP(),
			"service":      configs.Configs.Service.ServiceName,
			"status_code":  c.Response().Status,
		}

		// Log to MongoDB asynchronously
		go logToMongoDB(logEntry)

		return nil
	}
}

// logToMongoDB logs LLM usage data to MongoDB
func logToMongoDB(logEntry bson.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := repositories.DBS.MongoDB.Database("semo").Collection(llmLogsCollection)

	_, err := collection.InsertOne(ctx, logEntry)
	if err != nil {
		configs.Logger.Error("Failed to log LLM usage to MongoDB",
			zap.Error(err),
			zap.String("user_id", logEntry["user_id"].(string)))
	}
}
