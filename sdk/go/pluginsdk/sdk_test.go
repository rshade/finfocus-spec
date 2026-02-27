//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// mockPlugin implements Plugin interface for testing.
type mockPlugin struct {
	name string
}

func (m *mockPlugin) Name() string { return m.name }

func (m *mockPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (m *mockPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (m *mockPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (m *mockPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

// mockSupportsPlugin implements both Plugin and SupportsProvider interfaces.
type mockSupportsPlugin struct {
	mockPlugin

	supported bool
	reason    string
	err       error
}

func (m *mockSupportsPlugin) Supports(
	_ context.Context,
	_ *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &pbc.SupportsResponse{
		Supported: m.supported,
		Reason:    m.reason,
	}, nil
}

// contextCapturingMockPlugin implements Plugin and captures context in handler calls.
// This allows tests to verify that the handler receives the final modified context.
type contextCapturingMockPlugin struct {
	name        string
	captureFunc func(ctx context.Context)
}

func (m *contextCapturingMockPlugin) Name() string { return m.name }

func (m *contextCapturingMockPlugin) GetProjectedCost(
	ctx context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if m.captureFunc != nil {
		m.captureFunc(ctx)
	}
	return &pbc.GetProjectedCostResponse{}, nil
}

func (m *contextCapturingMockPlugin) GetActualCost(
	ctx context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if m.captureFunc != nil {
		m.captureFunc(ctx)
	}
	return &pbc.GetActualCostResponse{}, nil
}

func (m *contextCapturingMockPlugin) GetPricingSpec(
	ctx context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if m.captureFunc != nil {
		m.captureFunc(ctx)
	}
	return &pbc.GetPricingSpecResponse{}, nil
}

func (m *contextCapturingMockPlugin) EstimateCost(
	ctx context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	if m.captureFunc != nil {
		m.captureFunc(ctx)
	}
	return &pbc.EstimateCostResponse{}, nil
}

// mockRegistry implements RegistryLookup for testing.
type mockRegistry struct {
	plugins map[string]string // key: "provider:region", value: plugin name
}

func (m *mockRegistry) FindPlugin(provider, region string) string {
	key := provider + ":" + region
	return m.plugins[key]
}

// =============================================================================
// Interceptor Integration Test Harness
// =============================================================================

const bufConnSize = 1024 * 1024

// interceptorTestHarness provides an in-memory gRPC server for testing interceptors.
// It uses bufconn to avoid network dependencies and enables testing of interceptor chains.
type interceptorTestHarness struct {
	server   *grpc.Server
	listener *bufconn.Listener
	client   pbc.CostSourceServiceClient
	conn     *grpc.ClientConn
}

// newInterceptorTestHarness creates and starts an in-memory gRPC server with the
// tracing interceptor (built-in) followed by any user-provided interceptors.
// The harness.Stop() method must be called (typically via defer) to clean up resources.
func newInterceptorTestHarness(
	t *testing.T,
	plugin Plugin,
	userInterceptors ...grpc.UnaryServerInterceptor,
) *interceptorTestHarness {
	t.Helper()

	listener := bufconn.Listen(bufConnSize)

	// Build interceptor chain: tracing first, then user interceptors (matches Serve() behavior)
	interceptors := make([]grpc.UnaryServerInterceptor, 0, 1+len(userInterceptors))
	interceptors = append(interceptors, TracingUnaryServerInterceptor())
	interceptors = append(interceptors, userInterceptors...)

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
	pbc.RegisterCostSourceServiceServer(server, NewServer(plugin))

	// Start server in background
	go func() {
		_ = server.Serve(listener)
	}()

	// Create client connection
	//nolint:staticcheck // grpc.NewClient doesn't work with bufconn
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	return &interceptorTestHarness{
		server:   server,
		listener: listener,
		conn:     conn,
		client:   pbc.NewCostSourceServiceClient(conn),
	}
}

// Stop shuts down the gRPC server and closes the client connection.
func (h *interceptorTestHarness) Stop() {
	if h.conn != nil {
		_ = h.conn.Close()
	}
	if h.server != nil {
		h.server.Stop()
	}
}

// TestName tests the Name RPC.
func TestName(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	server := NewServer(plugin)

	resp, err := server.Name(context.Background(), &pbc.NameRequest{})

	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())
}

// TestGetProjectedCost tests the GetProjectedCost RPC.
func TestGetProjectedCost(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	server := NewServer(plugin)

	req := &pbc.GetProjectedCostRequest{}
	resp, err := server.GetProjectedCost(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestGetActualCost tests the GetActualCost RPC.
func TestGetActualCost(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	server := NewServer(plugin)

	req := &pbc.GetActualCostRequest{}
	resp, err := server.GetActualCost(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestGetPricingSpec tests the GetPricingSpec RPC.
func TestGetPricingSpec(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	server := NewServer(plugin)

	req := &pbc.GetPricingSpecRequest{}
	resp, err := server.GetPricingSpec(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// TestEstimateCost tests the EstimateCost RPC.
func TestEstimateCost(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	server := NewServer(plugin)

	req := &pbc.EstimateCostRequest{}
	resp, err := server.EstimateCost(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
}
func TestSupports_PluginImplementsAndReturnsSupported(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	resp, err := server.Supports(context.Background(), req)

	require.NoError(t, err)
	assert.True(t, resp.GetSupported())
	assert.Empty(t, resp.GetReason())
}

// TestSupports_PluginImplementsAndReturnsNotSupported tests T006: plugin returns not-supported with reason.
func TestSupports_PluginImplementsAndReturnsNotSupported(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  false,
		reason:     "Resource type not supported by this plugin",
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "gamelift",
			Region:       "us-east-1",
		},
	}

	resp, err := server.Supports(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.GetSupported())
	assert.Equal(t, "Resource type not supported by this plugin", resp.GetReason())
}

// TestSupports_InvalidProviderRegionReturnsInvalidArgument tests T007: invalid provider/region returns InvalidArgument.
func TestSupports_InvalidProviderRegionReturnsInvalidArgument(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "unknown-provider",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := server.Supports(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "no plugin registered")
}

// TestSupports_NoPluginRegisteredReturnsInvalidArgument tests T008: no plugin registered returns InvalidArgument.
func TestSupports_NoPluginRegisteredReturnsInvalidArgument(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	// Empty registry - no plugins registered
	registry := &mockRegistry{
		plugins: map[string]string{},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := server.Supports(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "no plugin registered")
}

// TestSupports_PluginErrorReturnsInternalStatus tests T009: plugin error returns Internal status.
func TestSupports_PluginErrorReturnsInternalStatus(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		err:        errors.New("database connection failed"),
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := server.Supports(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	// Error message should be generic, not expose internal details
	assert.Contains(t, st.Message(), "plugin failed to execute")
}

// TestSupports_PluginNotImplementsReturnsDefaultResponse tests T014: plugin without SupportsProvider returns default.
func TestSupports_PluginNotImplementsReturnsDefaultResponse(t *testing.T) {
	// Use mockPlugin which does NOT implement SupportsProvider
	plugin := &mockPlugin{name: "legacy-plugin"}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "legacy-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	resp, err := server.Supports(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.GetSupported())
	assert.NotEmpty(t, resp.GetReason())
}

// TestSupports_DefaultResponseIncludesReason tests T015: default response includes reason explaining not implemented.
func TestSupports_DefaultResponseIncludesReason(t *testing.T) {
	plugin := &mockPlugin{name: "legacy-plugin"}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "legacy-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	resp, err := server.Supports(context.Background(), req)

	require.NoError(t, err)
	assert.False(t, resp.GetSupported())
	assert.Contains(t, resp.GetReason(), "not implemented")
}

// TestSupports_NilResourceReturnsInvalidArgument tests that a nil resource descriptor returns InvalidArgument.
func TestSupports_NilResourceReturnsInvalidArgument(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	// Request with nil resource
	req := &pbc.SupportsRequest{Resource: nil}

	_, err := server.Supports(context.Background(), req)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "resource descriptor is required")
}

// TestSupports_NewServerUsesDefaultRegistry tests that NewServer uses DefaultRegistryLookup.
func TestSupports_NewServerUsesDefaultRegistry(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	// Use NewServer which uses DefaultRegistryLookup (always returns empty)
	server := NewServer(plugin)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := server.Supports(context.Background(), req)

	// DefaultRegistryLookup returns "" for all lookups, so this should fail with InvalidArgument
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "no plugin registered")
}

// TestNewServerWithRegistry_NilRegistryUsesDefault tests that nil registry falls back to default.
func TestNewServerWithRegistry_NilRegistryUsesDefault(t *testing.T) {
	plugin := &mockSupportsPlugin{
		mockPlugin: mockPlugin{name: "test-plugin"},
		supported:  true,
	}
	// Pass nil registry - should fall back to DefaultRegistryLookup
	server := NewServerWithRegistry(plugin, nil)

	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	_, err := server.Supports(context.Background(), req)

	// Should behave same as NewServer - DefaultRegistryLookup returns ""
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "no plugin registered")
}

// TestNewServerWithOptions_WithCustomLogger tests that custom logger is used.
func TestNewServerWithOptions_WithCustomLogger(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "test-plugin"},
	}
	logger := newDefaultLogger()

	server := NewServerWithOptions(plugin, registry, &logger, nil)

	// Verify the server was created successfully
	resp, err := server.Name(context.Background(), &pbc.NameRequest{})
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())
}

// TestNewServerWithOptions_NilRegistryAndLogger tests nil parameters use defaults.
func TestNewServerWithOptions_NilRegistryAndLogger(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}

	// All nil except plugin - should use defaults
	server := NewServerWithOptions(plugin, nil, nil, nil)

	resp, err := server.Name(context.Background(), &pbc.NameRequest{})
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())
}

// TestCreateTestResource tests the CreateTestResource helper.
func TestCreateTestResource(t *testing.T) {
	// Test with properties
	props := map[string]string{"instance_type": "t2.micro", "region": "us-east-1"}
	resource := CreateTestResource("aws", "aws:ec2:Instance", props)

	assert.Equal(t, "aws", resource.GetProvider())
	assert.Equal(t, "aws:ec2:Instance", resource.GetResourceType())
	assert.Equal(t, "t2.micro", resource.GetTags()["instance_type"])
	assert.Equal(t, "us-east-1", resource.GetTags()["region"])

	// Test with nil properties - should create empty map
	resourceNilProps := CreateTestResource("azure", "azure:compute:VirtualMachine", nil)

	assert.Equal(t, "azure", resourceNilProps.GetProvider())
	assert.Equal(t, "azure:compute:VirtualMachine", resourceNilProps.GetResourceType())
	assert.NotNil(t, resourceNilProps.GetTags())
	assert.Empty(t, resourceNilProps.GetTags())
}

// =============================================================================
// UnaryInterceptors Tests (User Stories 1, 2, 3)
// =============================================================================

// trackingInterceptor creates an interceptor that tracks execution order.
// For concurrent testing scenarios, pass a non-nil mutex to protect slice access.
func trackingInterceptor(id string, order *[]string, mu *sync.Mutex) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if mu != nil {
			mu.Lock()
			*order = append(*order, id)
			mu.Unlock()
		} else {
			*order = append(*order, id)
		}
		return handler(ctx, req)
	}
}

