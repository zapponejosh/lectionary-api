package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

// =============================================================================
// Progress Endpoints (Placeholders for future implementation)
// =============================================================================

// GetProgress handles GET /api/v1/progress
func (h *Handlers) GetProgress(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement progress tracking
	// Will need to:
	// 1. Get user ID from request (API key hash)
	// 2. Parse limit/offset from query params
	// 3. Query reading_progress table
	// 4. Return paginated results

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message":  "Progress tracking not yet implemented",
		"progress": []interface{}{},
		"limit":    50,
		"offset":   0,
	})
}

// CreateProgress handles POST /api/v1/progress
func (h *Handlers) CreateProgress(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement progress tracking
	// Will need to:
	// 1. Get user ID from request (API key hash)
	// 2. Parse JSON body (date, notes)
	// 3. Validate date exists in daily_readings
	// 4. Insert into reading_progress table
	// 5. Handle duplicate (already completed) case

	var req struct {
		Date  string `json:"date"`
		Notes string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.resp.WriteBadRequest(w, "Invalid request body")
		return
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message": "Progress tracking not yet implemented",
		"date":    req.Date,
	})
}

// DeleteProgress handles DELETE /api/v1/progress/{id}
func (h *Handlers) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement progress tracking
	// Will need to:
	// 1. Get user ID from request (API key hash)
	// 2. Get date/id from path parameter
	// 3. Delete from reading_progress table
	// 4. Return 404 if not found

	id := r.PathValue("id")

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message": "Progress tracking not yet implemented",
		"id":      id,
	})
}

// GetProgressStats handles GET /api/v1/progress/stats
func (h *Handlers) GetProgressStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement progress tracking
	// Will need to:
	// 1. Get user ID from request (API key hash)
	// 2. Count total readings in date range
	// 3. Count completed readings for user
	// 4. Calculate streaks
	// 5. Return statistics

	h.resp.WriteSuccess(w, map[string]interface{}{
		"message": "Progress tracking not yet implemented",
		"stats": map[string]interface{}{
			"total_readings":     0,
			"completed_readings": 0,
			"completion_percent": 0.0,
			"current_streak":     0,
			"longest_streak":     0,
		},
	})
}
