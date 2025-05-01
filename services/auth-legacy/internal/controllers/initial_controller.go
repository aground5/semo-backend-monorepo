package controllers

import (
	"net/http"

	"authn-server/internal/logics"
	"authn-server/internal/middlewares"
	"github.com/labstack/echo/v4"
)

// UpdateUserNameRequest is the payload for user name updates
type UpdateUserNameRequest struct {
	Name string `json:"name" form:"name"` // The new user name
}

// CreateOrganizationRequest is the payload for organization creation
type CreateOrganizationRequest struct {
	Name string `json:"name" form:"name"` // The organization name
}

// InitialStatusResponse contains the initial status information for a user
type InitialStatusResponse struct {
	NeedChangeName         bool `json:"needChangeName"`         // Whether the user needs to set their name
	NeedCreateOrganization bool `json:"needCreateOrganization"` // Whether the user needs to create an organization
}

// GetUserNameHandler handles the GET /initial/user endpoint
// Extracts user ID from JWT middleware and returns user's name
func GetUserNameHandler(c echo.Context) error {
	// Extract user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not authenticated"})
	}

	// Get user name using InitialService
	name, err := logics.InitialSvc.GetUserName(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"name": name})
}

// UpdateUserNameHandler handles the PUT /initial/user endpoint
// Updates user's name based on the request payload
func UpdateUserNameHandler(c echo.Context) error {
	req := new(UpdateUserNameRequest)
	if err := c.Bind(req); err != nil || req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Name is required"})
	}

	// Extract user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not authenticated"})
	}

	// Update user name
	if err := logics.InitialSvc.UpdateUserName(userID, req.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User name updated successfully"})
}

// CreateOrganizationHandler handles the POST /initial/organization endpoint
// Creates a new organization for the authenticated user
func CreateOrganizationHandler(c echo.Context) error {
	req := new(CreateOrganizationRequest)
	if err := c.Bind(req); err != nil || req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Organization name is required"})
	}

	// Extract user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not authenticated"})
	}

	// Create new organization
	org, err := logics.InitialSvc.CreateOrganization(req.Name, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, org)
}

// GetInitialStatusHandler handles the GET /initial/status endpoint
// Returns information about whether the user needs to set their name or create an organization
func GetInitialStatusHandler(c echo.Context) error {
	// Extract user ID from JWT context
	userID, err := middlewares.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not authenticated"})
	}

	// Check if user needs to change their name
	needChange, err := logics.InitialSvc.NeedChangeName(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Check if user needs to create an organization
	needCreate, err := logics.InitialSvc.NeedCreateOrganization(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	response := InitialStatusResponse{
		NeedChangeName:         needChange,
		NeedCreateOrganization: needCreate,
	}

	return c.JSON(http.StatusOK, response)
}