// contextModifyingInterceptor creates an interceptor that adds a value to context.
type testContextKey string

func contextModifyingInterceptor(key testContextKey, value string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx = context.WithValue(ctx, key, value)
		return handler(ctx, req)
	}
}

// countingInterceptor creates an interceptor that counts invocations.
func countingInterceptor(counter *atomic.Int32) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		counter.Add(1)
		return handler(ctx, req)
	}
}

// --- User Story 1 Tests: Single Interceptor Registration ---

// TestIntegration_SingleInterceptorInvocation tests T003: single interceptor is invoked via actual gRPC call.
// This test starts an in-memory gRPC server with bufconn, registers a counting interceptor,
// makes an actual RPC call, and verifies the interceptor was invoked.
func TestIntegration_SingleInterceptorInvocation(t *testing.T) {
	var counter atomic.Int32
	interceptor := countingInterceptor(&counter)

	// Create server with our interceptor chain
	harness := newInterceptorTestHarness(t, &mockPlugin{name: "test-plugin"}, interceptor)
	defer harness.Stop()

	// Make an actual RPC call to trigger the interceptor
	resp, err := harness.client.Name(context.Background(), &pbc.NameRequest{})

	// Verify the call succeeded
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())

	// Verify the interceptor was invoked exactly once
	assert.Equal(t, int32(1), counter.Load(), "interceptor should have been invoked once")

	// Make another call and verify counter increments
	_, err = harness.client.Name(context.Background(), &pbc.NameRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(2), counter.Load(), "interceptor should have been invoked twice")
}

