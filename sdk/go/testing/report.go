// Package testing provides a comprehensive testing framework for FinFocus plugins.
// This file implements reporting functionality for conformance test results.
package testing

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Report format constants.
const (
	// ReportVersion is the current version of the conformance report schema.
	ReportVersion = "1.0.0"

	// Report column widths for formatted output.
	reportPluginNameWidth = 56
	reportMethodWidth     = 60
	reportErrorMsgWidth   = 58
)

// ToJSON serializes the conformance result to JSON format.
func (r *ConformanceResult) ToJSON() ([]byte, error) {
	// Ensure string representations are set
	r.LevelAchievedStr = r.LevelAchieved.String()
	r.DurationStr = r.Duration.String()

	return json.MarshalIndent(r, "", "  ")
}

// PrintReport prints a human-readable summary of the conformance result to stdout.
func PrintReport(result *ConformanceResult) {
	PrintReportTo(result, os.Stdout)
}

// PrintReportTo prints a human-readable summary of the conformance result to the writer.
//
//nolint:gocognit // Report printing inherently requires sequential output
func PrintReportTo(result *ConformanceResult, w io.Writer) {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "╔══════════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(w, "║               Plugin Conformance Test Report                     ║\n")
	fmt.Fprintf(w, "╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Plugin: %-56s ║\n", truncate(result.PluginName, reportPluginNameWidth))
	fmt.Fprintf(w, "║ Level Achieved: %-48s ║\n", result.LevelAchieved.String())
	fmt.Fprintf(w, "║ Duration: %-54s ║\n", result.Duration.String())
	fmt.Fprintf(w, "╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Summary                                                          ║\n")
	fmt.Fprintf(w, "╠──────────────────────────────────────────────────────────────────╣\n")
	fmt.Fprintf(w, "║   Total:   %-53d ║\n", result.Summary.Total)
	fmt.Fprintf(w, "║   Passed:  %-53d ║\n", result.Summary.Passed)
	fmt.Fprintf(w, "║   Failed:  %-53d ║\n", result.Summary.Failed)
	fmt.Fprintf(w, "║   Skipped: %-53d ║\n", result.Summary.Skipped)
	fmt.Fprintf(w, "╠══════════════════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(w, "║ Categories                                                       ║\n")
	fmt.Fprintf(w, "╠──────────────────────────────────────────────────────────────────╣\n")

	// Print category results
	categories := []TestCategory{
		CategorySpecValidation,
		CategoryRPCCorrectness,
		CategoryPerformance,
		CategoryConcurrency,
	}

	for _, cat := range categories {
		if catResult, ok := result.Categories[cat]; ok {
			status := "✓"
			if catResult.Failed > 0 {
				status = "✗"
			}
			fmt.Fprintf(w, "║   %s %-16s  Passed: %2d  Failed: %2d  Skipped: %2d      ║\n",
				status, cat.String(), catResult.Passed, catResult.Failed, catResult.Skipped)
		}
	}

	fmt.Fprintf(w, "╠══════════════════════════════════════════════════════════════════╣\n")

	// Print failed tests if any
	if result.Summary.Failed > 0 {
		fmt.Fprintf(w, "║ Failed Tests                                                     ║\n")
		fmt.Fprintf(w, "╠──────────────────────────────────────────────────────────────────╣\n")
		for _, catResult := range result.Categories {
			for _, testResult := range catResult.Results {
				if !testResult.Success {
					fmt.Fprintf(w, "║   • %-60s ║\n", truncate(testResult.Method, reportMethodWidth))
					if testResult.Error != nil {
						errMsg := truncate(testResult.Error.Error(), reportErrorMsgWidth)
						fmt.Fprintf(w, "║     Error: %-53s ║\n", errMsg)
					}
				}
			}
		}
		fmt.Fprintf(w, "╠══════════════════════════════════════════════════════════════════╣\n")
	}

	// Final verdict
	if result.Passed() {
		fmt.Fprintf(w, "║                         ✓ ALL TESTS PASSED                       ║\n")
	} else {
		fmt.Fprintf(w, "║                         ✗ SOME TESTS FAILED                      ║\n")
	}
	fmt.Fprintf(w, "╚══════════════════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(w, "\n")
}

// truncate truncates a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatCategoryResults formats category results as a multi-line string.
func FormatCategoryResults(categories map[TestCategory]*CategoryResult) string {
	var sb strings.Builder

	for cat, result := range categories {
		fmt.Fprintf(&sb, "%s: %d passed, %d failed, %d skipped\n",
			cat.String(), result.Passed, result.Failed, result.Skipped)
	}

	return sb.String()
}
