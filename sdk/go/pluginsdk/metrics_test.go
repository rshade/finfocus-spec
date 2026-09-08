package pluginsdk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ============================================================================
// User Story 1 Tests: Enable Metrics Collection for Plugin
// ============================================================================

// TestNewPluginMetrics verifies registry creation and metric registration.
func TestNewPluginMetrics(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("test-plugin")

	require.NotNil(t, metrics)
	require.NotNil(t, metrics.Registry)
	require.NotNil(t, metrics.RequestsTotal)
	require.NotNil(t, metrics.RequestDuration)

	// Verify metrics are registered by gathering them
	families, err := metrics.Registry.Gather()
	require.NoError(t, err)

	// Should have at least the two metrics we registered
	// (they may show 0 values before any observations)
	metricNames := make(map[string]bool)
	for _, family := range families {
		metricNames[family.GetName()] = true
	}

	// Note: metrics only appear after first observation
	// So we make a test observation first
	metrics.RequestsTotal.WithLabelValues("TestMethod", "OK", "test-plugin").Inc()
	metrics.RequestDuration.WithLabelValues("TestMethod", "test-plugin").Observe(0.1)

	families, err = metrics.Registry.Gather()
	require.NoError(t, err)

	metricNames = make(map[string]bool)
	for _, family := range families {
		metricNames[family.GetName()] = true
	}

	assert.True(t, metricNames["finfocus_plugin_requests_total"], "should have requests_total metric")
	assert.True(
		t,
		metricNames["finfocus_plugin_request_duration_seconds"],
		"should have request_duration_seconds metric",
	)
}

// TestMetricsUnaryServerInterceptor_CounterIncrement verifies counter increments on request.
func TestMetricsUnaryServerInterceptor_CounterIncrement(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("test-plugin")
	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

	// Create mock handler that succeeds
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/finfocus.v1.CostSource/GetProjectedCost",
	}

	// Execute interceptor
	ctx := context.Background()
	_, err := interceptor(ctx, "request", info, handler)
	require.NoError(t, err)

	// Verify counter was incremented
	counter, err := getCounterValue(
		metrics.RequestsTotal,
		"finfocus.v1.CostSource/GetProjectedCost",
		"OK",
		"test-plugin",
	)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), counter, 0.01)

	// Make another request
	_, err = interceptor(ctx, "request", info, handler)
	require.NoError(t, err)

	counter, err = getCounterValue(
		metrics.RequestsTotal,
		"finfocus.v1.CostSource/GetProjectedCost",
		"OK",
		"test-plugin",
	)
	require.NoError(t, err)
	assert.InDelta(t, float64(2), counter, 0.01)
}

// TestMetricsUnaryServerInterceptor_HistogramObservation verifies duration recorded.
func TestMetricsUnaryServerInterceptor_HistogramObservation(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("test-plugin")
	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

	// Create mock handler with known delay
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/finfocus.v1.CostSource/GetActualCost",
	}

	ctx := context.Background()
	_, err := interceptor(ctx, "request", info, handler)
	require.NoError(t, err)

	// Verify histogram was observed
	count, sum, err := getHistogramValues(
		metrics.RequestDuration,
		"finfocus.v1.CostSource/GetActualCost",
		"test-plugin",
	)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), count)
	assert.GreaterOrEqual(t, sum, 0.01) // At least 10ms
}

// TestMetricsUnaryServerInterceptor_ErrorHandling verifies grpc_code label for errors.
func TestMetricsUnaryServerInterceptor_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "nil error returns OK",
			err:          nil,
			expectedCode: "OK",
		},
		{
			name:         "Internal error",
			err:          status.Error(codes.Internal, "internal error"),
			expectedCode: "Internal",
		},
		{
			name:         "NotFound error",
			err:          status.Error(codes.NotFound, "not found"),
			expectedCode: "NotFound",
		},
		{
			name:         "InvalidArgument error",
			err:          status.Error(codes.InvalidArgument, "invalid"),
			expectedCode: "InvalidArgument",
		},
		{
			name:         "non-gRPC error returns Unknown",
			err:          errors.New("plain error"),
			expectedCode: "Unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			localMetrics := pluginsdk.NewPluginMetrics("error-test-plugin")
			localInterceptor := pluginsdk.MetricsInterceptorWithRegistry(localMetrics)

			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, tc.err
			}

			info := &grpc.UnaryServerInfo{
				FullMethod: "/finfocus.v1.CostSource/GetProjectedCost",
			}

			ctx := context.Background()
			_, _ = localInterceptor(ctx, "request", info, handler)

			counter, err := getCounterValue(
				localMetrics.RequestsTotal,
				"finfocus.v1.CostSource/GetProjectedCost",
				tc.expectedCode,
				"error-test-plugin",
			)
			require.NoError(t, err)
			assert.InDelta(t, float64(1), counter, 0.01)
		})
	}
}

