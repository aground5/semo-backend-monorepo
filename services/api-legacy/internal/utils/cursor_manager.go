package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// CursorManager handles cursor-based pagination
type CursorManager struct {
	secret string
}

// CursorData stores the data encoded in a cursor
type CursorData struct {
	Timestamp time.Time
	ID        string
}

// NewCursorManager creates a new cursor manager with a given secret
func NewCursorManager(secret string) *CursorManager {
	return &CursorManager{
		secret: secret,
	}
}

// EncodeCursor generates a secure cursor from timestamp and ID
func (cm *CursorManager) EncodeCursor(timestamp time.Time, id string) string {
	// Create a cursor string with timestamp and ID
	cursorData := fmt.Sprintf("%d:%s", timestamp.UnixNano(), id)

	// Create HMAC hash
	h := hmac.New(sha256.New, []byte(cm.secret))
	h.Write([]byte(cursorData))
	hash := h.Sum(nil)

	// Combine data and hash, then base64 encode
	combined := fmt.Sprintf("%s:%s", cursorData, base64.StdEncoding.EncodeToString(hash))
	return base64.StdEncoding.EncodeToString([]byte(combined))
}

// DecodeCursor extracts timestamp and ID from a cursor, verifying the hash
func (cm *CursorManager) DecodeCursor(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, errors.New("empty cursor")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor format: %w", err)
	}

	// Split parts
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 3 {
		return nil, errors.New("invalid cursor format")
	}

	// Extract timestamp, ID, and hash
	timestampStr, id, receivedHashB64 := parts[0], parts[1], parts[2]

	// Verify hash
	h := hmac.New(sha256.New, []byte(cm.secret))
	h.Write([]byte(fmt.Sprintf("%s:%s", timestampStr, id)))
	computedHash := h.Sum(nil)
	computedHashB64 := base64.StdEncoding.EncodeToString(computedHash)

	if computedHashB64 != receivedHashB64 {
		return nil, errors.New("cursor tampering detected")
	}

	// Convert timestamp to time.Time
	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in cursor: %w", err)
	}

	timestamp := time.Unix(0, timestampInt)
	return &CursorData{
		Timestamp: timestamp,
		ID:        id,
	}, nil
}

// CursorPagination represents pagination parameters
type CursorPagination struct {
	Limit  int    `json:"limit" form:"limit"`
	Cursor string `json:"cursor" form:"cursor"`
}

// PaginationResult represents generic paginated results
type PaginationResult struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// GetPaginationDefaults sets default values for pagination parameters
func GetPaginationDefaults(pagination *CursorPagination, defaultLimit, maxLimit int) {
	if pagination.Limit <= 0 {
		pagination.Limit = defaultLimit
	} else if pagination.Limit > maxLimit {
		pagination.Limit = maxLimit
	}
}

// ExtractCursorPaginationFromContext extracts pagination parameters from Echo context
func ExtractCursorPaginationFromContext(c echo.Context) CursorPagination {
	var pagination CursorPagination

	// Get limit from query parameter, or use 0 as default (which will be set to default by GetPaginationDefaults)
	limitStr := c.QueryParam("limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			pagination.Limit = limit
		}
	}

	// Get cursor from query parameter
	pagination.Cursor = c.QueryParam("cursor")

	return pagination
}
