package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	domainErrors "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/errors"
	domainRepo "github.com/wekeepgrowing/semo-backend-monorepo/services/payment/internal/domain/repository"
	"go.uber.org/zap"
)

// SupabaseWorkspaceVerificationRepository implements workspace verification using Supabase REST API
type SupabaseWorkspaceVerificationRepository struct {
	client     *http.Client
	baseURL    string
	apiKey     string
	jwtSecret  string
	logger     *zap.Logger
}

// NewSupabaseWorkspaceVerificationRepository creates a new Supabase workspace verification repository
func NewSupabaseWorkspaceVerificationRepository(
	baseURL string,
	apiKey string,
	jwtSecret string,
	logger *zap.Logger,
) domainRepo.WorkspaceVerificationRepository {
	return &SupabaseWorkspaceVerificationRepository{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   baseURL,
		apiKey:    apiKey,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// VerifyWorkspaceMembership checks if a user belongs to a workspace via Supabase REST API
func (r *SupabaseWorkspaceVerificationRepository) VerifyWorkspaceMembership(
	ctx context.Context,
	userID,
	workspaceID string,
) (*domainRepo.WorkspaceMember, error) {
	// Start timing the repository operation
	startTime := time.Now()
	requestID := ""
	if ctx.Value("request_id") != nil {
		requestID = ctx.Value("request_id").(string)
	}
	if requestID == "" {
		requestID = fmt.Sprintf("repo_%d", time.Now().UnixNano())
	}

	r.logger.Info("SupabaseRepository: Starting workspace membership verification",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "repository_entry"),
		zap.String("status", "started"))

	// Step 1: Build the query URL
	r.logger.Debug("SupabaseRepository: Step 1 - Building Supabase query URL",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "build_query_url"))

	baseURL := fmt.Sprintf("%s/rest/v1/workspace_members", r.baseURL)
	
	// Add query parameters for filtering
	params := url.Values{}
	params.Add("user_id", fmt.Sprintf("eq.%s", userID))
	params.Add("workspace_id", fmt.Sprintf("eq.%s", workspaceID))
	params.Add("select", "*") // Select all fields
	
	queryURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	r.logger.Debug("SupabaseRepository: Query URL constructed",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("query_url", queryURL),
		zap.String("step", "build_query_url"),
		zap.String("status", "success"))

	// Step 2: Create HTTP request
	r.logger.Debug("SupabaseRepository: Step 2 - Creating HTTP request",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("method", "GET"),
		zap.String("step", "create_http_request"))

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		r.logger.Error("SupabaseRepository: Failed to create HTTP request",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("step", "create_http_request"),
			zap.String("status", "failed"),
			zap.Error(err))
		return nil, domainErrors.NewWorkspaceVerificationError(userID, workspaceID, 
			fmt.Errorf("failed to create request: %w", err))
	}

	r.logger.Debug("SupabaseRepository: HTTP request created successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "create_http_request"),
		zap.String("status", "success"))

	// Step 3: Set required headers for Supabase
	r.logger.Debug("SupabaseRepository: Step 3 - Setting Supabase headers",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("step", "set_headers"))

	req.Header.Set("apikey", r.apiKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation")

	r.logger.Debug("SupabaseRepository: Headers set successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("content_type", "application/json"),
		zap.String("prefer", "return=representation"),
		zap.String("step", "set_headers"),
		zap.String("status", "success"))

	// Step 4: Execute the HTTP request
	r.logger.Debug("SupabaseRepository: Step 4 - Executing HTTP request to Supabase",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("url", queryURL),
		zap.String("step", "execute_http_request"))

	requestStart := time.Now()
	resp, err := r.client.Do(req)
	requestDuration := time.Since(requestStart)

	if err != nil {
		r.logger.Error("SupabaseRepository: HTTP request failed",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("url", queryURL),
			zap.String("step", "execute_http_request"),
			zap.String("status", "failed"),
			zap.Duration("request_duration", requestDuration),
			zap.Error(err))
		return nil, domainErrors.NewSupabaseConnectionError(userID, workspaceID, 
			fmt.Errorf("http request failed: %w", err))
	}
	defer resp.Body.Close()

	r.logger.Debug("SupabaseRepository: HTTP request completed",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("url", queryURL),
		zap.Int("status_code", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")),
		zap.String("content_length", resp.Header.Get("Content-Length")),
		zap.String("step", "execute_http_request"),
		zap.String("status", "success"),
		zap.Duration("request_duration", requestDuration))

	// Step 5: Check HTTP response status
	r.logger.Debug("SupabaseRepository: Step 5 - Checking HTTP response status",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.Int("status_code", resp.StatusCode),
		zap.String("step", "check_response_status"))

	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		var errorBody []byte
		if resp.Body != nil {
			errorBody, _ = io.ReadAll(resp.Body)
		}
		
		r.logger.Warn("SupabaseRepository: Supabase API returned non-200 status",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.Int("status_code", resp.StatusCode),
			zap.String("status_text", resp.Status),
			zap.ByteString("response_body", errorBody),
			zap.String("step", "check_response_status"),
			zap.String("status", "failed"))
		
		if resp.StatusCode == http.StatusUnauthorized {
			r.logger.Error("SupabaseRepository: Unauthorized access to Supabase API - Check API key type and table permissions",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("workspace_id", workspaceID),
				zap.Int("status_code", resp.StatusCode),
				zap.String("suggested_fix", "Ensure using service_role key for server-side operations and workspace_members table allows read access"),
				zap.ByteString("response_body", errorBody))
			return nil, domainErrors.NewSupabaseConnectionError(userID, workspaceID,
				fmt.Errorf("unauthorized access to Supabase API - check API key permissions"))
		}
		if resp.StatusCode == http.StatusNotFound {
			r.logger.Debug("SupabaseRepository: Workspace or user not found",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("workspace_id", workspaceID),
				zap.Int("status_code", resp.StatusCode))
			return nil, domainErrors.NewWorkspaceNotFoundError(userID, workspaceID)
		}
		return nil, domainErrors.NewSupabaseConnectionError(userID, workspaceID,
			fmt.Errorf("supabase API error: status %d", resp.StatusCode))
	}

	r.logger.Debug("SupabaseRepository: HTTP response status check successful",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.Int("status_code", resp.StatusCode),
		zap.String("step", "check_response_status"),
		zap.String("status", "success"))

	// Step 6: Parse JSON response
	r.logger.Debug("SupabaseRepository: Step 6 - Parsing JSON response",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("response_content_type", resp.Header.Get("Content-Type")),
		zap.String("step", "parse_json_response"))

	parseStart := time.Now()
	var members []domainRepo.WorkspaceMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		parseDuration := time.Since(parseStart)
		r.logger.Error("SupabaseRepository: Failed to decode JSON response",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.String("step", "parse_json_response"),
			zap.String("status", "failed"),
			zap.Duration("parse_duration", parseDuration),
			zap.Error(err))
		return nil, domainErrors.NewSupabaseConnectionError(userID, workspaceID,
			fmt.Errorf("failed to decode response: %w", err))
	}
	parseDuration := time.Since(parseStart)

	r.logger.Debug("SupabaseRepository: JSON response parsed successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.Int("members_count", len(members)),
		zap.String("step", "parse_json_response"),
		zap.String("status", "success"),
		zap.Duration("parse_duration", parseDuration))

	// Step 7: Validate membership results
	r.logger.Debug("SupabaseRepository: Step 7 - Validating membership results",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.Int("members_found", len(members)),
		zap.String("step", "validate_membership_results"))

	if len(members) == 0 {
		r.logger.Warn("SupabaseRepository: User is not a member of the workspace",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.Int("members_found", len(members)),
			zap.String("step", "validate_membership_results"),
			zap.String("status", "failed"))
		return nil, domainErrors.NewUserNotMemberError(userID, workspaceID)
	}

	if len(members) > 1 {
		r.logger.Warn("SupabaseRepository: Multiple membership records found - using first one",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("workspace_id", workspaceID),
			zap.Int("members_found", len(members)),
			zap.String("step", "validate_membership_results"),
			zap.String("status", "warning"))
	}

	// Return the first (and should be only) member record
	member := &members[0]

	// Log successful completion with comprehensive details
	totalDuration := time.Since(startTime)
	r.logger.Info("SupabaseRepository: Workspace membership verification completed successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("workspace_id", workspaceID),
		zap.String("member_id", member.ID),
		zap.String("member_role", member.Role),
		zap.String("joined_at", member.JoinedAt),
		zap.String("invited_by", member.InvitedBy),
		zap.String("step", "repository_complete"),
		zap.String("status", "success"),
		zap.Duration("total_repository_duration", totalDuration),
		zap.Duration("http_request_duration", requestDuration),
		zap.Duration("json_parse_duration", parseDuration))

	return member, nil
}