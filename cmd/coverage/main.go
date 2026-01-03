package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// APIResponse matches the API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

type ErrorInfo struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type DateReadingsResponse struct {
	Date     string         `json:"date"`
	Readings *DailyReadings `json:"readings"`
}

type DailyReadings struct {
	Period        string    `json:"period"`
	DayIdentifier string    `json:"day_identifier"`
	YearCycle     int       `json:"year_cycle"`
	Readings      []Reading `json:"readings"`
}

type Reading struct {
	ID          int64  `json:"id"`
	ReadingType string `json:"reading_type"`
	Reference   string `json:"reference"`
}

// TestResult holds the result for a single date
type TestResult struct {
	Date          string
	Success       bool
	Period        string
	DayIdentifier string
	YearCycle     int
	ReadingCount  int
	Error         string
}

// PeriodStats tracks statistics for each period
type PeriodStats struct {
	Period      string
	TotalDays   int
	SuccessDays int
	FailedDays  int
	FailedDates []string
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "Base URL of the API")
	startYear := flag.Int("start", 2024, "Start year")
	years := flag.Int("years", 4, "Number of years to test")
	verbose := flag.Bool("v", false, "Verbose output (show each date)")
	outputFile := flag.String("o", "", "Output results to JSON file")
	flag.Parse()

	endYear := *startYear + *years - 1

	fmt.Println("================================================================")
	fmt.Println("Lectionary API - Full Coverage Test")
	fmt.Println("================================================================")
	fmt.Printf("Base URL:    %s\n", *baseURL)
	fmt.Printf("Date Range:  %d-01-01 to %d-12-31\n", *startYear, endYear)
	fmt.Printf("Total Years: %d\n", *years)
	fmt.Println()

	// Check if server is reachable
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := client.Get(*baseURL + "/health")
	if err != nil {
		fmt.Printf("Error: Cannot connect to %s\n", *baseURL)
		fmt.Println("Make sure the API server is running.")
		os.Exit(1)
	}

	// Test all dates
	results := testAllDates(client, *baseURL, *startYear, endYear, *verbose)

	// Analyze results
	analysis := analyzeResults(results)

	// Print summary
	printSummary(analysis, *startYear, endYear)

	// Print failures by period
	printFailuresByPeriod(analysis)

	// Print all failures
	printAllFailures(analysis)

	// Output to file if requested
	if *outputFile != "" {
		saveResults(*outputFile, results, analysis)
	}

	// Exit with error code if there were failures
	if analysis.TotalFailed > 0 {
		os.Exit(1)
	}
}

func testAllDates(client *http.Client, baseURL string, startYear, endYear int, verbose bool) []TestResult {
	var results []TestResult

	totalDays := 0
	for year := startYear; year <= endYear; year++ {
		start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
		totalDays += int(end.Sub(start).Hours()/24) + 1
	}

	fmt.Printf("Testing %d days...\n\n", totalDays)

	tested := 0
	failed := 0
	lastProgress := -1

	for year := startYear; year <= endYear; year++ {
		current := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endOfYear := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)

		for !current.After(endOfYear) {
			dateStr := current.Format("2006-01-02")
			result := testDate(client, baseURL, dateStr)
			results = append(results, result)

			tested++
			if !result.Success {
				failed++
			}

			// Show progress
			progress := (tested * 100) / totalDays
			if progress != lastProgress && progress%5 == 0 {
				fmt.Printf("  Progress: %d%% (%d/%d) - Failures: %d\n", progress, tested, totalDays, failed)
				lastProgress = progress
			}

			if verbose {
				status := "✓"
				if !result.Success {
					status = "✗"
				}
				fmt.Printf("  %s %s: %s / %s [Y%d, %d readings]\n",
					status, dateStr, result.Period, result.DayIdentifier,
					result.YearCycle, result.ReadingCount)
				if !result.Success {
					fmt.Printf("      Error: %s\n", result.Error)
				}
			}

			current = current.AddDate(0, 0, 1)
		}
	}

	fmt.Println()
	return results
}

