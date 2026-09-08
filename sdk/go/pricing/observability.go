package pricing

import (
	"errors"
	"fmt"
	"time"
)

// Constants for SLI targets and calculations.
const (
	// SLI target values.
	DefaultAvailabilityTarget  = 99.9  // 99.9% availability
	DefaultErrorRateTarget     = 0.1   // <0.1% error rate
	DefaultLatencyP99Target    = 2.0   // <2 seconds
	DefaultThroughputTarget    = 100.0 // >100 RPS
	DefaultDataFreshnessTarget = 24.0  // <24 hours old

	// Unit conversion constants.
	PercentageMultiplier      = 100.0     // Convert ratio to percentage
	NanosecondsToMicroseconds = 1000.0    // Convert ns to μs
	NanosecondsToMilliseconds = 1000000.0 // Convert ns to ms

	// Metric unit labels.
	unitPercentage = "percentage"
	unitSeconds    = "seconds"

	// Feature names.
	healthCheckFeature = "health_check"
)

// MetricType represents the different types of metrics that can be collected.
type MetricType string

const (
	// MetricTypeCounter represents counter metrics that increment over time (e.g., request counts, error counts).
	MetricTypeCounter MetricType = "counter"
	// MetricTypeGauge represents gauge metrics with current values that can go up or down (e.g., active connections).
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeHistogram represents histogram metrics that track distributions of values (e.g., response times).
	MetricTypeHistogram MetricType = "histogram"
	// MetricTypeSummary represents summary metrics that provide quantile calculations.
	MetricTypeSummary MetricType = "summary"
)

// StandardMetrics defines the core metrics that all plugins should expose.
//
//nolint:gochecknoglobals // Standard constants for public API
var StandardMetrics = struct {
	// Request metrics
	RequestsTotal          string
	RequestDurationSeconds string
	RequestSizeBytes       string
	ResponseSizeBytes      string

	// Error metrics
	ErrorsTotal      string
	ErrorRatePercent string

	// Performance metrics
	LatencyP50Seconds string
	LatencyP95Seconds string
	LatencyP99Seconds string

	// Resource metrics
	ActiveConnections string
	MemoryUsageBytes  string
	CPUUsagePercent   string

	// Business metrics
	CostQueriesTotal         string
	CacheHitRatePercent      string
	DataSourceLatencySeconds string
}{
	RequestsTotal:          "finfocus_requests_total",
	RequestDurationSeconds: "finfocus_request_duration_seconds",
	RequestSizeBytes:       "finfocus_request_size_bytes",
	ResponseSizeBytes:      "finfocus_response_size_bytes",

	ErrorsTotal:      "finfocus_errors_total",
	ErrorRatePercent: "finfocus_error_rate_percent",

	LatencyP50Seconds: "finfocus_latency_p50_seconds",
	LatencyP95Seconds: "finfocus_latency_p95_seconds",
	LatencyP99Seconds: "finfocus_latency_p99_seconds",

	ActiveConnections: "finfocus_active_connections",
	MemoryUsageBytes:  "finfocus_memory_usage_bytes",
	CPUUsagePercent:   "finfocus_cpu_usage_percent",

	CostQueriesTotal:         "finfocus_cost_queries_total",
	CacheHitRatePercent:      "finfocus_cache_hit_rate_percent",
	DataSourceLatencySeconds: "finfocus_data_source_latency_seconds",
}

// StandardLabels defines the standard label keys used across metrics.
//
//nolint:gochecknoglobals // Standard constants for public API
var StandardLabels = struct {
	Method       string
	Provider     string
	ResourceType string
	Region       string
	ErrorType    string
	Status       string
	CacheStatus  string
}{
	Method:       "method",
	Provider:     "provider",
	ResourceType: "resource_type",
	Region:       "region",
	ErrorType:    "error_type",
	Status:       "status",
	CacheStatus:  "cache_status",
}

// ServiceLevelIndicator represents standard SLIs that plugins should track.
type ServiceLevelIndicator struct {
	Name        string
	Description string
	Unit        string
	Target      float64
}