// TestIntegration_TraceIDPropagation tests T004: trace ID propagates through interceptor.
// This test starts an in-memory gRPC server, makes an actual RPC call, and verifies
// that the tracing interceptor generates a valid trace ID that is accessible in subsequent interceptors.
func TestIntegration_TraceIDPropagation(t *testing.T) {
	var capturedTraceID string
	var mu sync.Mutex
	interceptor := func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Capture trace ID from context (set by TracingUnaryServerInterceptor which runs before us)
		mu.Lock()
		capturedTraceID = TraceIDFromContext(ctx)
		mu.Unlock()
		return handler(ctx, req)
	}

	// Create server with tracing interceptor (built-in) + our capturing interceptor
	harness := newInterceptorTestHarness(t, &mockPlugin{name: "test-plugin"}, interceptor)
	defer harness.Stop()

	// Make an actual RPC call to trigger the interceptors
	resp, err := harness.client.Name(context.Background(), &pbc.NameRequest{})

	// Verify the call succeeded
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())

	// Verify trace ID was captured and is valid (32 hex chars)
	mu.Lock()
	traceID := capturedTraceID
	mu.Unlock()

	assert.NotEmpty(t, traceID, "trace ID should have been captured from context")
	assert.Len(t, traceID, 32, "trace ID should be 32 hex characters")

	// Verify it's valid hex
	for _, c := range traceID {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		assert.True(t, isHex, "trace ID should only contain hex characters, got: %c", c)
	}
}

