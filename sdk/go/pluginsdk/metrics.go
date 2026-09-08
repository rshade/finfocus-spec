// Package pluginsdk provides Prometheus metrics instrumentation for FinFocus plugins.
package pluginsdk

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ============================================================================
// Constants
// ============================================================================

const (
	// MetricNamespace is the Prometheus namespace for all plugin metrics.
	MetricNamespace = "finfocus"

	// MetricSubsystem is the Prometheus subsystem for plugin metrics.
	MetricSubsystem = "plugin"

	// DefaultMetricsPort is the default port for the metrics HTTP server.
	DefaultMetricsPort = 9090

	// DefaultMetricsPath is the default URL path for metrics.
	DefaultMetricsPath = "/metrics"

	// serverReadHeaderTimeout is the timeout for reading request headers.
	serverReadHeaderTimeout = 10 * time.Second

	// serverStartupTimeout is the time to wait for server startup.
	serverStartupTimeout = 50 * time.Millisecond
)

// DefaultHistogramBuckets are the histogram buckets for request duration.
// Values in seconds: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s
//
//nolint:gochecknoglobals // Intentional: exported package-level variable per API contract
var DefaultHistogramBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0,
}

// ============================================================================
// Types
// ============================================================================

// PluginMetrics holds the Prometheus metrics collectors for a plugin.
// Use NewPluginMetrics to create an instance, or access via MetricsRegistry()
// if using the default interceptor.
type PluginMetrics struct {
	// RequestsTotal is the counter for total requests.
	// Labels: grpc_method, grpc_code, plugin_name
	RequestsTotal *prometheus.CounterVec

	// RequestDuration is the histogram for request latency.
	// Labels: grpc_method, plugin_name
	RequestDuration *prometheus.HistogramVec

	// RecommendationsTotal is the counter for total recommendations returned.
	// Labels: plugin_name, category, action_type
	RecommendationsTotal *prometheus.CounterVec

	// RecommendationsPerResponse is the histogram for recommendations per response.
	// Labels: plugin_name
	RecommendationsPerResponse *prometheus.HistogramVec

	// ResolveResourceTypesResolved is the counter for source_types that were
	// successfully resolved to a mapping by ResolveResourceTypes.
	// Labels: plugin_name
	ResolveResourceTypesResolved *prometheus.CounterVec

	// ResolveResourceTypesUnresolved is the counter for source_types that were
	// requested but NOT resolved (omitted from the response) by ResolveResourceTypes.
	// Labels: plugin_name
	ResolveResourceTypesUnresolved *prometheus.CounterVec

	// Registry is the Prometheus registry containing these metrics.
	Registry *prometheus.Registry

	// pluginName is stored for internal use by the interceptor
	pluginName string
}

// MetricsServerConfig configures the optional metrics HTTP server.
type MetricsServerConfig struct {
	// Port is the HTTP port to listen on. Default: 9090
	Port int

	// Path is the URL path for the metrics endpoint. Default: "/metrics"
	Path string

	// Registry is the Prometheus registry to expose. If nil, a new registry
	// with default Go collectors is used.
	Registry *prometheus.Registry
}

// ============================================================================
// Core Interceptor API
// ============================================================================