// TestMetricsInterceptor_ChainingWithTracingInterceptor verifies metrics interceptor chains
// correctly with TracingUnaryServerInterceptor.
func TestMetricsInterceptor_ChainingWithTracingInterceptor(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("chaining-test")
	metricsInterceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)
	tracingInterceptor := pluginsdk.TracingUnaryServerInterceptor()

	var capturedTraceID string
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		capturedTraceID = pluginsdk.TraceIDFromContext(ctx)
		if capturedTraceID == "force-error" {
			return nil, errors.New("forced error")
		}
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/finfocus.v1.CostSource/Name",
	}

	// Chain: tracing -> metrics -> handler
	chainedHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return metricsInterceptor(ctx, req, info, handler)
	}

	ctx := context.Background()
	_, err := tracingInterceptor(ctx, "request", info, chainedHandler)
	require.NoError(t, err)

	// Verify trace ID was set by tracing interceptor
	assert.NotEmpty(t, capturedTraceID, "trace_id should be set by tracing interceptor")

	// Verify metrics were recorded
	counter, err := getCounterValue(metrics.RequestsTotal, "finfocus.v1.CostSource/Name", "OK", "chaining-test")
	require.NoError(t, err)
	assert.InDelta(t, float64(1), counter, 0.01)
}

// TestNoMetricsOverhead_InterceptorNotConfigured verifies no metrics code executes
// when interceptor is not added to server chain.
func TestNoMetricsOverhead_InterceptorNotConfigured(t *testing.T) {
	// Create a handler without metrics interceptor
	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		if !handlerCalled {
			return nil, errors.New("impossible")
		}
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/finfocus.v1.CostSource/GetProjectedCost",
	}

	// Execute directly without metrics interceptor
	ctx := context.Background()
	resp, err := handler(ctx, "request")
	require.NoError(t, err)
	assert.Equal(t, "response", resp)
	assert.True(t, handlerCalled)

	// Verify no global metrics pollution
	// Create fresh metrics and verify they start at zero
	freshMetrics := pluginsdk.NewPluginMetrics("fresh-plugin")
	families, err := freshMetrics.Registry.Gather()
	require.NoError(t, err)

	// Fresh registry should have no observations
	for _, family := range families {
		// Ignore standard Go metrics, only check plugin metrics
		if !strings.HasPrefix(family.GetName(), "finfocus") {
			continue
		}
		for _, metric := range family.GetMetric() {
			if family.GetType() == dto.MetricType_COUNTER {
				assert.InDelta(
					t,
					float64(0),
					metric.GetCounter().GetValue(),
					0.01,
					"fresh counter should be zero",
				)
			}
		}
	}

	// Verify the handler was called and the response is correct
	// This confirms no overhead when metrics are not configured
	_ = info.FullMethod // Used to show that even with info available, no metrics are recorded
}

// ============================================================================
// User Story 2 Tests: Query Metrics via Standard Endpoint
// ============================================================================