// --- User Story 2 Tests: Multiple Interceptors Chaining ---

// TestIntegration_MultipleInterceptorsOrder tests T008: multiple interceptors execute in order via actual gRPC call.
// This test verifies that interceptors execute in the order they are provided.
func TestIntegration_MultipleInterceptorsOrder(t *testing.T) {
	var order []string
	var mu sync.Mutex

	interceptor1 := trackingInterceptor("first", &order, &mu)
	interceptor2 := trackingInterceptor("second", &order, &mu)
	interceptor3 := trackingInterceptor("third", &order, &mu)

	// Create server with multiple user interceptors
	harness := newInterceptorTestHarness(t, &mockPlugin{name: "test-plugin"},
		interceptor1, interceptor2, interceptor3)
	defer harness.Stop()

	// Make an actual RPC call to trigger the interceptors
	resp, err := harness.client.Name(context.Background(), &pbc.NameRequest{})

	// Verify the call succeeded
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.GetName())

	// Verify interceptors executed in the correct order
	mu.Lock()
	executionOrder := make([]string, len(order))
	copy(executionOrder, order)
	mu.Unlock()

	assert.Equal(t, []string{"first", "second", "third"}, executionOrder,
		"interceptors should execute in the order they were registered")
}

// TestIntegration_ContextModificationsPropagation tests T009: context modifications propagate between interceptors.
// This test verifies that context values set by earlier interceptors are visible to later interceptors,
// and that the handler (plugin) receives the final modified context with all values.
func TestIntegration_ContextModificationsPropagation(t *testing.T) {
	key1 := testContextKey("key1")
	key2 := testContextKey("key2")

	var capturedVal1InInterceptor2, capturedVal2InInterceptor3 string
	var capturedVal1InHandler, capturedVal2InHandler string
	var mu sync.Mutex

	// First interceptor sets key1
	interceptor1 := contextModifyingInterceptor(key1, "value1")

	// Second interceptor sets key2 AND verifies it can see key1
	interceptor2 := func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Capture value from first interceptor
		mu.Lock()
		if val := ctx.Value(key1); val != nil {
			capturedVal1InInterceptor2 = val.(string)
		}
		mu.Unlock()
		// Add our own value
		ctx = context.WithValue(ctx, key2, "value2")
		return handler(ctx, req)
	}

	// Third interceptor verifies it can see both values
	interceptor3 := func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		mu.Lock()
		if val := ctx.Value(key2); val != nil {
			capturedVal2InInterceptor3 = val.(string)
		}
		mu.Unlock()
		return handler(ctx, req)
	}

	// Create a custom plugin that captures context values when handler is called
	contextCapturingPlugin := &contextCapturingMockPlugin{
		name: "context-test-plugin",
		captureFunc: func(ctx context.Context) {
			mu.Lock()
			defer mu.Unlock()
			if val := ctx.Value(key1); val != nil {
				capturedVal1InHandler = val.(string)
			}
			if val := ctx.Value(key2); val != nil {
				capturedVal2InHandler = val.(string)
			}
		},
	}

	// Create server with the interceptor chain
	harness := newInterceptorTestHarness(t, contextCapturingPlugin,
		interceptor1, interceptor2, interceptor3)
	defer harness.Stop()

	// Make an actual RPC call using GetProjectedCost (which passes context to plugin)
	// Note: Name() doesn't pass context to the plugin, so we use GetProjectedCost instead
	_, err := harness.client.GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{})

	// Verify the call succeeded
	require.NoError(t, err)

	// Verify context values were propagated correctly through interceptors
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "value1", capturedVal1InInterceptor2,
		"interceptor2 should see value1 from interceptor1")
	assert.Equal(t, "value2", capturedVal2InInterceptor3,
		"interceptor3 should see value2 from interceptor2")

	// Verify the handler received the final context with ALL values
	assert.Equal(t, "value1", capturedVal1InHandler,
		"handler should receive context with value1 from interceptor1")
	assert.Equal(t, "value2", capturedVal2InHandler,
		"handler should receive context with value2 from interceptor2")
}

