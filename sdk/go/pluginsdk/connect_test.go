package pluginsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

// connectTestPlugin implements the pluginsdk.Plugin interface for connect testing.
type connectTestPlugin struct {
	name string
}

func (m *connectTestPlugin) Name() string { return m.name }

func (m *connectTestPlugin) GetProjectedCost(
	_ context.Context,
	_ *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return &pbc.GetProjectedCostResponse{}, nil
}

func (m *connectTestPlugin) GetActualCost(
	_ context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return &pbc.GetActualCostResponse{}, nil
}

func (m *connectTestPlugin) GetPricingSpec(
	_ context.Context,
	_ *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return &pbc.GetPricingSpecResponse{}, nil
}

func (m *connectTestPlugin) EstimateCost(
	_ context.Context,
	_ *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return &pbc.EstimateCostResponse{
		CostMonthly: 100.0,
		Currency:    "USD",
	}, nil
}

func TestNewConnectHandler_NilServer_Panics(t *testing.T) {
	assert.PanicsWithValue(t, "NewConnectHandler: server cannot be nil", func() {
		pluginsdk.NewConnectHandler(nil)
	})
}

func TestConnectHandler_Name(t *testing.T) {
	plugin := &connectTestPlugin{name: "test-plugin"}
	server := pluginsdk.NewServer(plugin)
	handler := pluginsdk.NewConnectHandler(server)

	ctx := context.Background()
	req := connect.NewRequest(&pbc.NameRequest{})

	resp, err := handler.Name(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", resp.Msg.GetName())
}

func TestConnectHandler_EstimateCost(t *testing.T) {
	plugin := &connectTestPlugin{name: "test-plugin"}
	server := pluginsdk.NewServer(plugin)
	handler := pluginsdk.NewConnectHandler(server)

	ctx := context.Background()
	req := connect.NewRequest(&pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	})

	resp, err := handler.EstimateCost(ctx, req)
	require.NoError(t, err)
	assert.InDelta(t, 100.0, resp.Msg.GetCostMonthly(), 0.01)
	assert.Equal(t, "USD", resp.Msg.GetCurrency())
}

func TestConnectHandler_BatchCostFallback(t *testing.T) {
	plugin := &connectTestPlugin{name: "test-plugin"}
	server := pluginsdk.NewServer(plugin)
	handler := pluginsdk.NewConnectHandler(server)

	ctx := context.Background()
	req := connect.NewRequest(&pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			{
				Provider:     "aws",
				ResourceType: "ec2",
			},
		},
	})

	resp, err := handler.BatchCost(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetResults(), 1)
	assert.NotNil(t, resp.Msg.GetResults()[0].GetCostData().GetEstimate())
}