func testDate(client *http.Client, baseURL, dateStr string) TestResult {
	result := TestResult{Date: dateStr}

	url := fmt.Sprintf("%s/api/v1/readings/date/%s", baseURL, dateStr)
	resp, err := client.Get(url)
	if err != nil {
		result.Error = fmt.Sprintf("Connection error: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("Read error: %v", err)
		return result
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		result.Error = fmt.Sprintf("Parse error: %v", err)
		return result
	}

	if !apiResp.Success {
		errMsg := "Unknown error"
		if apiResp.Error != nil {
			errMsg = apiResp.Error.Message
		}
		result.Error = errMsg
		return result
	}

	// Parse the successful response
	dataBytes, _ := json.Marshal(apiResp.Data)
	var data DateReadingsResponse
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		result.Error = fmt.Sprintf("Data parse error: %v", err)
		return result
	}

	if data.Readings == nil {
		result.Error = "No readings data in response"
		return result
	}

	result.Success = true
	result.Period = data.Readings.Period
	result.DayIdentifier = data.Readings.DayIdentifier
	result.YearCycle = data.Readings.YearCycle
	result.ReadingCount = len(data.Readings.Readings)

	// Check if we got actual readings
	if result.ReadingCount == 0 {
		result.Success = false
		result.Error = "No readings returned (empty array)"
	}

	return result
}

// Analysis holds the analyzed results
type Analysis struct {
	TotalDays    int
	TotalSuccess int
	TotalFailed  int
	ByPeriod     map[string]*PeriodStats
	ByYear       map[int]*YearStats
	ByMonth      map[string]*MonthStats
	AllFailures  []TestResult
}

type YearStats struct {
	Year        int
	TotalDays   int
	SuccessDays int
	FailedDays  int
}

type MonthStats struct {
	YearMonth   string
	TotalDays   int
	SuccessDays int
	FailedDays  int
}

func analyzeResults(results []TestResult) *Analysis {
	analysis := &Analysis{
		ByPeriod: make(map[string]*PeriodStats),
		ByYear:   make(map[int]*YearStats),
		ByMonth:  make(map[string]*MonthStats),
	}

	for _, r := range results {
		analysis.TotalDays++

		// Parse date for year/month stats
		date, _ := time.Parse("2006-01-02", r.Date)
		year := date.Year()
		yearMonth := date.Format("2006-01")

		// Year stats
		if _, ok := analysis.ByYear[year]; !ok {
			analysis.ByYear[year] = &YearStats{Year: year}
		}
		analysis.ByYear[year].TotalDays++

		// Month stats
		if _, ok := analysis.ByMonth[yearMonth]; !ok {
			analysis.ByMonth[yearMonth] = &MonthStats{YearMonth: yearMonth}
		}
		analysis.ByMonth[yearMonth].TotalDays++

		// Period stats
		period := r.Period
		if period == "" {
			period = "(resolution failed)"
		}
		if _, ok := analysis.ByPeriod[period]; !ok {
			analysis.ByPeriod[period] = &PeriodStats{Period: period}
		}
		analysis.ByPeriod[period].TotalDays++

		if r.Success {
			analysis.TotalSuccess++
			analysis.ByYear[year].SuccessDays++
			analysis.ByMonth[yearMonth].SuccessDays++
			analysis.ByPeriod[period].SuccessDays++
		} else {
			analysis.TotalFailed++
			analysis.ByYear[year].FailedDays++
			analysis.ByMonth[yearMonth].FailedDays++
			analysis.ByPeriod[period].FailedDays++
			analysis.ByPeriod[period].FailedDates = append(analysis.ByPeriod[period].FailedDates, r.Date)
			analysis.AllFailures = append(analysis.AllFailures, r)
		}
	}

	return analysis
}

func printSummary(analysis *Analysis, startYear, endYear int) {
	fmt.Println("================================================================")
	fmt.Println("SUMMARY")
	fmt.Println("================================================================")
	fmt.Printf("Total Days Tested: %d\n", analysis.TotalDays)
	fmt.Printf("Successful:        %d (%.1f%%)\n", analysis.TotalSuccess,
		float64(analysis.TotalSuccess)/float64(analysis.TotalDays)*100)
	fmt.Printf("Failed:            %d (%.1f%%)\n", analysis.TotalFailed,
		float64(analysis.TotalFailed)/float64(analysis.TotalDays)*100)
	fmt.Println()

	// By year
	fmt.Println("By Year:")
	for year := startYear; year <= endYear; year++ {
		if stats, ok := analysis.ByYear[year]; ok {
			status := "✓"
			if stats.FailedDays > 0 {
				status = "✗"
			}
			fmt.Printf("  %s %d: %d/%d days (%.1f%% success)\n",
				status, year, stats.SuccessDays, stats.TotalDays,
				float64(stats.SuccessDays)/float64(stats.TotalDays)*100)
		}
	}
	fmt.Println()
}