// --- User Story 3 Tests: Backward Compatibility ---

// TestIntegration_NilUnaryInterceptorsField tests T012: nil field works (backward compat).
// This test verifies that a server with no user interceptors still works correctly,
// with only the built-in tracing interceptor running.
func TestIntegration_NilUnaryInterceptorsField(t *testing.T) {
	// Create server with no user interceptors (backward compatibility pattern)
	harness := newInterceptorTestHarness(t, &mockPlugin{name: "backward-compat-test"})
	defer harness.Stop()

	// Make an actual RPC call - should work with just the tracing interceptor
	resp, err := harness.client.Name(context.Background(), &pbc.NameRequest{})

	// Verify the call succeeded
	require.NoError(t, err)
	assert.Equal(t, "backward-compat-test", resp.GetName())
}

// TestIntegration_EmptySliceUnaryInterceptors tests T013: empty slice works.
// This is functionally identical to nil interceptors but tests the explicit empty slice case.
func TestIntegration_EmptySliceUnaryInterceptors(t *testing.T) {
	// newInterceptorTestHarness with no variadic args is equivalent to empty slice
	harness := newInterceptorTestHarness(t, &mockPlugin{name: "empty-slice-test"})
	defer harness.Stop()

	// Make an actual RPC call
	resp, err := harness.client.Name(context.Background(), &pbc.NameRequest{})

	// Verify the call succeeded
	require.NoError(t, err)
	assert.Equal(t, "empty-slice-test", resp.GetName())
}

// TestInterceptorChainBuilding tests the interceptor chain building logic from Serve().
// This is a unit test of the chain building pattern used in Serve().
func TestInterceptorChainBuilding(t *testing.T) {
	tests := []struct {
		name             string
		userInterceptors []grpc.UnaryServerInterceptor
		expectedChainLen int
		description      string
	}{
		{
			name:             "nil user interceptors",
			userInterceptors: nil,
			expectedChainLen: 1, // just tracing
			description:      "tracing only",
		},
		{
			name:             "empty user interceptors",
			userInterceptors: []grpc.UnaryServerInterceptor{},
			expectedChainLen: 1, // just tracing
			description:      "tracing only",
		},
		{
			name: "one user interceptor",
			userInterceptors: []grpc.UnaryServerInterceptor{
				func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				},
			},
			expectedChainLen: 2, // tracing + user
			description:      "tracing + 1 user",
		},
		{
			name: "three user interceptors",
			userInterceptors: []grpc.UnaryServerInterceptor{
				func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				},
				func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				},
				func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				},
			},
			expectedChainLen: 4, // tracing + 3 user
			description:      "tracing + 3 user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the chain building logic from Serve()
			interceptors := make([]grpc.UnaryServerInterceptor, 0, 1+len(tt.userInterceptors))
			interceptors = append(interceptors, TracingUnaryServerInterceptor())
			interceptors = append(interceptors, tt.userInterceptors...)

			assert.Len(t, interceptors, tt.expectedChainLen, tt.description)
			// First interceptor should always be tracing
			assert.NotNil(t, interceptors[0], "tracing interceptor should be first")
		})
	}
}

// =============================================================================
// Port Resolution Tests (--port flag, FINFOCUS_PLUGIN_PORT env var)
// =============================================================================

// TestResolvePort_RequestedTakesPrecedence tests that config.Port takes precedence over env var.
func TestResolvePort_RequestedTakesPrecedence(t *testing.T) {
	// Set env var to verify it gets ignored when requested port is specified
	t.Setenv(EnvPort, "9000")

	got := resolvePort(8080)

	assert.Equal(t, 8080, got, "requested port should take precedence over env var")
}

// TestResolvePort_FallsBackToEnvVar tests that FINFOCUS_PLUGIN_PORT is used when no port requested.
func TestResolvePort_FallsBackToEnvVar(t *testing.T) {
	t.Setenv(EnvPort, "7777")

	got := resolvePort(0)

	assert.Equal(t, 7777, got, "should fall back to FINFOCUS_PLUGIN_PORT when requested is 0")
}

// TestResolvePort_FallsBackToLegacyEnvVar tests that PULUMICOST_PLUGIN_PORT is used when FINFOCUS_PLUGIN_PORT is not set.
func TestResolvePort_FallsBackToLegacyEnvVar(t *testing.T) {
	require.NoError(t, os.Unsetenv(EnvPort))
	t.Setenv(EnvPortFallback, "8888")

	got := resolvePort(0)

	assert.Equal(t, 8888, got, "should fall back to PULUMICOST_PLUGIN_PORT when FINFOCUS_PLUGIN_PORT is not set")
}

