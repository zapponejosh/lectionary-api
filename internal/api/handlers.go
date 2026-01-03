package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/config"
	"github.com/zapponejosh/lectionary-api/internal/database"
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	db     *database.DB
	cfg    *config.Config
	logger *slog.Logger
	resp   *ResponseWriter
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *database.DB, cfg *config.Config, logger *slog.Logger) *Handlers {
	return &Handlers{
		db:     db,
		cfg:    cfg,
		logger: logger,
		resp:   NewResponseWriter(logger),
	}
}

// =============================================================================
// Health Check
// =============================================================================

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database health
	dbHealthy := true
	var stats *database.ReadingStats

	if err := h.db.Health(ctx); err != nil {
		h.logger.Warn("health check: database unhealthy", slog.Any("error", err))
		dbHealthy = false
	} else {
		// Get database stats if healthy
		stats, _ = h.db.GetReadingStats(ctx)
	}

	response := map[string]interface{}{
		"status": "healthy",
		"database": map[string]interface{}{
			"healthy": dbHealthy,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if stats != nil {
		response["database"].(map[string]interface{})["total_readings"] = stats.TotalDays
		response["database"].(map[string]interface{})["date_range"] = map[string]string{
			"earliest": stats.EarliestDate,
			"latest":   stats.LatestDate,
		}
	}

	if !dbHealthy {
		h.resp.WriteServiceUnavailable(w, "Database unhealthy")
		return
	}

	h.resp.WriteSuccess(w, response)
}

// =============================================================================
// Reading Endpoints
// =============================================================================

// GetTodayReadings handles GET /api/v1/readings/today
//
// Supports timezone via X-Timezone header.
// If no timezone is provided, defaults to UTC.
func (h *Handlers) GetTodayReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get "today" in the context of the user's timezone
	today := GetTodayForRequest(r)
	dateStr := today.Format("2006-01-02")

	h.logger.Debug("fetching today's readings",
		slog.String("date", dateStr),
		slog.String("timezone", today.Location().String()),
	)

	// Fetch from database
	readings, err := h.db.GetReadingByDate(ctx, dateStr)
	if err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, fmt.Sprintf("No readings found for %s", dateStr))
			return
		}
		h.logger.Error("failed to get today's readings",
			slog.String("date", dateStr),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	h.resp.WriteSuccess(w, readings)
}

// GetDateReadings handles GET /api/v1/readings/date/{date}
func (h *Handlers) GetDateReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract date from path
	dateStr := r.PathValue("date")
	if dateStr == "" {
		h.resp.WriteBadRequest(w, "Date parameter is required")
		return
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	h.logger.Debug("fetching readings for date",
		slog.String("date", dateStr),
	)

	// Fetch from database
	readings, err := h.db.GetReadingByDate(ctx, dateStr)
	if err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, fmt.Sprintf("No readings found for %s", dateStr))
			return
		}
		h.logger.Error("failed to get readings",
			slog.String("date", dateStr),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	h.resp.WriteSuccess(w, readings)
}

// GetRangeReadings handles GET /api/v1/readings/range
func (h *Handlers) GetRangeReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")

	// Validate required parameters
	if startDate == "" || endDate == "" {
		h.resp.WriteBadRequest(w, "Both start and end date parameters are required")
		return
	}

	// Validate date formats
	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid start date format. Use YYYY-MM-DD")
		return
	}

	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid end date format. Use YYYY-MM-DD")
		return
	}

	// Validate date range (start must be before or equal to end)
	if startDate > endDate {
		h.resp.WriteBadRequest(w, "Start date must be before or equal to end date")
		return
	}

	h.logger.Debug("fetching readings for range",
		slog.String("start", startDate),
		slog.String("end", endDate),
	)

	// Fetch from database
	readings, err := h.db.GetReadingsByDateRange(ctx, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to get readings range",
			slog.String("start", startDate),
			slog.String("end", endDate),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	// Return empty array if no readings found (not an error)
	if len(readings) == 0 {
		h.resp.WriteSuccess(w, []interface{}{})
		return
	}

	h.resp.WriteSuccess(w, readings)
}

