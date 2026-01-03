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

// Handler constants
const (
	// MaxRangeDays is the maximum number of days allowed in a date range query.
	MaxRangeDays = 90

	// DefaultProgressLimit is the default pagination limit for progress queries.
	DefaultProgressLimit = 50

	// MaxProgressLimit is the maximum pagination limit for progress queries.
	MaxProgressLimit = 100
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	db       *database.DB
	resolver *calendar.DateResolver
	cfg      *config.Config
	logger   *slog.Logger
	resp     *ResponseWriter
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(db *database.DB, cfg *config.Config, logger *slog.Logger) *Handlers {
	resolver := calendar.NewDateResolver(db)
	return &Handlers{
		db:       db,
		resolver: resolver,
		cfg:      cfg,
		logger:   logger,
		resp:     NewResponseWriter(logger),
	}
}

// HealthCheck handles GET /health
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database health
	if err := h.db.Health(ctx); err != nil {
		h.logger.Warn("health check failed", slog.Any("error", err))
		h.resp.WriteServiceUnavailable(w, "Database unhealthy")
		return
	}

	h.resp.WriteSuccess(w, map[string]string{
		"status": "healthy",
	})
}

// GetTodayReadings handles GET /api/v1/readings/today
//
// Supports timezone via X-Timezone header or ?tz= query parameter.
// If no timezone is provided, defaults to UTC.
func (h *Handlers) GetTodayReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get "today" in the context of the user's timezone
	today := GetTodayForRequest(r)

	readings, err := h.getReadingsForDate(ctx, today)
	if err != nil {
		h.logger.Error("failed to get today's readings",
			slog.Any("error", err),
			slog.String("date", calendar.FormatDate(today)),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	// Include the resolved date in response for clarity
	response := map[string]interface{}{
		"date":     calendar.FormatDate(today),
		"readings": readings,
	}

	h.resp.WriteSuccess(w, response)
}

// GetDateReadings handles GET /api/v1/readings/date/{YYYY-MM-DD}
func (h *Handlers) GetDateReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract date from path
	dateStr := r.PathValue("date")
	if dateStr == "" {
		h.resp.WriteBadRequest(w, "Date parameter is required")
		return
	}

	date, err := calendar.ParseDateString(dateStr)
	if err != nil {
		h.resp.WriteBadRequest(w, fmt.Sprintf("Invalid date format: %s. Use YYYY-MM-DD", dateStr))
		return
	}

	readings, err := h.getReadingsForDate(ctx, date)
	if err != nil {
		h.logger.Error("failed to get readings for date",
			slog.String("date", dateStr),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve readings")
		return
	}

	response := map[string]interface{}{
		"date":     dateStr,
		"readings": readings,
	}

	h.resp.WriteSuccess(w, response)
}

// GetRangeReadings handles GET /api/v1/readings/range?start=YYYY-MM-DD&end=YYYY-MM-DD
func (h *Handlers) GetRangeReadings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		h.resp.WriteBadRequest(w, "Both start and end date parameters are required")
		return
	}

	startDate, err := calendar.ParseDateString(startStr)
	if err != nil {
		h.resp.WriteBadRequest(w, fmt.Sprintf("Invalid start date format: %s. Use YYYY-MM-DD", startStr))
		return
	}

	endDate, err := calendar.ParseDateString(endStr)
	if err != nil {
		h.resp.WriteBadRequest(w, fmt.Sprintf("Invalid end date format: %s. Use YYYY-MM-DD", endStr))
		return
	}

	if startDate.After(endDate) {
		h.resp.WriteBadRequest(w, "Start date must be before or equal to end date")
		return
	}

	// Limit range to prevent abuse
	daysDiff := calendar.DaysBetween(startDate, endDate)
	if daysDiff > MaxRangeDays {
		h.resp.WriteBadRequest(w, fmt.Sprintf("Date range cannot exceed %d days", MaxRangeDays))
		return
	}

	var results []map[string]interface{}
	current := startDate
	for !current.After(endDate) {
		readings, err := h.getReadingsForDate(ctx, current)
		if err != nil {
			h.logger.Warn("failed to get readings for date in range",
				slog.String("date", calendar.FormatDate(current)),
				slog.Any("error", err),
			)
			// Include error indication but continue with other dates
			results = append(results, map[string]interface{}{
				"date":  calendar.FormatDate(current),
				"error": "Failed to retrieve readings",
			})
		} else {
			results = append(results, map[string]interface{}{
				"date":     calendar.FormatDate(current),
				"readings": readings,
			})
		}
		current = current.AddDate(0, 0, 1)
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
		"start":    startStr,
		"end":      endStr,
		"count":    len(results),
		"readings": results,
	})
}