func TestServeConnect_WithHealthEndpoint(t *testing.T) {
	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:              true,
				EnableHealthEndpoint: true,
			},
		})
	}()

	// Wait for server to be ready
	addr := listener.Addr().String()
	waitForServer(t, addr)
	resp, err := http.Get("http://" + addr + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "ok", string(body))

	// Cleanup
	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServeConnect_ConnectProtocol(t *testing.T) {
	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "connect-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:              true,
				EnableHealthEndpoint: true,
			},
		})
	}()

	// Wait for server to be ready
	addr := listener.Addr().String()
	waitForServer(t, addr)
	reqBody := []byte(`{}`)
	resp, err := http.Post(
		"http://"+addr+"/finfocus.v1.CostSourceService/Name",
		"application/json",
		bytes.NewReader(reqBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, "connect-test-plugin", result["name"])

	// Cleanup
	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServeConnect_ConnectClient(t *testing.T) {
	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "client-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:              true,
				EnableHealthEndpoint: true,
			},
		})
	}()

	// Wait for server to be ready
	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Create Connect client
	client := pbcconnect.NewCostSourceServiceClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Test Name RPC
	nameResp, err := client.Name(ctx, connect.NewRequest(&pbc.NameRequest{}))
	require.NoError(t, err)
	assert.Equal(t, "client-test-plugin", nameResp.Msg.GetName())

	// Test EstimateCost RPC
	estimateResp, err := client.EstimateCost(ctx, connect.NewRequest(&pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	}))
	require.NoError(t, err)
	assert.InDelta(t, 100.0, estimateResp.Msg.GetCostMonthly(), 0.01)

	// Cleanup
	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestHealthHandler(t *testing.T) {
	handler := pluginsdk.HealthHandler(nil)

	t.Run("GET returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
		assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	})

	t.Run("HEAD returns 200 OK with no body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("POST returns 405 Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestWebConfig_Defaults(t *testing.T) {
	config := pluginsdk.DefaultWebConfig()

	assert.False(t, config.Enabled)
	assert.Nil(t, config.AllowedOrigins)
	assert.False(t, config.AllowCredentials)
	assert.False(t, config.EnableHealthEndpoint)
}

func TestWebConfig_WithMethods(t *testing.T) {
	config := pluginsdk.DefaultWebConfig().
		WithWebEnabled(true).
		WithAllowedOrigins([]string{"http://localhost:3000"}).
		WithAllowCredentials(true).
		WithHealthEndpoint(true)

	assert.True(t, config.Enabled)
	assert.Equal(t, []string{"http://localhost:3000"}, config.AllowedOrigins)
	assert.True(t, config.AllowCredentials)
	assert.True(t, config.EnableHealthEndpoint)
}

func TestServeConnect_CORS(t *testing.T) {
	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server with CORS enabled
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:              true,
				AllowedOrigins:       []string{"http://localhost:3000"},
				AllowCredentials:     true,
				EnableHealthEndpoint: true,
			},
		})
	}()

	// Wait for server to be ready
	addr := listener.Addr().String()
	waitForServer(t, addr)

	t.Run("preflight request returns CORS headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")

		resp, respErr := http.DefaultClient.Do(req)
		require.NoError(t, respErr)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "86400", resp.Header.Get("Access-Control-Max-Age")) // Default 24 hours
	})

	t.Run("actual request includes CORS headers", func(t *testing.T) {
		req, _ := http.NewRequest(
			http.MethodPost,
			"http://"+addr+"/finfocus.v1.CostSourceService/Name",
			bytes.NewReader([]byte(`{}`)),
		)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Content-Type", "application/json")

		resp, respErr := http.DefaultClient.Do(req)
		require.NoError(t, respErr)
		defer resp.Body.Close()

		assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	})

	t.Run("disallowed origin gets no CORS headers", func(t *testing.T) {
		req, _ := http.NewRequest(
			http.MethodPost,
			"http://"+addr+"/finfocus.v1.CostSourceService/Name",
			bytes.NewReader([]byte(`{}`)),
		)
		req.Header.Set("Origin", "http://evil.com")
		req.Header.Set("Content-Type", "application/json")

		resp, respErr := http.DefaultClient.Do(req)
		require.NoError(t, respErr)
		defer resp.Body.Close()

		assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	})

	// Cleanup
	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServeGRPC_LegacyMode(t *testing.T) {
	// Create a listener on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "legacy-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server with Web.Enabled = false (default)
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			// Web.Enabled defaults to false
		})
	}()

	// Wait for server to be ready using TCP dial (can't use HTTP health check for gRPC)
	addr := listener.Addr().String()
	for range 50 { // 500ms total
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr == nil {
			conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// HTTP/1.1 requests to a pure gRPC server will fail because gRPC uses HTTP/2.
	// The server returns an HTTP/2 response that can't be parsed by HTTP/1.1 client.
	_, err = http.Get("http://" + addr + "/healthz")
	// Expect an error because gRPC server speaks HTTP/2, not HTTP/1.1
	require.Error(t, err, "HTTP/1.1 request to gRPC server should fail")

	// Cleanup
	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestServe_CORSWildcardCredentialsValidation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	plugin := &connectTestPlugin{name: "cors-validation-test"}
	ctx := context.Background()

	// Attempt to start server with AllowCredentials + wildcard origin
	// This should fail immediately with a validation error
	err = pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
		Plugin:   plugin,
		Listener: listener,
		Web: pluginsdk.WebConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "AllowCredentials cannot be used with wildcard origin")
}