// Replace the progress endpoint placeholders in handlers.go with these implementations

// =============================================================================
// Progress Endpoints (Fully Implemented)
// =============================================================================

// GetProgress handles GET /api/v1/progress
// Returns paginated list of completed readings for the authenticated user.
// Query params: limit (default 50, max 100), offset (default 0)
func (h *Handlers) GetProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			if parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil {
			if parsed >= 0 {
				offset = parsed
			}
		}
	}

	h.logger.Debug("fetching user progress",
		slog.String("user_id", userID),
		slog.Int("limit", limit),
		slog.Int("offset", offset),
	)

	// Fetch progress from database
	progress, err := h.db.GetProgressByUser(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get progress",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve progress")
		return
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
		"progress": progress,
		"limit":    limit,
		"offset":   offset,
		"count":    len(progress),
	})
}

// CreateProgress handles POST /api/v1/progress
// Marks a reading as completed for the authenticated user.
// Request body: {"date": "YYYY-MM-DD", "notes": "optional notes"}
func (h *Handlers) CreateProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	// Parse request body
	var req struct {
		Date  string `json:"date"`
		Notes string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.resp.WriteBadRequest(w, "Invalid request body")
		return
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	h.logger.Debug("creating progress entry",
		slog.String("user_id", userID),
		slog.String("date", req.Date),
	)

	// Check if reading exists for this date
	_, err = h.db.GetReadingByDate(ctx, req.Date)
	if err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, fmt.Sprintf("No reading found for %s", req.Date))
			return
		}
		h.logger.Error("failed to verify reading exists",
			slog.String("date", req.Date),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to verify reading")
		return
	}

	// Create progress entry
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	progress := &database.ReadingProgress{
		UserID:      userID,
		ReadingDate: req.Date,
		Notes:       notes,
		CompletedAt: time.Now(),
	}

	if err := h.db.CreateProgress(ctx, progress); err != nil {
		if err == database.ErrDuplicate {
			h.resp.WriteConflict(w, fmt.Sprintf("Reading for %s already marked as complete", req.Date))
			return
		}
		h.logger.Error("failed to create progress",
			slog.String("user_id", userID),
			slog.String("date", req.Date),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to mark reading as complete")
		return
	}

	h.logger.Info("progress created",
		slog.String("user_id", userID),
		slog.String("date", req.Date),
	)

	h.resp.WriteSuccess(w, progress)
}

// DeleteProgress handles DELETE /api/v1/progress/{id}
// Removes a completed reading for the authenticated user.
// Path parameter {id} is actually the date (YYYY-MM-DD)
func (h *Handlers) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	// Get date from path parameter
	// Note: routes.go uses {id} but we're treating it as a date
	date := r.PathValue("id")
	if date == "" {
		h.resp.WriteBadRequest(w, "Date parameter is required")
		return
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	h.logger.Debug("deleting progress entry",
		slog.String("user_id", userID),
		slog.String("date", date),
	)

	// Delete progress entry
	if err := h.db.DeleteProgress(ctx, userID, date); err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, fmt.Sprintf("No completed reading found for %s", date))
			return
		}
		h.logger.Error("failed to delete progress",
			slog.String("user_id", userID),
			slog.String("date", date),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to delete progress")
		return
	}

	h.logger.Info("progress deleted",
		slog.String("user_id", userID),
		slog.String("date", date),
	)

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message": "Progress entry deleted",
		"date":    date,
	})
}

