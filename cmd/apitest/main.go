package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// =============================================================================
// Response Types - Match the actual API response structure
// =============================================================================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// DateReadingsResponse is the response for /readings/date/{date} and /readings/today
type DateReadingsResponse struct {
	Date     string         `json:"date"`
	Readings *DailyReadings `json:"readings"`
}

// DailyReadings contains the lectionary data for a single day
type DailyReadings struct {
	Period        string    `json:"period"`
	DayIdentifier string    `json:"day_identifier"`
	SpecialName   *string   `json:"special_name,omitempty"`
	MorningPsalms []string  `json:"morning_psalms"`
	EveningPsalms []string  `json:"evening_psalms"`
	YearCycle     int       `json:"year_cycle"`
	Readings      []Reading `json:"readings"`
}

type Reading struct {
	ID          int64  `json:"id"`
	ReadingType string `json:"reading_type"`
	Position    int    `json:"position"`
	Reference   string `json:"reference"`
}

// RangeReadingsResponse is the response for /readings/range
type RangeReadingsResponse struct {
	Start    string                 `json:"start"`
	End      string                 `json:"end"`
	Count    int                    `json:"count"`
	Readings []DateReadingsResponse `json:"readings"`
}

// HealthResponse is the response for /health
type HealthResponse struct {
	Status string `json:"status"`
}

// =============================================================================
// Test Runner
// =============================================================================

type TestRunner struct {
	baseURL      string
	client       *http.Client
	verbose      bool
	successCount int
	errorCount   int
	errors       []string
}

func NewTestRunner(baseURL string, verbose bool) *TestRunner {
	return &TestRunner{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		verbose: verbose,
	}
}

func (tr *TestRunner) Run() {
	fmt.Println("==============================================")
	fmt.Println("Lectionary API Test Suite")
	fmt.Println("==============================================")
	fmt.Printf("Base URL: %s\n", tr.baseURL)
	fmt.Println()

	// Run test groups
	tr.testHealth()
	tr.testTodayReadings()
	tr.testSpecificDates()
	tr.testDateRange()
	tr.testEdgeCases()
	tr.testLiturgicalSeasons()

	// Print summary
	tr.printSummary()
}

// =============================================================================
// Test Groups
// =============================================================================

func (tr *TestRunner) testHealth() {
	tr.printSection("Health Check")

	resp, err := tr.get("/health")
	if err != nil {
		tr.recordError("Health", err.Error())
		return
	}

	var health HealthResponse
	if err := tr.parseDataAs(resp, &health); err != nil {
		tr.recordError("Health", err.Error())
		return
	}

	if health.Status == "healthy" {
		tr.recordSuccess("Health check passed")
	} else {
		tr.recordError("Health", fmt.Sprintf("Unexpected status: %s", health.Status))
	}
}