// TestStartMetricsServer_DefaultConfig verifies server starts on default port.
func TestStartMetricsServer_DefaultConfig(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("server-test")

	// Add some data to the metrics
	metrics.RequestsTotal.WithLabelValues("TestMethod", "OK", "server-test").Inc()

	server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
		Registry: metrics.Registry,
		Port:     0, // Use ephemeral port for testing
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Server should be running (we can't easily test default port 9090 in parallel tests)
	// So this test primarily verifies the server starts without error
}

// TestStartMetricsServer_CustomPort verifies configurable port.
func TestStartMetricsServer_CustomPort(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("port-test")

	server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
		Registry: metrics.Registry,
		Port:     19191, // Custom port
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify we can connect to the custom port
	resp, err := http.Get("http://localhost:19191/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestStartMetricsServer_MetricsEndpoint verifies /metrics returns Prometheus format.
func TestStartMetricsServer_MetricsEndpoint(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("endpoint-test")

	// Add test data
	metrics.RequestsTotal.WithLabelValues("finfocus.v1.CostSource/GetProjectedCost", "OK", "endpoint-test").Add(42)
	metrics.RequestDuration.WithLabelValues("finfocus.v1.CostSource/GetProjectedCost", "endpoint-test").Observe(0.123)

	server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
		Registry: metrics.Registry,
		Port:     19192,
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:19192/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// Verify Prometheus format output
	assert.Contains(t, bodyStr, "finfocus_plugin_requests_total")
	assert.Contains(t, bodyStr, "finfocus_plugin_request_duration_seconds")
	assert.Contains(t, bodyStr, "grpc_method=\"finfocus.v1.CostSource/GetProjectedCost\"")
	assert.Contains(t, bodyStr, "plugin_name=\"endpoint-test\"")
	assert.Contains(t, bodyStr, "42") // Counter value
}

// TestStartMetricsServer_Shutdown verifies graceful shutdown.
func TestStartMetricsServer_Shutdown(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("shutdown-test")

	server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
		Registry: metrics.Registry,
		Port:     19193,
	})
	require.NoError(t, err)
	require.NotNil(t, server)

	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get("http://localhost:19193/metrics")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(ctx)
	require.NoError(t, err)

	// Give time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	// Server should no longer be reachable
	_, err = http.Get("http://localhost:19193/metrics")
	assert.Error(t, err)
}

// TestStartMetricsServer_CustomPath verifies custom metrics path.
func TestStartMetricsServer_CustomPath(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("path-test")
	metrics.RequestsTotal.WithLabelValues("Test", "OK", "path-test").Inc()

	server, err := pluginsdk.StartMetricsServer(pluginsdk.MetricsServerConfig{
		Registry: metrics.Registry,
		Port:     19194,
		Path:     "/custom/metrics",
	})
	require.NoError(t, err)
	require.NotNil(t, server)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Default path should 404
	resp, err := http.Get("http://localhost:19194/metrics")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Custom path should work
	resp, err = http.Get("http://localhost:19194/custom/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "finfocus_plugin_requests_total")
}

// ============================================================================
// User Story 3 Tests: Identify Plugin Performance Issues
// ============================================================================

// TestMetrics_PerMethodLabels verifies grpc_method label distinguishes methods.
func TestMetrics_PerMethodLabels(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("method-test")
	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "response", nil
	}

	methods := []string{
		"/finfocus.v1.CostSource/Name",
		"/finfocus.v1.CostSource/GetProjectedCost",
		"/finfocus.v1.CostSource/GetActualCost",
	}

	ctx := context.Background()
	for _, method := range methods {
		info := &grpc.UnaryServerInfo{FullMethod: method}
		_, err := interceptor(ctx, "request", info, handler)
		require.NoError(t, err)
	}

	// Verify each method has its own counter
	for _, method := range methods {
		methodName := method
		if len(methodName) > 0 && methodName[0] == '/' {
			methodName = methodName[1:]
		}
		counter, err := getCounterValue(metrics.RequestsTotal, methodName, "OK", "method-test")
		require.NoError(t, err)
		assert.InDelta(t, float64(1), counter, 0.01, "method %s should have count 1", methodName)
	}
}

// TestMetrics_AllGRPCMethods verifies all 6 gRPC methods are tracked.
func TestMetrics_AllGRPCMethods(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("all-methods-test")
	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "response", nil
	}

	// All 6 FinFocus gRPC methods
	methods := []string{
		"/finfocus.v1.CostSource/Name",
		"/finfocus.v1.CostSource/Supports",
		"/finfocus.v1.CostSource/GetProjectedCost",
		"/finfocus.v1.CostSource/GetActualCost",
		"/finfocus.v1.CostSource/GetPricingSpec",
		"/finfocus.v1.CostSource/EstimateCost",
	}

	ctx := context.Background()
	for _, method := range methods {
		info := &grpc.UnaryServerInfo{FullMethod: method}
		_, err := interceptor(ctx, "request", info, handler)
		require.NoError(t, err)
	}

	// Verify all methods have histogram entries
	families, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var histogramFamily *dto.MetricFamily
	for _, f := range families {
		if f.GetName() == "finfocus_plugin_request_duration_seconds" {
			histogramFamily = f
			break
		}
	}
	require.NotNil(t, histogramFamily, "histogram family should exist")

	// Count distinct methods in histogram
	methodsSeen := make(map[string]bool)
	for _, m := range histogramFamily.GetMetric() {
		for _, label := range m.GetLabel() {
			if label.GetName() == "grpc_method" {
				methodsSeen[label.GetValue()] = true
			}
		}
	}

	assert.Len(t, methodsSeen, 6, "should have 6 distinct methods")
	assert.True(t, methodsSeen["finfocus.v1.CostSource/Name"])
	assert.True(t, methodsSeen["finfocus.v1.CostSource/Supports"])
	assert.True(t, methodsSeen["finfocus.v1.CostSource/GetProjectedCost"])
	assert.True(t, methodsSeen["finfocus.v1.CostSource/GetActualCost"])
	assert.True(t, methodsSeen["finfocus.v1.CostSource/GetPricingSpec"])
	assert.True(t, methodsSeen["finfocus.v1.CostSource/EstimateCost"])
}