// GetProgressStats handles GET /api/v1/progress/stats
// Returns reading statistics for the authenticated user.
// Includes: total days, completed days, completion %, current streak, longest streak
func (h *Handlers) GetProgressStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	h.logger.Debug("fetching progress stats",
		slog.String("user_id", userID),
	)

	// Get statistics from database
	stats, err := h.db.GetProgressStats(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get progress stats",
			slog.String("user_id", userID),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve statistics")
		return
	}

	h.resp.WriteSuccess(w, stats)
}

// CreateUser handles POST /api/v1/admin/users (admin only)
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Username string  `json:"username"`
		Email    *string `json:"email,omitempty"`
		FullName *string `json:"full_name,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.resp.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Username == "" {
		h.resp.WriteBadRequest(w, "username is required")
		return
	}

	user, err := h.db.CreateUser(ctx, req.Username, req.Email, req.FullName)
	if err != nil {
		if err == database.ErrDuplicate {
			h.resp.WriteConflict(w, "Username already exists")
			return
		}
		h.logger.Error("failed to create user",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to create user")
		return
	}

	h.logger.Info("user created",
		slog.String("username", user.Username),
		slog.Int64("user_id", user.ID),
	)

	h.resp.WriteSuccess(w, user)
}

// CreateAPIKey handles POST /api/v1/admin/users/{userID}/keys (admin only)
func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from path
	userIDStr := r.PathValue("userID")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid user ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.resp.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		h.resp.WriteBadRequest(w, "name is required")
		return
	}

	keyWithPlaintext, err := h.db.CreateAPIKey(ctx, userID, req.Name)
	if err != nil {
		h.logger.Error("failed to create api key",
			slog.Int64("user_id", userID),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to create API key")
		return
	}

	h.logger.Info("api key created",
		slog.Int64("user_id", userID),
		slog.String("key_name", req.Name),
	)

	// Return the key WITH plaintext (only time it's ever shown)
	h.resp.WriteSuccess(w, map[string]interface{}{
		"api_key": keyWithPlaintext,
		"warning": "Save this key now. You won't be able to see it again.",
	})
}

// ListUsers handles GET /api/v1/admin/users (admin only)
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	users, err := h.db.ListUsers(ctx)
	if err != nil {
		h.logger.Error("failed to list users",
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to list users")
		return
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// GetCurrentUser handles GET /api/v1/me
// Returns the authenticated user's profile
func (h *Handlers) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	if user == nil {
		h.resp.WriteUnauthorized(w, "Not authenticated")
		return
	}

	h.resp.WriteSuccess(w, user)
}

// GetMyAPIKeys handles GET /api/v1/me/keys
// Returns the authenticated user's API keys
func (h *Handlers) GetMyAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := GetUser(r)
	if user == nil {
		h.resp.WriteUnauthorized(w, "Not authenticated")
		return
	}

	keys, err := h.db.ListUserAPIKeys(ctx, user.ID)
	if err != nil {
		h.logger.Error("failed to list user api keys",
			slog.Int64("user_id", user.ID),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to list API keys")
		return
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
		"api_keys": keys,
		"count":    len(keys),
	})
}

// RevokeMyAPIKey handles DELETE /api/v1/me/keys/{keyID}
// Allows user to revoke their own API key
func (h *Handlers) RevokeMyAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := GetUser(r)
	if user == nil {
		h.resp.WriteUnauthorized(w, "Not authenticated")
		return
	}

	keyIDStr := r.PathValue("keyID")
	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid key ID")
		return
	}

	if err := h.db.RevokeAPIKey(ctx, keyID, user.ID); err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, "API key not found")
			return
		}
		h.logger.Error("failed to revoke api key",
			slog.Int64("user_id", user.ID),
			slog.Int64("key_id", keyID),
			slog.String("error", err.Error()),
		)
		h.resp.WriteInternalError(w, "Failed to revoke API key")
		return
	}

	h.logger.Info("api key revoked",
		slog.Int64("user_id", user.ID),
		slog.Int64("key_id", keyID),
	)

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message": "API key revoked successfully",
	})
}
