package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/zapponejosh/lectionary-api/internal/calendar"
	"github.com/zapponejosh/lectionary-api/internal/config"
	"github.com/zapponejosh/lectionary-api/internal/database"
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	db       *database.DB
	resolver *calendar.DateResolver
	cfg      *config.Config
	logger   *slog.Logger
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *database.DB, cfg *config.Config, logger *slog.Logger) *Handlers {
	resolver := calendar.NewDateResolver(db)
	return &Handlers{
		db:       db,
		resolver: resolver,
		cfg:      cfg,
		logger:   logger,
	}
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database health
	if err := h.db.Health(ctx); err != nil {
		h.logger.Warn("health check failed", slog.Any("error", err))
		WriteError(w, http.StatusServiceUnavailable, "Database unhealthy", "HEALTH_CHECK_FAILED")
		return
	}

	WriteSuccess(w, map[string]string{
		"status": "healthy",
	})
}

// GetTodayReadings handles GET /api/v1/readings/today
func (h *Handlers) GetTodayReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := time.Now()

	readings, err := h.getReadingsForDate(ctx, today)
	if err != nil {
		h.logger.Error("failed to get today's readings", slog.Any("error", err))
		WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	WriteSuccess(w, readings)
}

// GetDateReadings handles GET /api/v1/readings/date/{YYYY-MM-DD}
func (h *Handlers) GetDateReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract date from path
	dateStr := r.PathValue("date")
	if dateStr == "" {
		WriteBadRequest(w, "Date parameter is required")
		return
	}

	date, err := calendar.ParseDateString(dateStr)
	if err != nil {
		WriteBadRequest(w, fmt.Sprintf("Invalid date format: %s. Use YYYY-MM-DD", dateStr))
		return
	}

	readings, err := h.getReadingsForDate(ctx, date)
	if err != nil {
		h.logger.Error("failed to get readings for date",
			slog.String("date", dateStr),
			slog.Any("error", err))
		WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	WriteSuccess(w, readings)
}

// GetRangeReadings handles GET /api/v1/readings/range?start=YYYY-MM-DD&end=YYYY-MM-DD
func (h *Handlers) GetRangeReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		WriteBadRequest(w, "Both start and end date parameters are required")
		return
	}

	startDate, err := calendar.ParseDateString(startStr)
	if err != nil {
		WriteBadRequest(w, fmt.Sprintf("Invalid start date format: %s. Use YYYY-MM-DD", startStr))
		return
	}

	endDate, err := calendar.ParseDateString(endStr)
	if err != nil {
		WriteBadRequest(w, fmt.Sprintf("Invalid end date format: %s. Use YYYY-MM-DD", endStr))
		return
	}

	if startDate.After(endDate) {
		WriteBadRequest(w, "Start date must be before or equal to end date")
		return
	}

	// Limit range to 90 days to prevent abuse
	daysDiff := int(endDate.Sub(startDate).Hours() / 24)
	if daysDiff > 90 {
		WriteBadRequest(w, "Date range cannot exceed 90 days")
		return
	}

	var results []interface{}
	current := startDate
	for !current.After(endDate) {
		readings, err := h.getReadingsForDate(ctx, current)
		if err != nil {
			h.logger.Warn("failed to get readings for date in range",
				slog.String("date", calendar.FormatDate(current)),
				slog.Any("error", err))
			// Continue with other dates even if one fails
		} else {
			results = append(results, readings)
		}
		current = current.AddDate(0, 0, 1)
	}

	WriteSuccess(w, map[string]interface{}{
		"start":    startStr,
		"end":      endStr,
		"readings": results,
	})
}

// getReadingsForDate retrieves readings for a specific date.
func (h *Handlers) getReadingsForDate(ctx context.Context, date time.Time) (*database.DailyReadings, error) {
	// Resolve date to lectionary position
	position, err := h.resolver.ResolveDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("resolve date: %w", err)
	}

	// Get readings from database
	readings, err := h.db.GetDailyReadings(ctx, position.Period, position.DayIdentifier, position.YearCycle)
	if err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("no readings found for position: %s/%s", position.Period, position.DayIdentifier)
		}
		return nil, fmt.Errorf("get daily readings: %w", err)
	}

	return readings, nil
}

// GetProgress handles GET /api/v1/progress
func (h *Handlers) GetProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // default
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	progress, err := h.db.GetProgressByUser(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get progress", slog.Any("error", err))
		WriteInternalError(w, "Failed to retrieve progress")
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"progress": progress,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateProgress handles POST /api/v1/progress
func (h *Handlers) CreateProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	var req struct {
		ReadingID int64  `json:"reading_id"`
		Notes     string `json:"notes,omitempty"`
	}

	if err := decodeJSON(r, &req); err != nil {
		WriteBadRequest(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReadingID <= 0 {
		WriteBadRequest(w, "reading_id must be a positive integer")
		return
	}

	// Verify reading exists
	_, err := h.db.GetReadingByID(ctx, req.ReadingID)
	if err != nil {
		if database.IsNotFound(err) {
			WriteNotFound(w, "Reading not found")
			return
		}
		h.logger.Error("failed to get reading", slog.Any("error", err))
		WriteInternalError(w, "Failed to verify reading")
		return
	}

	// Create progress entry
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	progress := &database.ReadingProgress{
		UserID:      userID,
		ReadingID:   req.ReadingID,
		Notes:       notes,
		CompletedAt: time.Now(),
	}

	if err := h.db.CreateProgress(ctx, progress); err != nil {
		if err == database.ErrDuplicate {
			WriteError(w, http.StatusConflict, "Reading already marked as complete", "DUPLICATE")
			return
		}
		h.logger.Error("failed to create progress", slog.Any("error", err))
		WriteInternalError(w, "Failed to mark reading as complete")
		return
	}

	WriteSuccess(w, progress)
}

// DeleteProgress handles DELETE /api/v1/progress/{id}
func (h *Handlers) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	progressIDStr := r.PathValue("id")
	if progressIDStr == "" {
		WriteBadRequest(w, "Progress ID is required")
		return
	}

	progressID, err := strconv.ParseInt(progressIDStr, 10, 64)
	if err != nil {
		WriteBadRequest(w, "Invalid progress ID")
		return
	}

	// Verify progress belongs to user
	progress, err := h.db.GetProgressByID(ctx, progressID)
	if err != nil {
		if database.IsNotFound(err) {
			WriteNotFound(w, "Progress entry not found")
			return
		}
		h.logger.Error("failed to get progress", slog.Any("error", err))
		WriteInternalError(w, "Failed to retrieve progress")
		return
	}

	if progress.UserID != userID {
		WriteUnauthorized(w, "Progress entry does not belong to user")
		return
	}

	if err := h.db.DeleteProgress(ctx, progressID); err != nil {
		h.logger.Error("failed to delete progress", slog.Any("error", err))
		WriteInternalError(w, "Failed to delete progress")
		return
	}

	WriteSuccess(w, map[string]string{"message": "Progress deleted"})
}

// GetProgressStats handles GET /api/v1/progress/stats
func (h *Handlers) GetProgressStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	stats, err := h.db.GetProgressStats(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get progress stats", slog.Any("error", err))
		WriteInternalError(w, "Failed to retrieve statistics")
		return
	}

	WriteSuccess(w, stats)
}

// decodeJSON decodes JSON request body.
func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(v)
}