// StandardSLIs defines the core SLIs that plugins should measure.
//
//nolint:gochecknoglobals // Standard constants for public API
var StandardSLIs = struct {
	Availability  ServiceLevelIndicator
	ErrorRate     ServiceLevelIndicator
	LatencyP99    ServiceLevelIndicator
	LatencyP95    ServiceLevelIndicator
	Throughput    ServiceLevelIndicator
	DataFreshness ServiceLevelIndicator
}{
	Availability: ServiceLevelIndicator{
		Name:        "availability",
		Description: "Percentage of successful requests over total requests",
		Unit:        unitPercentage,
		Target:      DefaultAvailabilityTarget,
	},
	ErrorRate: ServiceLevelIndicator{
		Name:        "error_rate",
		Description: "Percentage of requests that result in errors",
		Unit:        unitPercentage,
		Target:      DefaultErrorRateTarget,
	},
	LatencyP99: ServiceLevelIndicator{
		Name:        "latency_p99",
		Description: "99th percentile response latency",
		Unit:        unitSeconds,
		Target:      DefaultLatencyP99Target,
	},
	LatencyP95: ServiceLevelIndicator{
		Name:        "latency_p95",
		Description: "95th percentile response latency",
		Unit:        unitSeconds,
		Target:      1.0, // <1 second
	},
	Throughput: ServiceLevelIndicator{
		Name:        "throughput",
		Description: "Requests processed per second",
		Unit:        "requests_per_second",
		Target:      DefaultThroughputTarget,
	},
	DataFreshness: ServiceLevelIndicator{
		Name:        "data_freshness",
		Description: "Age of the most recent cost data",
		Unit:        "hours",
		Target:      DefaultDataFreshnessTarget,
	},
}

// LogLevel represents the different logging levels.
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogContext provides structured context for logging operations.
type LogContext struct {
	TraceID   string
	SpanID    string
	RequestID string
	Method    string
	Provider  string
	Region    string
	UserID    string
	SessionID string
	Component string
}

// ObservabilityErrorCategory represents different categories of errors for classification in observability contexts.
type ObservabilityErrorCategory string

const (
	ErrorCategoryNetwork    ObservabilityErrorCategory = "network"
	ErrorCategoryAuth       ObservabilityErrorCategory = "auth"
	ErrorCategoryData       ObservabilityErrorCategory = "data"
	ErrorCategoryValidation ObservabilityErrorCategory = "validation"
	ErrorCategoryRateLimit  ObservabilityErrorCategory = "rate_limit"
	ErrorCategoryInternal   ObservabilityErrorCategory = "internal"
	ErrorCategoryUpstream   ObservabilityErrorCategory = "upstream"
	ErrorCategoryObsConfig  ObservabilityErrorCategory = "configuration"
)

// HealthStatus represents the health status values.
type HealthStatus string

const (
	HealthStatusUnknown        HealthStatus = "UNKNOWN"
	HealthStatusServing        HealthStatus = "SERVING"
	HealthStatusNotServing     HealthStatus = "NOT_SERVING"
	HealthStatusServiceUnknown HealthStatus = "SERVICE_UNKNOWN"
)