// T003: Test that nil AllowedHeaders uses default headers.
func TestServe_CORS_NilAllowedHeaders_UsesDefaults(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-nil-allowed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				// AllowedHeaders is nil - should use DefaultAllowedHeaders
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send preflight request
	req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify defaults are used
	allowedHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.Equal(t, pluginsdk.DefaultAllowedHeaders, allowedHeaders)

	cancel()
	<-errCh
}

// T004: Test that custom AllowedHeaders are used.
func TestServe_CORS_CustomAllowedHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-custom-allowed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	customHeaders := []string{"Content-Type", "Authorization", "X-Request-ID"}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedHeaders: customHeaders,
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send preflight request
	req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify custom headers are used (joined by ", ")
	allowedHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.Equal(t, "Content-Type, Authorization, X-Request-ID", allowedHeaders)

	cancel()
	<-errCh
}

// T005: Test that empty AllowedHeaders slice results in empty header (FR-008).
func TestServe_CORS_EmptyAllowedHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-empty-allowed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedHeaders: []string{}, // Empty - should result in empty header
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send preflight request
	req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify empty header is set (browser will only allow CORS-safelisted headers)
	allowedHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	assert.Empty(t, allowedHeaders)

	cancel()
	<-errCh
}

// T009: Test that nil ExposedHeaders uses default headers.
func TestServe_CORS_NilExposedHeaders_UsesDefaults(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-nil-exposed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				// ExposedHeaders is nil - should use DefaultExposedHeaders
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send actual request (not preflight) to check Expose-Headers
	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/finfocus.v1.CostSourceService/Name",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify defaults are used
	exposedHeaders := resp.Header.Get("Access-Control-Expose-Headers")
	assert.Equal(t, pluginsdk.DefaultExposedHeaders, exposedHeaders)

	cancel()
	<-errCh
}

// T010: Test that custom ExposedHeaders are used.
func TestServe_CORS_CustomExposedHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-custom-exposed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	customHeaders := []string{"X-Request-ID", "X-Trace-ID", "Grpc-Status"}

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				ExposedHeaders: customHeaders,
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send actual request to check Expose-Headers
	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/finfocus.v1.CostSourceService/Name",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify custom headers are used (joined by ", ")
	exposedHeaders := resp.Header.Get("Access-Control-Expose-Headers")
	assert.Equal(t, "X-Request-ID, X-Trace-ID, Grpc-Status", exposedHeaders)

	cancel()
	<-errCh
}

// T011: Test that empty ExposedHeaders slice results in empty header (FR-009).
func TestServe_CORS_EmptyExposedHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "cors-empty-exposed-headers-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				ExposedHeaders: []string{}, // Empty - should result in empty header
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send actual request to check Expose-Headers
	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/finfocus.v1.CostSourceService/Name",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Content-Type", "application/json")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	// Verify empty header is set (no custom headers exposed to JavaScript)
	exposedHeaders := resp.Header.Get("Access-Control-Expose-Headers")
	assert.Empty(t, exposedHeaders)

	cancel()
	<-errCh
}

// T021: Test custom MaxAge configuration (#229).
func TestServeConnect_CustomMaxAge(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "maxage-test"}
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				MaxAge:         intPtr(3600), // 1 hour instead of default 24h
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send preflight request and verify custom max-age
	req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "3600", resp.Header.Get("Access-Control-Max-Age"))

	cancel()
	<-errCh
}

// T022: Test MaxAge of zero disables caching (#229).
func TestServeConnect_MaxAgeZero(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "maxage-zero-test"}
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				MaxAge:         intPtr(0), // Disable caching
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Send preflight request and verify zero max-age
	req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, respErr := http.DefaultClient.Do(req)
	require.NoError(t, respErr)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "0", resp.Header.Get("Access-Control-Max-Age"))

	cancel()
	<-errCh
}

// intPtr is a helper to create a pointer to an int.
func intPtr(v int) *int {
	return &v
}