func printFailuresByPeriod(analysis *Analysis) {
	if analysis.TotalFailed == 0 {
		fmt.Println("No failures! 🎉")
		return
	}

	fmt.Println("================================================================")
	fmt.Println("FAILURES BY PERIOD")
	fmt.Println("================================================================")

	// Sort periods by failure count
	var periods []*PeriodStats
	for _, stats := range analysis.ByPeriod {
		if stats.FailedDays > 0 {
			periods = append(periods, stats)
		}
	}
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].FailedDays > periods[j].FailedDays
	})

	for _, stats := range periods {
		fmt.Printf("\n%s: %d failures\n", stats.Period, stats.FailedDays)
		// Show up to 5 example dates
		shown := 0
		for _, date := range stats.FailedDates {
			if shown >= 5 {
				fmt.Printf("  ... and %d more\n", len(stats.FailedDates)-5)
				break
			}
			fmt.Printf("  - %s\n", date)
			shown++
		}
	}
	fmt.Println()
}

func printAllFailures(analysis *Analysis) {
	if analysis.TotalFailed == 0 {
		return
	}

	if analysis.TotalFailed > 50 {
		fmt.Printf("(Showing first 50 of %d failures)\n\n", analysis.TotalFailed)
	}

	fmt.Println("================================================================")
	fmt.Println("ALL FAILURES (Date | Period | Error)")
	fmt.Println("================================================================")

	// Group by error type
	errorGroups := make(map[string][]TestResult)
	for _, f := range analysis.AllFailures {
		errorGroups[f.Error] = append(errorGroups[f.Error], f)
	}

	shown := 0
	for errorType, failures := range errorGroups {
		fmt.Printf("\nError: %s (%d occurrences)\n", errorType, len(failures))
		for _, f := range failures {
			if shown >= 50 {
				break
			}
			period := f.Period
			if period == "" {
				period = "(unresolved)"
			}
			fmt.Printf("  %s | %s / %s\n", f.Date, period, f.DayIdentifier)
			shown++
		}
		if shown >= 50 {
			break
		}
	}
	fmt.Println()
}

func saveResults(filename string, results []TestResult, analysis *Analysis) {
	output := struct {
		GeneratedAt string                  `json:"generated_at"`
		Summary     map[string]interface{}  `json:"summary"`
		ByPeriod    map[string]*PeriodStats `json:"by_period"`
		Failures    []TestResult            `json:"failures"`
	}{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Summary: map[string]interface{}{
			"total_days":    analysis.TotalDays,
			"total_success": analysis.TotalSuccess,
			"total_failed":  analysis.TotalFailed,
			"success_rate":  fmt.Sprintf("%.2f%%", float64(analysis.TotalSuccess)/float64(analysis.TotalDays)*100),
		},
		ByPeriod: analysis.ByPeriod,
		Failures: analysis.AllFailures,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Printf("Results saved to: %s\n", filename)
}

// Helper to categorize periods for grouping
func categorizePeriod(period string) string {
	period = strings.ToLower(period)
	switch {
	case strings.Contains(period, "advent"):
		return "Advent"
	case strings.Contains(period, "christmas"):
		return "Christmas"
	case strings.Contains(period, "epiphany"):
		return "Epiphany"
	case strings.Contains(period, "baptism"):
		return "Baptism"
	case strings.Contains(period, "lent"):
		return "Lent"
	case strings.Contains(period, "holy week"):
		return "Holy Week"
	case strings.Contains(period, "easter"):
		return "Easter"
	case strings.Contains(period, "pentecost"):
		return "Pentecost"
	case strings.Contains(period, "trinity"):
		return "Trinity"
	default:
		return "Other"
	}
}