// ValidationResult represents the result of validating observability data.
type ValidationResult struct {
	Valid    bool             `json:"valid"`
	Errors   []string         `json:"errors,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
	Bypasses []BypassMetadata `json:"bypasses,omitempty"`
}

// ValidateMetricName checks if a metric name follows standard conventions.
func ValidateMetricName(name string) error {
	if name == "" {
		return errors.New("metric name cannot be empty")
	}

	// Metric names should follow Prometheus naming conventions
	// Should contain only [a-zA-Z0-9:_] and not start with digit
	if len(name) == 0 {
		return errors.New("metric name is required")
	}

	// Basic validation - in practice you'd want more comprehensive regex validation
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("metric name cannot start with a digit: %s", name)
	}

	return nil
}

// ValidateMetricLabels ensures metric labels follow best practices.
func ValidateMetricLabels(labels map[string]string) ValidationResult {
	result := ValidationResult{Valid: true}

	for key, value := range labels {
		if key == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "label key cannot be empty")
			continue
		}

		if value == "" {
			result.Warnings = append(
				result.Warnings,
				fmt.Sprintf("label '%s' has empty value", key),
			)
		}

		// Label keys should not start with __
		if len(key) >= 2 && key[:2] == "__" {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("label key '%s' cannot start with '__' (reserved prefix)", key))
		}
	}

	return result
}

// ValidateSLIValue checks if an SLI value is within acceptable ranges.
func ValidateSLIValue(sli ServiceLevelIndicator, value float64) error {
	switch sli.Unit {
	case unitPercentage:
		if value < 0 || value > 100 {
			return fmt.Errorf(
				"percentage SLI '%s' must be between 0 and 100, got %f",
				sli.Name,
				value,
			)
		}
	case unitSeconds:
		if value < 0 {
			return fmt.Errorf("time-based SLI '%s' cannot be negative, got %f", sli.Name, value)
		}
	case "ratio":
		if value < 0 || value > 1 {
			return fmt.Errorf("ratio SLI '%s' must be between 0 and 1, got %f", sli.Name, value)
		}
	}

	return nil
}

// CalculateErrorRate computes error rate from success and total counts.
func CalculateErrorRate(totalRequests, successfulRequests int64) float64 {
	if totalRequests == 0 {
		return 0.0
	}

	errorRequests := totalRequests - successfulRequests
	return float64(errorRequests) / float64(totalRequests) * PercentageMultiplier
}

// CalculateAvailability computes availability from uptime and total time.
func CalculateAvailability(uptimeSeconds, totalSeconds int64) float64 {
	if totalSeconds == 0 {
		return 0.0
	}

	return float64(uptimeSeconds) / float64(totalSeconds) * PercentageMultiplier
}

// FormatDuration formats a duration for human-readable display.
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/NanosecondsToMicroseconds)
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/NanosecondsToMilliseconds)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// ConformanceLevel represents different levels of observability conformance.
type ConformanceLevel string

const (
	ConformanceBasic    ConformanceLevel = "basic"
	ConformanceStandard ConformanceLevel = "standard"
	ConformanceAdvanced ConformanceLevel = "advanced"
)

// ObservabilityRequirements defines what metrics and SLIs are required for each conformance level.
type ObservabilityRequirements struct {
	Level            ConformanceLevel
	RequiredMetrics  []string
	RequiredSLIs     []string
	RequiredFeatures []string
}

// GetObservabilityRequirements returns the requirements for a given conformance level.
func GetObservabilityRequirements(level ConformanceLevel) ObservabilityRequirements {
	switch level {
	case ConformanceBasic:
		return ObservabilityRequirements{
			Level: ConformanceBasic,
			RequiredMetrics: []string{
				StandardMetrics.RequestsTotal,
				StandardMetrics.ErrorsTotal,
				StandardMetrics.RequestDurationSeconds,
			},
			RequiredSLIs: []string{
				StandardSLIs.Availability.Name,
				StandardSLIs.ErrorRate.Name,
			},
			RequiredFeatures: []string{
				healthCheckFeature,
				"basic_metrics",
			},
		}
	case ConformanceStandard:
		return ObservabilityRequirements{
			Level: ConformanceStandard,
			RequiredMetrics: []string{
				StandardMetrics.RequestsTotal,
				StandardMetrics.ErrorsTotal,
				StandardMetrics.RequestDurationSeconds,
				StandardMetrics.LatencyP95Seconds,
				StandardMetrics.CacheHitRatePercent,
				StandardMetrics.ActiveConnections,
			},
			RequiredSLIs: []string{
				StandardSLIs.Availability.Name,
				StandardSLIs.ErrorRate.Name,
				StandardSLIs.LatencyP95.Name,
				StandardSLIs.Throughput.Name,
			},
			RequiredFeatures: []string{
				healthCheckFeature,
				"metrics_endpoint",
				"structured_logging",
				"basic_tracing",
			},
		}
	case ConformanceAdvanced:
		return ObservabilityRequirements{
			Level: ConformanceAdvanced,
			RequiredMetrics: []string{
				StandardMetrics.RequestsTotal,
				StandardMetrics.ErrorsTotal,
				StandardMetrics.RequestDurationSeconds,
				StandardMetrics.LatencyP95Seconds,
				StandardMetrics.LatencyP99Seconds,
				StandardMetrics.CacheHitRatePercent,
				StandardMetrics.ActiveConnections,
				StandardMetrics.MemoryUsageBytes,
				StandardMetrics.CPUUsagePercent,
				StandardMetrics.DataSourceLatencySeconds,
			},
			RequiredSLIs: []string{
				StandardSLIs.Availability.Name,
				StandardSLIs.ErrorRate.Name,
				StandardSLIs.LatencyP95.Name,
				StandardSLIs.LatencyP99.Name,
				StandardSLIs.Throughput.Name,
				StandardSLIs.DataFreshness.Name,
			},
			RequiredFeatures: []string{
				healthCheckFeature,
				"metrics_endpoint",
				"structured_logging",
				"distributed_tracing",
				"sli_endpoint",
				"custom_metrics",
				"performance_monitoring",
			},
		}
	default:
		return GetObservabilityRequirements(ConformanceBasic)
	}
}