// NewPluginMetrics creates a new PluginMetrics instance with metrics registered
// to a new prometheus.Registry.
//
// Parameters:
//   - pluginName: Identifier for the plugin (used in metric help text)
//
// The returned PluginMetrics can be used to:
//   - Access the Registry for custom HTTP handler setup
//   - Create a MetricsUnaryServerInterceptor with MetricsInterceptorWithRegistry
func NewPluginMetrics(pluginName string) *PluginMetrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(collectors.NewGoCollector())

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "requests_total",
			Help:      "Total gRPC requests",
		},
		[]string{"grpc_method", "grpc_code", FieldPluginName},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "request_duration_seconds",
			Help:      "Request duration histogram",
			Buckets:   DefaultHistogramBuckets,
		},
		[]string{"grpc_method", FieldPluginName},
	)

	recommendationsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "recommendations_returned_total",
			Help:      "Total recommendations returned by GetRecommendations",
		},
		[]string{FieldPluginName, "category", "action_type"},
	)

	recommendationsPerResponse := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "recommendations_per_response",
			Help:      "Number of recommendations per GetRecommendations response",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{FieldPluginName},
	)

	resolveResourceTypesResolved := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "resolve_resource_types_resolved_total",
			Help:      "Total source_types successfully resolved by ResolveResourceTypes",
		},
		[]string{FieldPluginName},
	)

	resolveResourceTypesUnresolved := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: MetricNamespace,
			Subsystem: MetricSubsystem,
			Name:      "resolve_resource_types_unresolved_total",
			Help:      "Total source_types requested but not resolved by ResolveResourceTypes",
		},
		[]string{FieldPluginName},
	)

	reg.MustRegister(requestsTotal)
	reg.MustRegister(requestDuration)
	reg.MustRegister(recommendationsTotal)
	reg.MustRegister(recommendationsPerResponse)
	reg.MustRegister(resolveResourceTypesResolved)
	reg.MustRegister(resolveResourceTypesUnresolved)

	return &PluginMetrics{
		RequestsTotal:                  requestsTotal,
		RequestDuration:                requestDuration,
		RecommendationsTotal:           recommendationsTotal,
		RecommendationsPerResponse:     recommendationsPerResponse,
		ResolveResourceTypesResolved:   resolveResourceTypesResolved,
		ResolveResourceTypesUnresolved: resolveResourceTypesUnresolved,
		Registry:                       reg,
		pluginName:                     pluginName,
	}
}

// MetricsUnaryServerInterceptor returns a gRPC server interceptor that records
// Prometheus metrics for each unary RPC call.
//
// The interceptor records:
//   - finfocus_plugin_requests_total: Counter with labels grpc_method, grpc_code, plugin_name
//   - finfocus_plugin_request_duration_seconds: Histogram with labels grpc_method, plugin_name
//
// Parameters:
//   - pluginName: Identifier for the plugin, used as the plugin_name label value
//
// Returns a gRPC unary server interceptor suitable for use with grpc.ChainUnaryInterceptor
// or ServeConfig.UnaryInterceptors.
//
// Example:
//
//	config := pluginsdk.ServeConfig{
//	    UnaryInterceptors: []grpc.UnaryServerInterceptor{
//	        pluginsdk.MetricsUnaryServerInterceptor("my-plugin"),
//	    },
//	}
func MetricsUnaryServerInterceptor(pluginName string) grpc.UnaryServerInterceptor {
	metrics := NewPluginMetrics(pluginName)
	return MetricsInterceptorWithRegistry(metrics)
}

// MetricsInterceptorWithRegistry returns an interceptor that uses the provided
// PluginMetrics for recording. Use this when you need access to the underlying
// registry for custom metrics exposure.
//
// Example:
//
//	metrics := pluginsdk.NewPluginMetrics("my-plugin")
//	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)
//	// Use metrics.Registry with promhttp.HandlerFor()
func MetricsInterceptorWithRegistry(metrics *PluginMetrics) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Execute the actual handler
		resp, err := handler(ctx, req)

		// Record metrics after handler completes
		duration := time.Since(start)
		code := status.Code(err)

		// Use the full method (e.g., "finfocus.v1.CostSource/GetProjectedCost")
		// Trim leading slash to ensure uniqueness across services while keeping clean labels.
		method := info.FullMethod
		if len(method) > 0 && method[0] == '/' {
			method = method[1:]
		}

		metrics.RequestsTotal.WithLabelValues(method, code.String(), metrics.pluginName).Inc()
		metrics.RequestDuration.WithLabelValues(method, metrics.pluginName).Observe(duration.Seconds())

		// Record recommendation-specific metrics for GetRecommendations
		if method == "finfocus.v1.CostSource/GetRecommendations" && err == nil {
			if recResp, ok := resp.(*pbc.GetRecommendationsResponse); ok {
				recordRecommendationMetrics(metrics, recResp)
			}
		}

		// Record hit/miss metrics for ResolveResourceTypes
		if method == "finfocus.v1.CostSource/ResolveResourceTypes" && err == nil {
			if resolveReq, reqOK := req.(*pbc.ResolveResourceTypesRequest); reqOK {
				if resolveResp, respOK := resp.(*pbc.ResolveResourceTypesResponse); respOK {
					recordResolveResourceTypesMetrics(metrics, resolveReq, resolveResp)
				}
			}
		}

		return resp, err
	}
}