// getReadingsForDate retrieves readings for a specific date.
func (h *Handlers) getReadingsForDate(ctx context.Context, date time.Time) (*database.DailyReadings, error) {
	// Resolve date to lectionary position
	position, err := h.resolver.ResolveDate(ctx, date)
	if err != nil {
		h.logger.Error("date resolution failed",
			slog.String("date", calendar.FormatDate(date)),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("resolve date: %w", err)
	}

	h.logger.Debug("resolved date to position",
		slog.String("date", calendar.FormatDate(date)),
		slog.String("period", position.Period),
		slog.String("day_identifier", position.DayIdentifier),
		slog.Int("year_cycle", position.YearCycle),
	)

	// Get readings from database
	readings, err := h.db.GetDailyReadings(ctx, position.Period, position.DayIdentifier, position.YearCycle)
	if err != nil {
		h.logger.Error("database lookup failed",
			slog.String("period", position.Period),
			slog.String("day_identifier", position.DayIdentifier),
			slog.Int("year_cycle", position.YearCycle),
			slog.Any("error", err),
		)
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("no readings found for position: %s/%s (year %d)",
				position.Period, position.DayIdentifier, position.YearCycle)
		}
		return nil, fmt.Errorf("get daily readings: %w", err)
	}

	return readings, nil
}

// GetProgress handles GET /api/v1/progress
func (h *Handlers) GetProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	// Parse pagination parameters
	limit := parsePaginationParam(r.URL.Query().Get("limit"), DefaultProgressLimit, MaxProgressLimit)
	offset := parsePaginationParam(r.URL.Query().Get("offset"), 0, -1) // No max for offset

	progress, err := h.db.GetProgressByUser(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get progress",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve progress")
		return
	}

	h.resp.WriteSuccess(w, map[string]interface{}{
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
		h.resp.WriteBadRequest(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	if req.ReadingID <= 0 {
		h.resp.WriteBadRequest(w, "reading_id must be a positive integer")
		return
	}

	// Verify reading exists
	_, err := h.db.GetReadingByID(ctx, req.ReadingID)
	if err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, "Reading not found")
			return
		}
		h.logger.Error("failed to get reading",
			slog.Int64("reading_id", req.ReadingID),
			slog.Any("error", err),
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
		ReadingID:   req.ReadingID,
		Notes:       notes,
		CompletedAt: time.Now(),
	}

	if err := h.db.CreateProgress(ctx, progress); err != nil {
		if err == database.ErrDuplicate {
			h.resp.WriteConflict(w, "Reading already marked as complete")
			return
		}
		h.logger.Error("failed to create progress",
			slog.String("user_id", userID),
			slog.Int64("reading_id", req.ReadingID),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to mark reading as complete")
		return
	}

	h.resp.WriteSuccess(w, progress)
}

// DeleteProgress handles DELETE /api/v1/progress/{id}
func (h *Handlers) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	progressIDStr := r.PathValue("id")
	if progressIDStr == "" {
		h.resp.WriteBadRequest(w, "Progress ID is required")
		return
	}

	progressID, err := strconv.ParseInt(progressIDStr, 10, 64)
	if err != nil {
		h.resp.WriteBadRequest(w, "Invalid progress ID")
		return
	}

	// Verify progress belongs to user
	progress, err := h.db.GetProgressByID(ctx, progressID)
	if err != nil {
		if database.IsNotFound(err) {
			h.resp.WriteNotFound(w, "Progress entry not found")
			return
		}
		h.logger.Error("failed to get progress",
			slog.Int64("progress_id", progressID),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve progress")
		return
	}

	if progress.UserID != userID {
		h.resp.WriteUnauthorized(w, "Progress entry does not belong to user")
		return
	}

	if err := h.db.DeleteProgress(ctx, progressID); err != nil {
		h.logger.Error("failed to delete progress",
			slog.Int64("progress_id", progressID),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to delete progress")
		return
	}

	h.resp.WriteSuccess(w, map[string]string{"message": "Progress deleted"})
}

// GetProgressStats handles GET /api/v1/progress/stats
func (h *Handlers) GetProgressStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := GetUserID(r, h.cfg)

	stats, err := h.db.GetProgressStats(ctx, userID)
	if err != nil {
		h.logger.Error("failed to get progress stats",
			slog.String("user_id", userID),
			slog.Any("error", err),
		)
		h.resp.WriteInternalError(w, "Failed to retrieve statistics")
		return
	}

	h.resp.WriteSuccess(w, stats)
}

// decodeJSON decodes JSON request body.
func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Reject requests with unknown fields
	return decoder.Decode(v)
}

// parsePaginationParam parses a pagination parameter string.
// Returns defaultVal if the string is empty or invalid.
// If maxVal is positive, clamps the result to maxVal.
func parsePaginationParam(s string, defaultVal, maxVal int) int {
	if s == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(s)
	if err != nil || val < 0 {
		return defaultVal
	}

	if maxVal > 0 && val > maxVal {
		return maxVal
	}

	return val
}