// TestResolvePort_ReturnsZeroWhenNeitherSet tests ephemeral port behavior.
func TestResolvePort_ReturnsZeroWhenNeitherSet(t *testing.T) {
	// Ensure env vars are not set
	t.Setenv(EnvPort, "")
	require.NoError(t, os.Unsetenv(EnvPort))
	t.Setenv(EnvPortFallback, "")
	require.NoError(t, os.Unsetenv(EnvPortFallback))

	got := resolvePort(0)

	assert.Equal(t, 0, got, "should return 0 (ephemeral) when neither port is specified")
}

// TestResolvePort_IgnoresGenericPORT tests that PORT env var is NOT read.
// This is a critical security test - the generic PORT env var causes multi-plugin conflicts.
func TestResolvePort_IgnoresGenericPORT(t *testing.T) {
	// Set the DANGEROUS generic PORT env var
	t.Setenv("PORT", "5000")
	// Ensure our canonical env vars are NOT set
	t.Setenv(EnvPort, "")
	require.NoError(t, os.Unsetenv(EnvPort))
	t.Setenv(EnvPortFallback, "")
	require.NoError(t, os.Unsetenv(EnvPortFallback))

	got := resolvePort(0)

	// PORT should be IGNORED - we should get 0 (ephemeral), not 5000
	msg := "PORT env var should be ignored; only FINFOCUS_PLUGIN_PORT and legacy PULUMICOST_PLUGIN_PORT are read"
	assert.Equal(t, 0, got, msg)
}

