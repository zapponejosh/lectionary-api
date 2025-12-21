package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Data    *Data  `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Data struct {
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
	Reference   string `json:"reference"`
}

type Error struct {
	Message string `json:"message"`
}

func main() {
	baseURL := "http://localhost:8080/api/v1/readings/date"
	year := 2025

	fmt.Printf("Testing all December dates for %d...\n", year)
	fmt.Println("==========================================")
	fmt.Println()

	successCount := 0
	errorCount := 0
	var errors []string

	for day := 1; day <= 31; day++ {
		date := fmt.Sprintf("%d-12-%02d", year, day)
		url := fmt.Sprintf("%s/%s", baseURL, date)

		fmt.Printf("Testing %s... ", date)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("✗ Connection error: %v\n", err)
			errorCount++
			errors = append(errors, fmt.Sprintf("%s: Connection error", date))
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("✗ Read error: %v\n", err)
			errorCount++
			errors = append(errors, fmt.Sprintf("%s: Read error", date))
			continue
		}

		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			fmt.Printf("✗ Parse error: %v\n", err)
			errorCount++
			errors = append(errors, fmt.Sprintf("%s: Parse error", date))
			continue
		}

		if resp.StatusCode == 200 && apiResp.Success && apiResp.Data != nil {
			data := apiResp.Data
			period := data.Period
			if period != "" {
				fmt.Printf("✓ %s (%s)\n", period, data.DayIdentifier)
				if data.SpecialName != nil {
					fmt.Printf("  Special: %s\n", *data.SpecialName)
				}
				fmt.Printf("  Year Cycle: %d\n", data.YearCycle)

				if len(data.MorningPsalms) > 0 {
					fmt.Printf("  Morning Psalms: %v\n", data.MorningPsalms)
				}
				if len(data.EveningPsalms) > 0 {
					fmt.Printf("  Evening Psalms: %v\n", data.EveningPsalms)
				}

				if len(data.Readings) > 0 {
					fmt.Printf("  Readings:\n")
					for _, reading := range data.Readings {
						fmt.Printf("    - %s: %s\n", reading.ReadingType, reading.Reference)
					}
				} else {
					fmt.Printf("  Readings: (none)\n")
				}
				successCount++
			} else {
				fmt.Printf("✗ No period found\n")
				errorCount++
				errors = append(errors, fmt.Sprintf("%s: No period in response", date))
			}
		} else {
			errorMsg := "Unknown error"
			if apiResp.Error != nil {
				errorMsg = apiResp.Error.Message
			}
			fmt.Printf("✗ HTTP %d: %s\n", resp.StatusCode, errorMsg)
			errorCount++
			errors = append(errors, fmt.Sprintf("%s: HTTP %d - %s", date, resp.StatusCode, errorMsg))
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Printf("Summary:\n")
	fmt.Printf("  Success: %d\n", successCount)
	fmt.Printf("  Errors:  %d\n", errorCount)
	fmt.Println()

	if errorCount > 0 {
		fmt.Println("Errors:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
	}
}