// recordRecommendationMetrics records recommendation-specific Prometheus metrics.
func recordRecommendationMetrics(metrics *PluginMetrics, resp *pbc.GetRecommendationsResponse) {
	recs := resp.GetRecommendations()

	// Record total recommendations per response
	metrics.RecommendationsPerResponse.WithLabelValues(metrics.pluginName).
		Observe(float64(len(recs)))

	// Record each recommendation by category and action type
	for _, rec := range recs {
		if rec == nil {
			continue // avoid panics from malformed responses
		}
		category := rec.GetCategory().String()
		actionType := rec.GetActionType().String()
		metrics.RecommendationsTotal.WithLabelValues(
			metrics.pluginName,
			category,
			actionType,
		).Inc()
	}
}

// recordResolveResourceTypesMetrics records hit/miss metrics for ResolveResourceTypes:
// how many requested source_types were resolved (present in the response mappings)
// versus unresolved (omitted).
func recordResolveResourceTypesMetrics(
	metrics *PluginMetrics,
	req *pbc.ResolveResourceTypesRequest,
	resp *pbc.ResolveResourceTypesResponse,
) {
	resolved := len(resp.GetMappings())
	requested := len(req.GetSourceTypes())
	unresolved := requested - resolved
	if unresolved < 0 {
		unresolved = 0
	}

	if resolved > 0 {
		metrics.ResolveResourceTypesResolved.WithLabelValues(metrics.pluginName).Add(float64(resolved))
	}
	if unresolved > 0 {
		metrics.ResolveResourceTypesUnresolved.WithLabelValues(metrics.pluginName).Add(float64(unresolved))
	}
}

// ============================================================================
// Optional HTTP Server Helper
// ============================================================================

// StartMetricsServer starts a lightweight HTTP server that exposes Prometheus
// metrics at the configured endpoint.
//
// This is provided as a convenience for plugin authors who don't have an
// existing HTTP server. For production deployments, consider integrating
// with your existing HTTP infrastructure instead.
//
// The returned *http.Server can be used to gracefully shutdown the server:
//
//	server, err := pluginsdk.StartMetricsServer(config)
//	if err != nil { ... }
//	defer server.Shutdown(context.Background())
//
// Parameters:
//   - config: Server configuration (port, path, registry)
//
// Returns:
//   - *http.Server: The running HTTP server (for shutdown control)
//   - error: If the server fails to start
func StartMetricsServer(config MetricsServerConfig) (*http.Server, error) {
	// Apply defaults
	port := config.Port
	if port == 0 {
		port = DefaultMetricsPort
	}

	metricsPath := config.Path
	if metricsPath == "" {
		metricsPath = DefaultMetricsPath
	}

	registry := config.Registry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	// Create HTTP mux and handler
	mux := http.NewServeMux()
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	mux.Handle(metricsPath, handler)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Give the server a moment to start and check for immediate errors
	select {
	case err := <-errCh:
		return nil, fmt.Errorf("failed to start metrics server: %w", err)
	case <-time.After(serverStartupTimeout):
		// Server started successfully
	}

	return server, nil
}