// TestServe_InvalidPluginInfoReturnsError tests T020: Serve returns error for invalid PluginInfo.
func TestServe_InvalidPluginInfoReturnsError(t *testing.T) {
	plugin := &mockPlugin{name: "test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create listener but we expect it to be closed or not used for serving
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	// Invalid PluginInfo (missing version)
	config := ServeConfig{
		Plugin:   plugin,
		Listener: ln,
		PluginInfo: &PluginInfo{
			Name: "test-plugin",
			// Version: "", // Missing
			SpecVersion: SpecVersion,
		},
	}

	err = Serve(ctx, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PluginInfo")
	assert.Contains(t, err.Error(), "version is required")
}

// TestParsePortFlag tests the ParsePortFlag function.
// Note: Flag values cannot be easily tested in unit tests because flag.Parse()
// affects global state. This test verifies the function doesn't panic.
func TestParsePortFlag(t *testing.T) {
	// Simply verify the function doesn't panic and returns a value
	got := ParsePortFlag()

	// Default value when flag is not explicitly set is 0
	assert.GreaterOrEqual(t, got, 0, "ParsePortFlag should return non-negative value")
}

func TestGetPluginInfo(t *testing.T) {
	t.Run("provider implements GetPluginInfoProvider returns metadata", func(t *testing.T) {
		plugin := &mockPluginInfoPlugin{
			name:      "test-plugin",
			version:   "v1.0.0",
			providers: []string{"aws", "azure"},
		}
		server := NewServer(plugin)

		req := &pbc.GetPluginInfoRequest{}
		resp, err := server.GetPluginInfo(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test-plugin", resp.GetName())
		assert.Equal(t, "v1.0.0", resp.GetVersion())
		assert.Equal(t, []string{"aws", "azure"}, resp.GetProviders())
		assert.Equal(t, SpecVersion, resp.GetSpecVersion())
	})

	t.Run("provider does not implement GetPluginInfoProvider returns Unimplemented", func(t *testing.T) {
		plugin := &mockPlugin{name: "legacy-plugin"}
		server := NewServer(plugin)

		req := &pbc.GetPluginInfoRequest{}
		resp, err := server.GetPluginInfo(context.Background(), req)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, codes.Unimplemented, status.Code(err))
	})

	t.Run("plugin returns nil response - friendly error", func(t *testing.T) {
		plugin := &errorPluginInfoPlugin{shouldReturnNil: true}
		server := NewServer(plugin)
		_, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to retrieve plugin metadata")
	})

	t.Run("plugin returns incomplete metadata - friendly error", func(t *testing.T) {
		plugin := &errorPluginInfoPlugin{incomplete: true}
		server := NewServer(plugin)
		_, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin metadata is incomplete")
	})

	t.Run("plugin returns invalid spec version - friendly error", func(t *testing.T) {
		plugin := &errorPluginInfoPlugin{invalidSpec: true}
		server := NewServer(plugin)
		_, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plugin reported an invalid specification version")
	})
}

type errorPluginInfoPlugin struct {
	mockPlugin

	shouldReturnNil bool
	incomplete      bool
	invalidSpec     bool
}

func (m *errorPluginInfoPlugin) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	if m.shouldReturnNil {
		//nolint:nilnil // Intentional nil return for testing
		return nil, nil
	}
	if m.incomplete {
		return &pbc.GetPluginInfoResponse{Name: "test"}, nil
	}
	if m.invalidSpec {
		return &pbc.GetPluginInfoResponse{
			Name:        "test",
			Version:     "v1.0.0",
			SpecVersion: "invalid",
		}, nil
	}
	return nil, errors.New("unexpected error")
}

type mockPluginInfoPlugin struct {
	mockPlugin

	name      string
	version   string
	providers []string
	metadata  map[string]string
}

// backwardCompatTestPlugin implements Plugin and optional interfaces for testing capability discovery.
type backwardCompatTestPlugin struct {
	name string
}

func (p *backwardCompatTestPlugin) Name() string { return p.name }

// Implements Plugin interface.
func (p *backwardCompatTestPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (p *backwardCompatTestPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (p *backwardCompatTestPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *backwardCompatTestPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{}, nil
}

// Implements RecommendationsProvider.
func (p *backwardCompatTestPlugin) GetRecommendations(
	_ context.Context,
	_ *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return &pbc.GetRecommendationsResponse{}, nil
}

// Implements BudgetsProvider.
func (p *backwardCompatTestPlugin) GetBudgets(
	_ context.Context,
	_ *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	return &pbc.GetBudgetsResponse{}, nil
}

// Implements DismissProvider.
func (p *backwardCompatTestPlugin) DismissRecommendation(
	_ context.Context,
	_ *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	return &pbc.DismissRecommendationResponse{}, nil
}

// Implements DryRunHandler.
func (p *backwardCompatTestPlugin) HandleDryRun(
	_ context.Context,
	_ *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	return &pbc.DryRunResponse{}, nil
}

func (m *mockPluginInfoPlugin) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        m.name,
		Version:     m.version,
		SpecVersion: SpecVersion,
		Providers:   m.providers,
		Metadata:    m.metadata,
	}, nil
}

func TestValidateCORSConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  WebConfig
		wantErr bool
	}{
		{
			name:    "nil config (disabled)",
			config:  WebConfig{Enabled: false},
			wantErr: false,
		},
		{
			name:    "valid config",
			config:  WebConfig{Enabled: true, AllowedOrigins: []string{"http://localhost:3000"}},
			wantErr: false,
		},
		{
			name:    "wildcard origin",
			config:  WebConfig{Enabled: true, AllowedOrigins: []string{"*"}},
			wantErr: false,
		},
		{
			name:    "wildcard mixed with specific",
			config:  WebConfig{Enabled: true, AllowedOrigins: []string{"*", "http://localhost:3000"}},
			wantErr: true,
		},
		{
			name:    "credentials with wildcard",
			config:  WebConfig{Enabled: true, AllowedOrigins: []string{"*"}, AllowCredentials: true},
			wantErr: true,
		},
		{
			name: "credentials with specific origin",
			config: WebConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"http://localhost:3000"},
				AllowCredentials: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCORSConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetPluginInfo_BackwardCompatibility_CapabilitiesEnumAndStringMap tests that
// GetPluginInfo returns both the new enum-based capabilities AND the legacy string map
// for backward compatibility with older clients.
func TestGetPluginInfo_BackwardCompatibility_CapabilitiesEnumAndStringMap(t *testing.T) {
	// Create a plugin that implements multiple optional interfaces
	plugin := &backwardCompatTestPlugin{
		name: "backward-compat-test-plugin",
	}

	// Create server with basic plugin info to enable GetPluginInfo
	pluginInfo := NewPluginInfo("backward-compat-test-plugin", "v1.0.0")
	server := NewServerWithOptions(plugin, nil, nil, pluginInfo)

	ctx := context.Background()
	req := &pbc.GetPluginInfoRequest{}

	resp, err := server.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify enum-based capabilities are present
	require.NotEmpty(t, resp.GetCapabilities(), "capabilities enum should not be empty")
	// backwardCompatTestPlugin now implements all 4 optional interfaces + base interface capabilities
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
	}
	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilities(),
		"capabilities enum should include all supported capabilities")

	// Verify legacy string map is present in metadata for backward compatibility
	require.NotNil(t, resp.GetMetadata(), "metadata should not be nil")
	assert.Equal(t, "true", resp.GetMetadata()["supports_projected_costs"],
		"legacy projected_costs should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_actual_costs"],
		"legacy actual_costs should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_pricing_spec"],
		"legacy pricing_spec should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_estimate_cost"],
		"legacy estimate_cost should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_recommendations"],
		"legacy recommendations should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_dismiss_recommendations"],
		"legacy dismiss_recommendations should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_dry_run"],
		"legacy dry_run should be 'true' in metadata")
	assert.Equal(t, "true", resp.GetMetadata()["supports_budgets"],
		"legacy budgets should be 'true' in metadata")

	// Verify all expected keys are present in legacy metadata
	expectedKeys := []string{
		"supports_projected_costs",
		"supports_actual_costs",
		"supports_pricing_spec",
		"supports_estimate_cost",
		"supports_recommendations",
		"supports_dismiss_recommendations",
		"supports_dry_run",
		"supports_budgets",
	}
	for _, key := range expectedKeys {
		assert.Contains(t, resp.GetMetadata(), key, "metadata should contain expected legacy capability key: %s", key)
	}

	// Test with explicitly set capabilities (override auto-discovery)
	explicitCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
	}
	explicitPluginInfo := NewPluginInfo("explicit-test-plugin", "v1.0.0",
		WithCapabilities(explicitCapabilities...))
	explicitServer := NewServerWithOptions(
		&backwardCompatTestPlugin{name: "explicit-test-plugin"},
		nil,
		nil,
		explicitPluginInfo,
	)

	explicitResp, err := explicitServer.GetPluginInfo(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, explicitResp)

	// Verify only explicitly set capabilities are reported
	assert.ElementsMatch(t, explicitCapabilities, explicitResp.GetCapabilities(),
		"explicitly set capabilities should be returned")

	// Verify legacy metadata reflects only the explicitly set capabilities
	assert.Equal(t, "true", explicitResp.GetMetadata()["supports_projected_costs"], "projected_costs should be 'true'")
	assert.Equal(t, "true", explicitResp.GetMetadata()["supports_actual_costs"], "actual_costs should be 'true'")
	assert.NotContains(t, explicitResp.GetMetadata(), "supports_recommendations",
		"recommendations should not be present (not explicitly set)")
	assert.NotContains(t, explicitResp.GetMetadata(), "supports_budgets",
		"budgets should not be present (not explicitly set)")
}

