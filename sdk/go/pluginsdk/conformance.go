// Package pluginsdk provides a development SDK for FinFocus plugins.
// This file provides adapter functions for running conformance tests directly
// on Plugin implementations without manual conversion to CostSourceServiceServer.
package pluginsdk

import (
	"errors"
	"io"
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// Type aliases for convenient access to testing package types.
// These allow plugin developers to work entirely within the pluginsdk package
// without needing to import sdk/go/testing directly.
type (
	// ConformanceResult contains the complete result of conformance suite execution.
	// This struct is JSON-serializable for CI/CD integration.
	ConformanceResult = plugintesting.ConformanceResult

	// ConformanceLevel defines the certification level for plugin validation.
	// Higher levels include all tests from lower levels.
	ConformanceLevel = plugintesting.ConformanceLevel

	// ResultSummary contains aggregate test counts.
	ResultSummary = plugintesting.ResultSummary

	// ConformanceOption adjusts the configuration RunConformance uses for a level.
	ConformanceOption = plugintesting.ConformanceOption
)

// Conformance level constants for convenient access.
const (
	// ConformanceLevelBasic - Core functionality, required for all plugins.
	ConformanceLevelBasic = plugintesting.ConformanceLevelBasic

	// ConformanceLevelStandard - Production readiness, recommended for deployment.
	ConformanceLevelStandard = plugintesting.ConformanceLevelStandard

	// ConformanceLevelAdvanced - High performance, for demanding environments.
	ConformanceLevelAdvanced = plugintesting.ConformanceLevelAdvanced
)

// ErrNilPlugin is returned when a nil plugin is passed to adapter functions.
var ErrNilPlugin = errors.New("plugin cannot be nil")

// validatePlugin validates that the plugin is not nil.
// This validation runs before any server creation to provide clear error messages.
func validatePlugin(plugin Plugin) error {
	if plugin == nil {
		return ErrNilPlugin
	}
	return nil
}

// WithSampleResource makes every check that sends a resource descriptor send a copy
// of r. Supply a resource your plugin prices; the default is
// plugintesting.DefaultSampleResource(). A nil r restores the default.
func WithSampleResource(r *pbc.ResourceDescriptor) ConformanceOption {
	return plugintesting.WithSampleResource(r)
}

// RunConformance runs the checks for level against a Plugin implementation, adjusted
// by opts. It returns ErrNilPlugin for a nil plugin, and an error for an unknown level
// or an invalid sample resource.
func RunConformance(plugin Plugin, level ConformanceLevel, opts ...ConformanceOption) (*ConformanceResult, error) {
	if err := validatePlugin(plugin); err != nil {
		return nil, err
	}

	return plugintesting.RunConformance(NewServer(plugin), level, opts...)
}

// RunBasicConformance runs basic conformance tests against a Plugin implementation.
// Basic conformance validates core functionality required for all plugins:
// - Plugin name validation
// - Supports handling
// - Basic GetProjectedCost/GetPricingSpec behavior
//
// Returns an error if the plugin is nil.
func RunBasicConformance(plugin Plugin) (*ConformanceResult, error) {
	if err := validatePlugin(plugin); err != nil {
		return nil, err
	}

	server := NewServer(plugin)
	return plugintesting.RunBasicConformance(server)
}

// RunStandardConformance runs standard conformance tests against a Plugin implementation.
// Standard conformance includes all Basic tests plus:
// - Error handling validation
// - Response consistency
// - 24-hour data range support
// - 10 concurrent request handling
//
// Returns an error if the plugin is nil.
func RunStandardConformance(plugin Plugin) (*ConformanceResult, error) {
	if err := validatePlugin(plugin); err != nil {
		return nil, err
	}

	server := NewServer(plugin)
	return plugintesting.RunStandardConformance(server)
}

// RunAdvancedConformance runs advanced conformance tests against a Plugin implementation.
// Advanced conformance includes all Standard tests plus:
// - Strict latency thresholds
// - 50 concurrent request handling
// - 30-day data range support
// - Memory efficiency requirements
//
// Returns an error if the plugin is nil.
func RunAdvancedConformance(plugin Plugin) (*ConformanceResult, error) {
	if err := validatePlugin(plugin); err != nil {
		return nil, err
	}

	server := NewServer(plugin)
	return plugintesting.RunAdvancedConformance(server)
}

// testLogWriter adapts *testing.T to io.Writer for report output.
type testLogWriter struct {
	t *testing.T
}

// Write implements io.Writer by logging to the test output.
func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// PrintConformanceReport prints a formatted conformance report to the test log.
// The report includes:
// - Plugin name and conformance level achieved
// - Test duration
// - Summary counts (total, passed, failed, skipped)
// - Per-category results
// - Failed test details with error messages
//
// If result is nil, a warning message is logged instead of panicking.
func PrintConformanceReport(t *testing.T, result *ConformanceResult) {
	if result == nil {
		t.Log("Warning: conformance result is nil, skipping report")
		return
	}

	writer := &testLogWriter{t: t}
	plugintesting.PrintReportTo(result, writer)
}

// PrintConformanceReportTo prints a formatted conformance report to any io.Writer.
// This is useful for outputting reports to files, buffers, or custom destinations.
//
// If result is nil, the function returns immediately without writing anything.
func PrintConformanceReportTo(result *ConformanceResult, w io.Writer) {
	if result == nil {
		return
	}
	plugintesting.PrintReportTo(result, w)
}