// T023: Benchmark corsMiddleware overhead for SC-005 compliance (<1μs per request).
func BenchmarkCORSMiddleware(b *testing.B) {
	// Create a simple handler that just returns OK
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware via test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}

	plugin := &connectTestPlugin{name: "cors-benchmark-test"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled:        true,
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedHeaders: []string{"Content-Type", "Authorization", "X-Request-ID"},
				ExposedHeaders: []string{"X-Request-ID", "Grpc-Status"},
			},
		})
	}()

	// Wait for server to be ready
	addr := listener.Addr().String()
	for range 50 {
		healthResp, healthErr := http.Get("http://" + addr + "/healthz")
		if healthErr == nil {
			healthResp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Benchmark preflight requests (OPTIONS)
	b.Run("preflight_request", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			req, _ := http.NewRequest(http.MethodOptions, "http://"+addr+"/finfocus.v1.CostSourceService/Name", nil)
			req.Header.Set("Origin", "http://localhost:3000")
			req.Header.Set("Access-Control-Request-Method", "POST")

			resp, doErr := http.DefaultClient.Do(req)
			if doErr != nil {
				b.Fatal(doErr)
			}
			resp.Body.Close()
		}
	})

	// Benchmark actual CORS requests
	b.Run("cors_request", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/finfocus.v1.CostSourceService/Name",
				bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Origin", "http://localhost:3000")
			req.Header.Set("Content-Type", "application/json")

			resp, doErr := http.DefaultClient.Do(req)
			if doErr != nil {
				b.Fatal(doErr)
			}
			resp.Body.Close()
		}
	})

	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		b.Fatal("server did not shut down in time")
	}

	// Silence unused variable warning
	_ = baseHandler
}

func TestServeConnect_ConcurrentRequests(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "concurrent-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled: true,
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	client := pbcconnect.NewCostSourceServiceClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Run 100 concurrent requests
	concurrency := 100
	doneCh := make(chan error, concurrency)

	for range concurrency {
		go func() {
			_, reqErr := client.Name(context.Background(), connect.NewRequest(&pbc.NameRequest{}))
			doneCh <- reqErr
		}()
	}

	for range concurrency {
		reqErr := <-doneCh
		require.NoError(t, reqErr)
	}

	cancel()
	<-errCh
}

func TestServeConnect_LargePayload(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &connectTestPlugin{name: "payload-test-plugin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled: true,
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	// Create >1MB payload
	largeData := make([]byte, 1024*1024+100) // 1MB + 100 bytes
	reqBody := bytes.NewReader(largeData)

	// Send raw HTTP request to bypass client-side limits if any
	resp, err := http.Post(
		"http://"+addr+"/finfocus.v1.CostSourceService/Name",
		"application/json",
		reqBody,
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be rejected with 4xx client error (likely 413 or 400)
	is4xx := resp.StatusCode >= 400 && resp.StatusCode < 500
	assert.True(t, is4xx, "Should reject large payload with 4xx status, got: %d", resp.StatusCode)
}

func TestServeConnect_GracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Plugin that sleeps
	// We need to mock a slow method. Currently connectTestPlugin returns immediately.
	// We can't easily modify it without changing struct.
	// Let's create a new type.
	slowPlugin := &slowPlugin{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   slowPlugin,
			Listener: listener,
			Web: pluginsdk.WebConfig{
				Enabled: true,
			},
		})
	}()

	addr := listener.Addr().String()
	waitForServer(t, addr)

	client := pbcconnect.NewCostSourceServiceClient(
		http.DefaultClient,
		"http://"+addr,
	)

	// Start a slow request
	reqDone := make(chan error)
	go func() {
		_, reqErr := client.Name(context.Background(), connect.NewRequest(&pbc.NameRequest{}))
		reqDone <- reqErr
	}()

	// Wait a bit to ensure request is in-flight
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	cancel()

	// Request should complete successfully
	select {
	case reqErr := <-reqDone:
		require.NoError(t, reqErr, "In-flight request should complete during graceful shutdown")
	case <-time.After(2 * time.Second):
		t.Fatal("Request timed out during shutdown")
	}

	<-errCh
}

// slowPlugin sleeps in Name().
type slowPlugin struct {
	connectTestPlugin
}

func (p *slowPlugin) Name() string {
	time.Sleep(500 * time.Millisecond)
	return "slow-plugin"
}