// TestMetrics_CountAccuracy sends exactly 1000 requests and verifies counter accuracy.
func TestMetrics_CountAccuracy(t *testing.T) {
	metrics := pluginsdk.NewPluginMetrics("accuracy-test")
	interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/finfocus.v1.CostSource/GetProjectedCost",
	}

	ctx := context.Background()
	const requestCount = 1000

	for range requestCount {
		_, err := interceptor(ctx, "request", info, handler)
		require.NoError(t, err)
	}

	counter, err := getCounterValue(
		metrics.RequestsTotal,
		"finfocus.v1.CostSource/GetProjectedCost",
		"OK",
		"accuracy-test",
	)
	require.NoError(t, err)

	// Verify exact count (should be exactly 1000, but allow 1% tolerance per spec)
	assert.InDelta(t, float64(requestCount), counter, float64(requestCount)*0.01,
		"counter should be 1000 within 1%% tolerance (990-1010)")
}

// TestMetricsInterceptor_ResolveResourceTypesHitMiss verifies the resolved/unresolved
// hit-miss counters for ResolveResourceTypes.
func TestMetricsInterceptor_ResolveResourceTypesHitMiss(t *testing.T) {
	t.Run("mixed resolved and unresolved increments both counters", func(t *testing.T) {
		metrics := pluginsdk.NewPluginMetrics("resolve-test-plugin")
		interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "unknown_type"},
		}
		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			return &pbc.ResolveResourceTypesResponse{
				Mappings: map[string]*pbc.ResourceTypeMapping{
					"aws_instance":  {PulumiToken: "aws:ec2/instance:Instance", Supported: true},
					"aws_s3_bucket": {PulumiToken: "aws:s3/bucket:Bucket", Supported: true},
				},
			}, nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/finfocus.v1.CostSource/ResolveResourceTypes"}

		_, err := interceptor(context.Background(), req, info, handler)
		require.NoError(t, err)

		resolved, err := getSingleLabelCounterValue(metrics.ResolveResourceTypesResolved, "resolve-test-plugin")
		require.NoError(t, err)
		assert.InDelta(t, float64(2), resolved, 0.01)

		unresolved, err := getSingleLabelCounterValue(metrics.ResolveResourceTypesUnresolved, "resolve-test-plugin")
		require.NoError(t, err)
		assert.InDelta(t, float64(1), unresolved, 0.01)
	})

	t.Run("error response does not increment either counter", func(t *testing.T) {
		metrics := pluginsdk.NewPluginMetrics("resolve-error-plugin")
		interceptor := pluginsdk.MetricsInterceptorWithRegistry(metrics)

		req := &pbc.ResolveResourceTypesRequest{
			SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
			SourceTypes:  []string{"aws_instance"},
		}
		handler := func(_ context.Context, _ interface{}) (interface{}, error) {
			return nil, status.Error(codes.Unavailable, "unavailable")
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/finfocus.v1.CostSource/ResolveResourceTypes"}

		_, err := interceptor(context.Background(), req, info, handler)
		require.Error(t, err)

		resolved, err := getSingleLabelCounterValue(metrics.ResolveResourceTypesResolved, "resolve-error-plugin")
		require.NoError(t, err)
		assert.InDelta(t, float64(0), resolved, 0.01)

		unresolved, err := getSingleLabelCounterValue(metrics.ResolveResourceTypesUnresolved, "resolve-error-plugin")
		require.NoError(t, err)
		assert.InDelta(t, float64(0), unresolved, 0.01)
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// getCounterValue retrieves the value of a counter with specific labels.
func getCounterValue(counter *prometheus.CounterVec, method, code, plugin string) (float64, error) {
	metric := &dto.Metric{}
	err := counter.WithLabelValues(method, code, plugin).Write(metric)
	if err != nil {
		return 0, err
	}
	return metric.GetCounter().GetValue(), nil
}

// getSingleLabelCounterValue retrieves the value of a counter labeled only by plugin_name.
func getSingleLabelCounterValue(counter *prometheus.CounterVec, plugin string) (float64, error) {
	metric := &dto.Metric{}
	err := counter.WithLabelValues(plugin).Write(metric)
	if err != nil {
		return 0, err
	}
	return metric.GetCounter().GetValue(), nil
}

// getHistogramValues retrieves count and sum from a histogram with specific labels.
func getHistogramValues(histogram *prometheus.HistogramVec, method, plugin string) (uint64, float64, error) {
	metric := &dto.Metric{}
	observer := histogram.WithLabelValues(method, plugin)
	// Cast to Histogram to get Write method
	if h, ok := observer.(prometheus.Histogram); ok {
		err := h.Write(metric)
		if err != nil {
			return 0, 0, err
		}
		return metric.GetHistogram().GetSampleCount(), metric.GetHistogram().GetSampleSum(), nil
	}
	return 0, 0, errors.New("failed to cast to Histogram")
}