func (tr *TestRunner) testTodayReadings() {
	tr.printSection("Today's Readings")

	// Test without timezone
	resp, err := tr.get("/api/v1/readings/today")
	if err != nil {
		tr.recordError("Today (no TZ)", err.Error())
		return
	}

	var data DateReadingsResponse
	if err := tr.parseDataAs(resp, &data); err != nil {
		tr.recordError("Today (no TZ)", err.Error())
		return
	}

	tr.recordSuccess(fmt.Sprintf("Today (%s): %s / %s",
		data.Date, data.Readings.Period, data.Readings.DayIdentifier))
	tr.printReadingsDetail(data.Readings)

	// Test with timezone header
	req, _ := http.NewRequest("GET", tr.baseURL+"/api/v1/readings/today", nil)
	req.Header.Set("X-Timezone", "America/New_York")
	httpResp, err := tr.client.Do(req)
	if err != nil {
		tr.recordError("Today (EST)", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == 200 {
		tr.recordSuccess("Today with X-Timezone header works")
	} else {
		tr.recordError("Today (EST)", fmt.Sprintf("HTTP %d", httpResp.StatusCode))
	}
}

func (tr *TestRunner) testSpecificDates() {
	tr.printSection("Specific Date Tests")

	testCases := []struct {
		date           string
		expectedPeriod string
		description    string
	}{
		// Advent 2024 (Year 1)
		{"2024-12-01", "1st Week of Advent", "First Sunday of Advent 2024"},
		{"2024-12-08", "2nd Week of Advent", "Second Sunday of Advent"},
		{"2024-12-15", "3rd Week of Advent", "Third Sunday of Advent"},
		{"2024-12-22", "4th Week of Advent", "Fourth Sunday of Advent"},

		// Christmas Season
		{"2024-12-25", "Christmas", "Christmas Day 2024"},
		{"2024-12-26", "Christmas Season", "Day after Christmas"},
		{"2024-12-31", "Christmas Season", "New Year's Eve"},
		{"2025-01-01", "Christmas Season", "New Year's Day"},
		{"2025-01-05", "Christmas Season", "Last day of Christmas Season"},

		// Epiphany
		{"2025-01-06", "Epiphany and Following", "Epiphany"},
		{"2025-01-12", "Epiphany and Following", "Last day of Epiphany week"},

		// Baptism of the Lord (Sunday between Jan 7-13)
		{"2025-01-12", "Epiphany and Following", "Could be Baptism Sunday"},

		// Lent 2025
		{"2025-03-05", "Ash Wednesday and Following", "Ash Wednesday 2025"},
		{"2025-03-09", "1st Week of Lent", "First Sunday of Lent"},

		// Holy Week & Easter 2025
		{"2025-04-13", "Holy Week", "Palm Sunday 2025"},
		{"2025-04-17", "Holy Week", "Maundy Thursday"},
		{"2025-04-18", "Holy Week", "Good Friday"},
		{"2025-04-20", "Easter Week", "Easter Sunday 2025"}, // First week is "Easter Week", not "1st Week of Easter"

		// Pentecost 2025
		{"2025-06-08", "Pentecost", "Pentecost Sunday 2025"},
		{"2025-06-15", "Trinity Sunday and Following", "Trinity Sunday (Sunday after Pentecost)"}, // Special name for this Sunday

		// Advent 2025 (Year 2)
		{"2025-11-30", "1st Week of Advent", "First Sunday of Advent 2025"},
	}

	for _, tc := range testCases {
		resp, err := tr.get(fmt.Sprintf("/api/v1/readings/date/%s", tc.date))
		if err != nil {
			tr.recordError(tc.date, err.Error())
			continue
		}

		var data DateReadingsResponse
		if err := tr.parseDataAs(resp, &data); err != nil {
			tr.recordError(tc.date, err.Error())
			continue
		}

		// Check if period matches expected (prefix match for flexibility)
		if strings.HasPrefix(data.Readings.Period, tc.expectedPeriod) ||
			data.Readings.Period == tc.expectedPeriod {
			tr.recordSuccess(fmt.Sprintf("%s: %s (%s)",
				tc.date, data.Readings.Period, tc.description))
		} else {
			tr.recordError(tc.date, fmt.Sprintf("Expected period '%s', got '%s'",
				tc.expectedPeriod, data.Readings.Period))
		}

		if tr.verbose {
			tr.printReadingsDetail(data.Readings)
		}
	}
}

func (tr *TestRunner) testDateRange() {
	tr.printSection("Date Range Tests")

	// Test a week range
	resp, err := tr.get("/api/v1/readings/range?start=2025-12-21&end=2025-12-27")
	if err != nil {
		tr.recordError("Range (week)", err.Error())
		return
	}

	var rangeData RangeReadingsResponse
	if err := tr.parseDataAs(resp, &rangeData); err != nil {
		tr.recordError("Range (week)", err.Error())
		return
	}

	if rangeData.Count == 7 {
		tr.recordSuccess(fmt.Sprintf("Week range returned %d days", rangeData.Count))
	} else {
		tr.recordError("Range (week)", fmt.Sprintf("Expected 7 days, got %d", rangeData.Count))
	}

	// Test range limit (should reject > 90 days)
	resp2, _ := tr.getRaw("/api/v1/readings/range?start=2025-01-01&end=2025-12-31")
	if resp2 != nil && resp2.StatusCode == 400 {
		tr.recordSuccess("Range limit enforced (>90 days rejected)")
	} else {
		tr.recordError("Range limit", "Should reject ranges > 90 days")
	}

	// Test invalid range (end before start)
	resp3, _ := tr.getRaw("/api/v1/readings/range?start=2025-12-31&end=2025-01-01")
	if resp3 != nil && resp3.StatusCode == 400 {
		tr.recordSuccess("Invalid range rejected (end before start)")
	} else {
		tr.recordError("Invalid range", "Should reject end < start")
	}
}

func (tr *TestRunner) testEdgeCases() {
	tr.printSection("Edge Cases")

	// Invalid date format
	resp, _ := tr.getRaw("/api/v1/readings/date/invalid")
	if resp != nil && resp.StatusCode == 400 {
		tr.recordSuccess("Invalid date format rejected")
	} else {
		tr.recordError("Invalid date", "Should return 400")
	}

	// Invalid date format (wrong separator)
	resp2, _ := tr.getRaw("/api/v1/readings/date/2025/12/25")
	if resp2 != nil && resp2.StatusCode != 200 {
		tr.recordSuccess("Wrong date format rejected")
	} else {
		tr.recordError("Wrong format", "Should reject 2025/12/25")
	}

	// Missing parameters for range
	resp3, _ := tr.getRaw("/api/v1/readings/range?start=2025-01-01")
	if resp3 != nil && resp3.StatusCode == 400 {
		tr.recordSuccess("Missing end parameter rejected")
	} else {
		tr.recordError("Missing param", "Should reject missing end")
	}

	// Leap year date
	resp4, err := tr.get("/api/v1/readings/date/2024-02-29")
	if err != nil {
		tr.recordError("Leap year", err.Error())
	} else {
		tr.recordSuccess("Leap year date (2024-02-29) handled")
	}
	_ = resp4

	// Far future date
	resp5, err := tr.get("/api/v1/readings/date/2030-06-15")
	if err != nil {
		tr.recordError("Future date", err.Error())
	} else {
		tr.recordSuccess("Far future date (2030) handled")
	}
	_ = resp5
}

func (tr *TestRunner) testLiturgicalSeasons() {
	tr.printSection("Full December 2025 (Advent → Christmas)")

	for day := 1; day <= 31; day++ {
		date := fmt.Sprintf("2025-12-%02d", day)
		resp, err := tr.get(fmt.Sprintf("/api/v1/readings/date/%s", date))
		if err != nil {
			tr.recordError(date, err.Error())
			continue
		}

		var data DateReadingsResponse
		if err := tr.parseDataAs(resp, &data); err != nil {
			tr.recordError(date, err.Error())
			continue
		}

		readings := data.Readings
		yearCycleStr := fmt.Sprintf("Y%d", readings.YearCycle)
		readingCount := len(readings.Readings)

		tr.recordSuccess(fmt.Sprintf("%s: %s / %s [%s, %d readings]",
			date, readings.Period, readings.DayIdentifier, yearCycleStr, readingCount))

		if tr.verbose {
			tr.printReadingsDetail(readings)
		}
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

func (tr *TestRunner) get(path string) (*APIResponse, error) {
	resp, err := tr.getRaw(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	if !apiResp.Success {
		errMsg := "unknown error"
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("API error: %s", errMsg)
	}

	return &apiResp, nil
}

func (tr *TestRunner) getRaw(path string) (*http.Response, error) {
	url := tr.baseURL + path
	return tr.client.Get(url)
}

func (tr *TestRunner) parseDataAs(resp *APIResponse, target interface{}) error {
	// Re-marshal and unmarshal to convert map to struct
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	return json.Unmarshal(dataBytes, target)
}

func (tr *TestRunner) printSection(name string) {
	fmt.Println()
	fmt.Printf("--- %s ---\n", name)
	fmt.Println()
}

func (tr *TestRunner) printReadingsDetail(r *DailyReadings) {
	if r == nil {
		return
	}
	if r.SpecialName != nil {
		fmt.Printf("    Special: %s\n", *r.SpecialName)
	}
	if len(r.MorningPsalms) > 0 {
		fmt.Printf("    Morning Psalms: %v\n", r.MorningPsalms)
	}
	if len(r.EveningPsalms) > 0 {
		fmt.Printf("    Evening Psalms: %v\n", r.EveningPsalms)
	}
	if len(r.Readings) > 0 {
		fmt.Printf("    Readings:\n")
		for _, reading := range r.Readings {
			fmt.Printf("      - %s: %s\n", reading.ReadingType, reading.Reference)
		}
	}
	fmt.Println()
}

func (tr *TestRunner) recordSuccess(msg string) {
	tr.successCount++
	fmt.Printf("  ✓ %s\n", msg)
}

func (tr *TestRunner) recordError(context, msg string) {
	tr.errorCount++
	errStr := fmt.Sprintf("%s: %s", context, msg)
	tr.errors = append(tr.errors, errStr)
	fmt.Printf("  ✗ %s\n", errStr)
}

func (tr *TestRunner) printSummary() {
	fmt.Println()
	fmt.Println("==============================================")
	fmt.Println("Summary")
	fmt.Println("==============================================")
	fmt.Printf("  Passed: %d\n", tr.successCount)
	fmt.Printf("  Failed: %d\n", tr.errorCount)
	fmt.Println()

	if tr.errorCount > 0 {
		fmt.Println("Failures:")
		for _, err := range tr.errors {
			fmt.Printf("  • %s\n", err)
		}
		fmt.Println()
	}

	if tr.errorCount == 0 {
		fmt.Println("All tests passed! ✓")
	} else {
		fmt.Printf("Tests completed with %d failure(s)\n", tr.errorCount)
	}
}

// =============================================================================
// Main
// =============================================================================

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Base URL of the API")
	verbose := flag.Bool("v", false, "Verbose output (show reading details)")
	flag.Parse()

	// Check if server is reachable
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(*baseURL + "/health")
	if err != nil {
		fmt.Printf("Error: Cannot connect to %s\n", *baseURL)
		fmt.Println("Make sure the API server is running.")
		os.Exit(1)
	}

	runner := NewTestRunner(*baseURL, *verbose)
	runner.Run()

	// Exit with error code if tests failed
	if runner.errorCount > 0 {
		os.Exit(1)
	}
}