// TestSupports_AutoDiscovery tests that Supports RPC automatically populates
// both the new capabilities_enum AND the legacy capabilities map.
func TestSupports_AutoDiscovery(t *testing.T) {
	// Create a plugin that implements multiple optional interfaces
	plugin := &backwardCompatTestPlugin{
		name: "auto-discovery-test-plugin",
	}
	registry := &mockRegistry{
		plugins: map[string]string{"aws:us-east-1": "auto-discovery-test-plugin"},
	}
	server := NewServerWithRegistry(plugin, registry)

	ctx := context.Background()
	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Region:       "us-east-1",
		},
	}

	resp, err := server.Supports(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify enum-based capabilities are present in SupportsResponse
	require.NotEmpty(t, resp.GetCapabilitiesEnum(), "capabilities_enum should not be empty")
	// Plugin implements all 4 optional interfaces + base interface capabilities
	expectedCapabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DISMISS_RECOMMENDATIONS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS,
	}
	assert.ElementsMatch(t, expectedCapabilities, resp.GetCapabilitiesEnum(),
		"capabilities_enum should include all supported capabilities")

	// Verify legacy string map is present in SupportsResponse for backward compatibility
	require.NotNil(t, resp.GetCapabilities(), "capabilities map should not be nil")
	assert.True(t, resp.GetCapabilities()["supports_projected_costs"],
		"legacy projected_costs should be true")
	assert.True(t, resp.GetCapabilities()["supports_actual_costs"],
		"legacy actual_costs should be true")
	assert.True(t, resp.GetCapabilities()["supports_pricing_spec"],
		"legacy pricing_spec should be true")
	assert.True(t, resp.GetCapabilities()["supports_estimate_cost"],
		"legacy estimate_cost should be true")
	assert.True(t, resp.GetCapabilities()["supports_recommendations"],
		"legacy recommendations should be true")
	assert.True(
		t,
		resp.GetCapabilities()["supports_dismiss_recommendations"],
		"legacy dismiss_recommendations should be true",
	)
	assert.True(t, resp.GetCapabilities()["supports_dry_run"],
		"legacy dry_run should be true")
	assert.True(t, resp.GetCapabilities()["supports_budgets"],
		"legacy budgets should be true")
}
