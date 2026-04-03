// Package pluginsdk provides a development SDK for FinFocus plugins.
package pluginsdk

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

const (
	maxPayloadSize = 1 << 20 // 1MB
)

// portFlag is the --port command-line flag for specifying the gRPC server port.
// This is registered at package initialization time.
//
//nolint:gochecknoglobals // Package-level flag required for command-line parsing
var portFlag = flag.Int("port", 0, "TCP port for gRPC server (overrides FINFOCUS_PLUGIN_PORT)")

// ParsePortFlag returns the value of the --port command-line flag.
// Returns 0 if the flag was not specified or if flag.Parse() has not been called.
//
// IMPORTANT: The caller must call flag.Parse() before calling this function.
//
// Example usage in plugin main():
//
//	func main() {
//	    flag.Parse()  // Must be called first
//	    port := pluginsdk.ParsePortFlag()
//	    ctx := context.Background()
//	    if err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
//	        Plugin: &MyPlugin{},
//	        Port:   port,
//	    }); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func ParsePortFlag() int {
	if portFlag == nil {
		return 0
	}
	return *portFlag
}

// DefaultSupportsNotImplementedReason is the standardized message returned when
// a plugin does not implement the SupportsProvider interface.
const DefaultSupportsNotImplementedReason = "Supports capability not implemented by this plugin"

// DefaultReadHeaderTimeout is the timeout for reading HTTP headers.
// This prevents Slowloris-style attacks on the HTTP server.
const DefaultReadHeaderTimeout = 10 * time.Second

// DefaultReadTimeout is the timeout for reading the entire request including body.
// This prevents slow-read attacks where clients send data very slowly.
const DefaultReadTimeout = 60 * time.Second

// DefaultWriteTimeout is the timeout for writing the response.
// This prevents slow-write attacks and ensures resources are released.
const DefaultWriteTimeout = 30 * time.Second

// DefaultIdleTimeout is the maximum time to wait for the next request on a keep-alive connection.
// This prevents resource exhaustion from long-lived idle connections.
const DefaultIdleTimeout = 120 * time.Second

// DefaultShutdownTimeout is the timeout for graceful server shutdown.
// This prevents the server from blocking indefinitely during shutdown.
const DefaultShutdownTimeout = 30 * time.Second

// DefaultMaxHeaderBytes is the maximum size of request headers.
// This prevents header bomb attacks where clients send extremely large headers
// to exhaust server memory. Set to 1MB which is generous for normal use.
const DefaultMaxHeaderBytes = 1 << 20 // 1 MB

// ServerTimeouts configures HTTP server timeouts for the plugin server.
// All timeouts are optional - if not specified, sensible defaults are used.
// Consider increasing WriteTimeout for plugins with long-running operations
// like GetActualCost with large time ranges.
type ServerTimeouts struct {
	// ReadHeaderTimeout is the timeout for reading HTTP headers (default: 10s).
	// Prevents Slowloris-style attacks.
	ReadHeaderTimeout time.Duration

	// ReadTimeout is the timeout for reading the entire request (default: 60s).
	// Prevents slow-read attacks where clients send data very slowly.
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for writing the response (default: 30s).
	// Consider increasing for long-running operations.
	WriteTimeout time.Duration

	// IdleTimeout is the max time for the next request on keep-alive (default: 120s).
	// Prevents resource exhaustion from long-lived idle connections.
	IdleTimeout time.Duration

	// ShutdownTimeout is the timeout for graceful server shutdown (default: 30s).
	ShutdownTimeout time.Duration
}

// DefaultServerTimeouts returns the default server timeout configuration.
func DefaultServerTimeouts() ServerTimeouts {
	return ServerTimeouts{
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		ReadTimeout:       DefaultReadTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		ShutdownTimeout:   DefaultShutdownTimeout,
	}
}

// applyDefaults fills in any zero values with defaults.
func (t ServerTimeouts) applyDefaults() ServerTimeouts {
	if t.ReadHeaderTimeout == 0 {
		t.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}
	if t.ReadTimeout == 0 {
		t.ReadTimeout = DefaultReadTimeout
	}
	if t.WriteTimeout == 0 {
		t.WriteTimeout = DefaultWriteTimeout
	}
	if t.IdleTimeout == 0 {
		t.IdleTimeout = DefaultIdleTimeout
	}
	if t.ShutdownTimeout == 0 {
		t.ShutdownTimeout = DefaultShutdownTimeout
	}
	return t
}

// Plugin represents a FinFocus plugin implementation.
type Plugin interface {
	// Name returns the plugin name identifier.
	Name() string
	// GetProjectedCost calculates projected cost for a resource.
	GetProjectedCost(ctx context.Context, req *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error)
	// GetActualCost retrieves actual cost for a resource.
	GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error)
	// GetPricingSpec returns detailed pricing specification for a resource type.
	GetPricingSpec(ctx context.Context, req *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error)
	// EstimateCost returns an estimated monthly cost for a resource based on its type and configuration attributes.
	EstimateCost(ctx context.Context, req *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error)
}

// SupportsProvider is an optional interface that plugins can implement to indicate
// whether they support pricing for specific resource types. Plugins that do not
// implement this interface will receive a default "not supported" response.
type SupportsProvider interface {
	// Supports checks if the plugin supports pricing for the given resource.
	Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error)
}

// RecommendationsProvider is an optional interface that plugins can implement
// to provide cost optimization recommendations. Plugins that do not implement
// this interface will return an empty list when GetRecommendations is called.
type RecommendationsProvider interface {
	// GetRecommendations retrieves cost optimization recommendations.
	GetRecommendations(ctx context.Context, req *pbc.GetRecommendationsRequest) (
		*pbc.GetRecommendationsResponse, error)
}

// BudgetsProvider is an optional interface that plugins can implement
// to provide budget information from cloud cost management services.
// Plugins that do not implement this interface will return Unimplemented
// when GetBudgets is called.
type BudgetsProvider interface {
	// GetBudgets retrieves budget information from the cost management service.
	GetBudgets(ctx context.Context, req *pbc.GetBudgetsRequest) (
		*pbc.GetBudgetsResponse, error)
}

// DismissProvider is an optional interface that plugins can implement
// to allow dismissing cost optimization recommendations. Plugins that do not
// implement this interface will return Unimplemented when DismissRecommendation is called.
type DismissProvider interface {
	// DismissRecommendation marks a recommendation as dismissed/ignored.
	DismissRecommendation(ctx context.Context, req *pbc.DismissRecommendationRequest) (
		*pbc.DismissRecommendationResponse, error)
}

// PluginInfoProvider is an optional interface that plugins can implement
// to provide custom metadata via GetPluginInfo RPC. Plugins that do not
// implement this interface will return metadata from ServeConfig.PluginInfo
// if configured, or Unimplemented if no PluginInfo is provided.
//
// This interface is useful when plugins need dynamic metadata that can't
// be determined at startup, such as runtime-computed values.
type PluginInfoProvider interface {
	// GetPluginInfo returns metadata about the plugin including name, version,
	// spec version, supported providers, and optional key-value metadata.
	GetPluginInfo(ctx context.Context, req *pbc.GetPluginInfoRequest) (
		*pbc.GetPluginInfoResponse, error)
}

// BatchCostHandler is an optional interface that plugins can implement
// for optimized batch processing. Plugins that do not implement this
// interface are served via SDK fallback processing.
type BatchCostHandler interface {
	// BatchCost queries cost data for multiple resources in a single request.
	BatchCost(ctx context.Context, req *pbc.BatchCostRequest) (
		*pbc.BatchCostResponse, error)
}

// RegistryLookup defines the interface for looking up plugins by provider and region.
// This is used to validate incoming Supports requests against registered plugins.
type RegistryLookup interface {
	// FindPlugin returns the plugin name for the given provider and region.
	// Returns empty string if no plugin is registered for the combination.
	FindPlugin(provider, region string) string
}

// DefaultRegistryLookup provides a no-op registry lookup that always returns empty.
// This causes all Supports() calls to return InvalidArgument since no plugin
// can be found. Use a real RegistryLookup implementation in production.
type DefaultRegistryLookup struct{}

// FindPlugin always returns empty string indicating no plugin is registered.
func (d *DefaultRegistryLookup) FindPlugin(_, _ string) string {
	return ""
}

// Server wraps a Plugin implementation with a gRPC server.
type Server struct {
	pbc.UnimplementedCostSourceServiceServer

	plugin   Plugin
	registry RegistryLookup
	logger   zerolog.Logger
	// pluginInfo is optional plugin metadata for GetPluginInfo RPC.
	//
	// Thread Safety: This field is set during NewServerWithOptions() construction
	// before the server accepts any requests. The happens-before relationship
	// established by the construction pattern (caller creates Server, then passes
	// it to gRPC registration) ensures all subsequent reads by concurrent
	// GetPluginInfo handlers see the initialized value without additional
	// synchronization required.
	pluginInfo *PluginInfo

	// globalCapabilities are the capabilities inferred from the plugin interface
	// or explicitly configured. Populated at server initialization.
	globalCapabilities []pbc.PluginCapability

	// maxBatchSize is the configured per-plugin batch size limit.
	// Defaults to DefaultMaxBatchSize when not configured.
	maxBatchSize int32

	// batchWorkers controls fallback concurrency when the plugin does not
	// implement BatchCostHandler. Defaults to DefaultBatchWorkers.
	batchWorkers int
}

// NewServer creates a Server that wraps the provided Plugin and initializes sensible defaults.
//
// The returned Server uses a DefaultRegistryLookup for provider resolution, a package-default logger,
// and infers global capabilities from the plugin. Batch-related defaults are applied: maxBatchSize is
// set to DefaultMaxBatchSize and batchWorkers to DefaultBatchWorkers.
func NewServer(plugin Plugin) *Server {
	return &Server{
		plugin:             plugin,
		registry:           &DefaultRegistryLookup{},
		logger:             newDefaultLogger(),
		globalCapabilities: inferCapabilities(plugin),
		maxBatchSize:       DefaultMaxBatchSize,
		batchWorkers:       DefaultBatchWorkers,
	}
}

// NewServerWithRegistry creates a Server with a custom registry lookup.
// If registry is nil, DefaultRegistryLookup is used.
// GetPluginInfo will return Unimplemented (legacy plugin behavior).
func NewServerWithRegistry(plugin Plugin, registry RegistryLookup) *Server {
	return NewServerWithOptions(plugin, registry, nil, nil)
}

// NewServerWithOptions creates a Server with custom registry, logger, and plugin info.
// If registry is nil, DefaultRegistryLookup is used.
// If logger is nil, a default logger is used.
// If info is nil, GetPluginInfo will return Unimplemented (legacy plugin behavior).
//
// Thread Safety: pluginInfo is set during construction before the server accepts
// connections, so it is safe for concurrent use once returned.
func NewServerWithOptions(plugin Plugin, registry RegistryLookup, logger *zerolog.Logger, info *PluginInfo) *Server {
	if registry == nil {
		registry = &DefaultRegistryLookup{}
	}
	var log zerolog.Logger
	if logger != nil {
		log = *logger
	} else {
		log = newDefaultLogger()
	}

	// Determine capabilities: use explicitly configured if available, otherwise infer
	var caps []pbc.PluginCapability
	if info != nil && len(info.Capabilities) > 0 {
		caps = append([]pbc.PluginCapability{}, info.Capabilities...)
	} else {
		caps = inferCapabilities(plugin)
	}

	return &Server{
		plugin:             plugin,
		registry:           registry,
		logger:             log,
		pluginInfo:         info, // Thread-safe: set during construction before requests accepted
		globalCapabilities: caps,
		maxBatchSize:       DefaultMaxBatchSize,
		batchWorkers:       DefaultBatchWorkers,
	}
}

// GetGlobalCapabilities returns a copy of the capabilities supported by this server.
func (s *Server) GetGlobalCapabilities() []pbc.PluginCapability {
	return append([]pbc.PluginCapability{}, s.globalCapabilities...)
}

// SetBatchWorkersForTesting sets the batch worker pool size for testing purposes.
// workers must be in the range [MinBatchWorkers, MaxBatchWorkers]. Values outside
// this range are clamped to the nearest bound.
//
// This method intentionally does NOT mirror resolveBatchWorkers semantics used by
// the Serve path: values below MinBatchWorkers clamp to MinBatchWorkers here,
// while Serve treats non-positive values as DefaultBatchWorkers. The stricter
// clamping behavior allows tests to force single-worker fallback execution.
//
// This method is exported to support cross-package integration tests and is not
// protected by build tags. Production code should configure BatchWorkers via
// ServeConfig to preserve production defaults.
//
// This method should only be used in tests to verify concurrency behavior.
// Returns the server to allow method chaining.
//
// Example:
//
//	server := pluginsdk.NewServer(plugin).SetBatchWorkersForTesting(1)
func (s *Server) SetBatchWorkersForTesting(workers int) *Server {
	if workers < MinBatchWorkers {
		workers = MinBatchWorkers
	}
	if workers > MaxBatchWorkers {
		workers = MaxBatchWorkers
	}
	s.batchWorkers = workers
	return s
}

// Name implements the gRPC Name method.
func (s *Server) Name(_ context.Context, _ *pbc.NameRequest) (*pbc.NameResponse, error) {
	return &pbc.NameResponse{Name: s.plugin.Name()}, nil
}

// GetPluginInfo implements the gRPC GetPluginInfo method.
// It retrieves metadata and capabilities from the plugin, with support for
// auto-discovery and backward compatibility.
//
// Priority order for response:
//  1. If the plugin implements PluginInfoProvider, delegate to it
//  2. If PluginInfo was configured via ServeConfig, return it
//  3. Return Unimplemented error (enables graceful degradation for legacy plugins)
//
// When calling GetPluginInfo on potentially legacy plugins, consumers should handle
// the Unimplemented error gracefully:
//
//	resp, err := client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
//	if err != nil {
//	    if status.Code(err) == codes.Unimplemented {
//	        // Legacy plugin - use fallback values
//	        return &PluginMetadata{Name: "unknown", Version: "unknown"}
//	    }
//	    return nil, fmt.Errorf("GetPluginInfo failed: %w", err)
//	}
func (s *Server) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	// Check if plugin implements PluginInfoProvider interface
	if provider, ok := s.plugin.(PluginInfoProvider); ok {
		return s.handleProviderPluginInfo(ctx, req, provider)
	}

	// Use configured PluginInfo if available
	if s.pluginInfo != nil {
		return s.handleConfiguredPluginInfo()
	}

	// Log at debug level to aid debugging while maintaining graceful degradation
	s.logger.Debug().
		Str("plugin", s.plugin.Name()).
		Msg("GetPluginInfo not implemented (legacy plugin)")

	// Return Unimplemented for legacy plugins (enables graceful degradation)
	return nil, status.Error(codes.Unimplemented, "GetPluginInfo not implemented")
}

// handleProviderPluginInfo handles GetPluginInfo for plugins implementing PluginInfoProvider.
func (s *Server) handleProviderPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
	provider PluginInfoProvider,
) (*pbc.GetPluginInfoResponse, error) {
	resp, err := provider.GetPluginInfo(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).Msg("GetPluginInfo handler error")
		return nil, status.Error(codes.Internal, "plugin failed to retrieve metadata")
	}

	if resp == nil {
		s.logger.Error().Msg("GetPluginInfo returned nil response")
		return nil, status.Error(codes.Internal, "unable to retrieve plugin metadata")
	}

	if resp.GetName() == "" || resp.GetVersion() == "" || resp.GetSpecVersion() == "" {
		s.logger.Error().
			Str("name", resp.GetName()).
			Str("version", resp.GetVersion()).
			Str("spec_version", resp.GetSpecVersion()).
			Msg("GetPluginInfo returned incomplete response")
		return nil, status.Error(codes.Internal, "plugin metadata is incomplete")
	}

	if specErr := ValidateSpecVersion(resp.GetSpecVersion()); specErr != nil {
		s.logger.Error().Err(specErr).Msg("GetPluginInfo returned invalid spec_version")
		return nil, status.Error(codes.Internal, "plugin reported an invalid specification version")
	}

	// Enrich response with server-side batch metadata when the plugin supports batch cost.
	// Create a new map to avoid mutating the plugin-returned response in place.
	if containsCapability(resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST) {
		meta := make(map[string]string, len(resp.GetMetadata())+1)
		for k, v := range resp.GetMetadata() {
			meta[k] = v
		}
		meta["max_batch_size"] = strconv.Itoa(int(s.maxBatchSize))
		resp.Metadata = meta
	}

	return resp, nil
}

// handleConfiguredPluginInfo handles GetPluginInfo using the server's configured PluginInfo.
func (s *Server) handleConfiguredPluginInfo() (*pbc.GetPluginInfoResponse, error) {
	// Create defensive copies to prevent concurrent modification
	providers := append([]string{}, s.pluginInfo.Providers...)

	var metadata map[string]string
	if s.pluginInfo.Metadata != nil {
		metadata = make(map[string]string, len(s.pluginInfo.Metadata))
		for k, v := range s.pluginInfo.Metadata {
			metadata[k] = v
		}
	}

	// Determine capabilities: use explicit if set, otherwise use globalCapabilities
	var capabilities []pbc.PluginCapability
	if len(s.pluginInfo.Capabilities) > 0 {
		capabilities = append([]pbc.PluginCapability{}, s.pluginInfo.Capabilities...)
	} else {
		capabilities = append([]pbc.PluginCapability{}, s.globalCapabilities...)
	}

	// Add legacy capability metadata for backward compatibility
	// Use warning-aware function to detect and log invalid/unmapped capabilities
	if len(capabilities) > 0 {
		legacyMeta, warnings := CapabilitiesToLegacyMetadataWithWarnings(capabilities)

		// Log any warnings about unmapped capabilities
		for _, w := range warnings {
			s.logger.Warn().
				Int32("capability", int32(w.Capability)).
				Str("reason", w.Reason).
				Msg("Capability has no legacy metadata mapping")
		}

		if metadata == nil {
			metadata = make(map[string]string, len(legacyMeta))
		}
		for key, val := range legacyMeta {
			metadata[key] = val
		}

		if containsCapability(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST) {
			metadata["max_batch_size"] = strconv.Itoa(int(s.maxBatchSize))
		}
	}

	return &pbc.GetPluginInfoResponse{
		Name:         s.pluginInfo.Name,
		Version:      s.pluginInfo.Version,
		SpecVersion:  s.pluginInfo.SpecVersion,
		Providers:    providers,
		Metadata:     metadata,
		Capabilities: capabilities,
	}, nil
}

// containsCapability reports whether target is present in capabilities.
func containsCapability(capabilities []pbc.PluginCapability, target pbc.PluginCapability) bool {
	return slices.Contains(capabilities, target)
}

// GetProjectedCost implements the gRPC GetProjectedCost method.
func (s *Server) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return s.plugin.GetProjectedCost(ctx, req)
}

// GetActualCost implements the gRPC GetActualCost method.
func (s *Server) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	return s.plugin.GetActualCost(ctx, req)
}

// GetPricingSpec implements the gRPC GetPricingSpec method.
func (s *Server) GetPricingSpec(
	ctx context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	return s.plugin.GetPricingSpec(ctx, req)
}

// EstimateCost implements the gRPC EstimateCost method.
func (s *Server) EstimateCost(
	ctx context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return s.plugin.EstimateCost(ctx, req)
}

// Supports implements the gRPC Supports method.
// It performs two-step validation: first checks registry for plugin by provider/region,
// then delegates to the plugin's Supports method if implemented.
func (s *Server) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	// Validate request has resource descriptor
	if req.GetResource() == nil {
		return nil, status.Error(codes.InvalidArgument, "resource descriptor is required")
	}

	resource := req.GetResource()
	provider := resource.GetProvider()
	region := resource.GetRegion()

	// Step 1: Registry lookup - validate provider/region combination
	pluginName := s.registry.FindPlugin(provider, region)
	if pluginName == "" {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"no plugin registered for provider %q and region %q",
			provider,
			region,
		)
	}

	// Step 2: Check if plugin implements SupportsProvider
	supportsProvider, ok := s.plugin.(SupportsProvider)
	var resp *pbc.SupportsResponse
	var err error

	if !ok {
		// Plugin does not implement SupportsProvider - return default response
		resp = &pbc.SupportsResponse{
			Supported: false,
			Reason:    DefaultSupportsNotImplementedReason,
		}
	} else {
		// Delegate to plugin's Supports method
		resp, err = supportsProvider.Supports(ctx, req)
		if err != nil {
			// Log the detailed error server-side for debugging
			s.logger.Error().
				Str(FieldResourceType, resource.GetResourceType()).
				Str(FieldProvider, provider).
				Str(FieldRegion, region).
				Err(err).
				Msg("Supports handler error")
			// Return generic message to client (internal error details not exposed)
			return nil, status.Error(codes.Internal, "plugin failed to execute")
		}
	}

	// Auto-populate capabilities based on implemented interfaces (User Story 1, Issue #194)
	if resp != nil {
		// Populate typed capabilities if not already set by the plugin (Inherit Global)
		if len(resp.GetCapabilitiesEnum()) == 0 {
			caps := append([]pbc.PluginCapability{}, s.globalCapabilities...)
			resp.CapabilitiesEnum = caps
		}

		// Always sync legacy capabilities map from enum to ensure consistency.
		// This handles both cases: plugin didn't set capabilities, or plugin manually set Capabilities map.
		// By always regenerating from the enum, we ensure both formats represent the same capabilities.
		if len(resp.GetCapabilitiesEnum()) > 0 {
			legacyMeta, warnings := CapabilitiesToLegacyMetadataWithWarnings(resp.GetCapabilitiesEnum())

			// Log any warnings about unmapped capabilities
			for _, w := range warnings {
				s.logger.Warn().
					Int32("capability", int32(w.Capability)).
					Str("reason", w.Reason).
					Msg("Capability has no legacy metadata mapping in Supports response")
			}

			// Convert map[string]string to map[string]bool for SupportsResponse
			boolMap := make(map[string]bool, len(legacyMeta))
			for k := range legacyMeta {
				boolMap[k] = true
			}
			resp.Capabilities = boolMap
		}
	}

	return resp, nil
}

// GetRecommendations implements the gRPC GetRecommendations method.
// GetRecommendations handles GetRecommendations RPC requests.
// If the plugin implements RecommendationsProvider, delegates to it.
// Otherwise returns an empty list (not an error) per FR-012.
func (s *Server) GetRecommendations(
	ctx context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	// Log incoming request with filter details
	filter := req.GetFilter()
	s.logger.Debug().
		Str(FieldFilterCategory, filter.GetCategory().String()).
		Str(FieldFilterActionType, filter.GetActionType().String()).
		Int32(FieldPageSize, req.GetPageSize()).
		Msg("GetRecommendations request received")

	// Check if plugin implements RecommendationsProvider
	recProvider, ok := s.plugin.(RecommendationsProvider)
	if !ok {
		// Plugin does not implement recommendations - return empty list per FR-012
		// Include projection_period from request for client consistency
		s.logger.Debug().
			Int(FieldRecommendationCount, 0).
			Msg("GetRecommendations returning empty response (not implemented)")
		return &pbc.GetRecommendationsResponse{
			Recommendations: []*pbc.Recommendation{},
			Summary: &pbc.RecommendationSummary{
				TotalRecommendations: 0,
				ProjectionPeriod:     req.GetProjectionPeriod(),
			},
			NextPageToken: "",
		}, nil
	}

	// Delegate to plugin's GetRecommendations method
	resp, err := recProvider.GetRecommendations(ctx, req)
	if err != nil {
		s.logger.Error().
			Str(FieldFilterCategory, filter.GetCategory().String()).
			Str(FieldFilterActionType, filter.GetActionType().String()).
			Err(err).
			Msg("GetRecommendations handler error")
		return nil, status.Error(codes.Internal, "plugin failed to execute GetRecommendations")
	}

	// Guard against nil response from plugin.
	if resp == nil {
		s.logger.Error().
			Str(FieldFilterCategory, filter.GetCategory().String()).
			Str(FieldFilterActionType, filter.GetActionType().String()).
			Msg("GetRecommendations handler returned a nil response")
		return nil, status.Error(codes.Internal, "plugin returned a nil response")
	}

	// Log successful response with summary
	summary := resp.GetSummary()
	if summary == nil {
		s.logger.Error().
			Str(FieldFilterCategory, filter.GetCategory().String()).
			Str(FieldFilterActionType, filter.GetActionType().String()).
			Msg("GetRecommendations response has nil summary")
		return nil, status.Error(codes.Internal, "plugin returned response with nil summary")
	}
	s.logger.Info().
		Int32(FieldRecommendationCount, summary.GetTotalRecommendations()).
		Float64(FieldTotalSavings, summary.GetTotalEstimatedSavings()).
		Msg("GetRecommendations completed")

	return resp, nil
}

// GetBudgets implements the gRPC GetBudgets method.
// GetBudgets handles GetBudgets RPC requests.
// If the plugin implements BudgetsProvider, delegates to it.
// Otherwise returns Unimplemented error per specification.
func (s *Server) GetBudgets(
	ctx context.Context,
	req *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	// Log incoming request
	s.logger.Debug().
		Bool(FieldIncludeStatus, req.GetIncludeStatus()).
		Msg("GetBudgets request received")

	// Check if plugin implements BudgetsProvider
	budgetsProvider, ok := s.plugin.(BudgetsProvider)
	if !ok {
		// Plugin does not implement budgets - return Unimplemented per spec
		s.logger.Debug().Msg("GetBudgets returning Unimplemented (not supported by plugin)")
		return nil, status.Error(codes.Unimplemented, "plugin does not support GetBudgets")
	}

	// Delegate to plugin's GetBudgets method
	resp, err := budgetsProvider.GetBudgets(ctx, req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Msg("GetBudgets handler error")
		return nil, status.Error(codes.Internal, "plugin failed to execute GetBudgets")
	}

	// Guard against nil response from plugin
	if resp == nil {
		s.logger.Error().Msg("GetBudgets handler returned a nil response")
		return nil, status.Error(codes.Internal, "plugin returned a nil response")
	}

	// Log successful response
	summary := resp.GetSummary()
	if summary != nil {
		s.logger.Info().
			Int32(FieldTotalBudgets, summary.GetTotalBudgets()).
			Int32(FieldBudgetsOk, summary.GetBudgetsOk()).
			Int32(FieldBudgetsWarning, summary.GetBudgetsWarning()).
			Int32(FieldBudgetsCritical, summary.GetBudgetsCritical()).
			Int32(FieldBudgetsExceeded, summary.GetBudgetsExceeded()).
			Msg("GetBudgets completed")
	}

	return resp, nil
}

// DismissRecommendation implements the gRPC DismissRecommendation method.
// If the plugin implements DismissProvider, delegates to it.
// Otherwise returns Unimplemented error per specification.
func (s *Server) DismissRecommendation(
	ctx context.Context,
	req *pbc.DismissRecommendationRequest,
) (*pbc.DismissRecommendationResponse, error) {
	// Log incoming request
	s.logger.Debug().
		Str("recommendation_id", req.GetRecommendationId()).
		Msg("DismissRecommendation request received")

	// Check if plugin implements DismissProvider
	dismissProvider, ok := s.plugin.(DismissProvider)
	if !ok {
		// Plugin does not implement dismiss - return Unimplemented per spec
		s.logger.Debug().Msg("DismissRecommendation returning Unimplemented (not supported by plugin)")
		return nil, status.Error(codes.Unimplemented, "plugin does not support DismissRecommendation")
	}

	// Delegate to plugin's DismissRecommendation method
	resp, err := dismissProvider.DismissRecommendation(ctx, req)
	if err != nil {
		s.logger.Error().
			Str("recommendation_id", req.GetRecommendationId()).
			Err(err).
			Msg("DismissRecommendation handler error")
		return nil, status.Error(codes.Internal, "plugin failed to execute DismissRecommendation")
	}

	// Guard against nil response from plugin
	if resp == nil {
		s.logger.Error().Msg("DismissRecommendation handler returned a nil response")
		return nil, status.Error(codes.Internal, "plugin returned a nil response")
	}

	// Log successful response
	s.logger.Info().
		Str("recommendation_id", req.GetRecommendationId()).
		Bool("success", resp.GetSuccess()).
		Msg("DismissRecommendation completed")

	return resp, nil
}

// BatchCost implements the gRPC BatchCost method.
// It validates requests, dispatches to custom handler when available,
// and falls back to bounded parallel processing otherwise.
func (s *Server) BatchCost(
	ctx context.Context,
	req *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	startedAt := logBatchCostRequest(s.logger, req)

	result, err := ValidateBatchCostRequest(req, s.maxBatchSize)
	if err != nil {
		return nil, err
	}
	if result.EarlyResponse != nil {
		errorCount := logBatchCostResourceErrors(s.logger, result.EarlyResponse.GetResults())
		logBatchCostCompletion(s.logger, startedAt, result.EarlyResponse, errorCount)
		return result.EarlyResponse, nil
	}

	if handler, hasCustomHandler := s.plugin.(BatchCostHandler); hasCustomHandler {
		req.QueryType = NormalizeCostQueryType(req.GetQueryType())
		resp, batchErr := handler.BatchCost(ctx, req)
		if batchErr != nil {
			s.logger.Error().Err(batchErr).Msg("BatchCost handler error")
			return nil, status.Error(codes.Internal, "plugin failed to execute BatchCost")
		}
		if resp == nil {
			return nil, status.Error(codes.Internal, "plugin returned nil BatchCost response")
		}
		if resp.GetMaxBatchSize() <= 0 {
			resp.MaxBatchSize = s.maxBatchSize
		}

		errorCount := logBatchCostResourceErrors(s.logger, resp.GetResults())
		logBatchCostCompletion(s.logger, startedAt, resp, errorCount)
		return resp, nil
	}

	// Fallback path: skip proto.Clone because batchCostFallback is read-only on req.
	// It normalizes query_type into a local variable, never writing back to req.
	resp := batchCostFallback(ctx, s.plugin, req, s.maxBatchSize, s.batchWorkers)
	errorCount := logBatchCostResourceErrors(s.logger, resp.GetResults())
	logBatchCostCompletion(s.logger, startedAt, resp, errorCount)
	return resp, nil
}

// ServeConfig holds configuration for serving a plugin.
type ServeConfig struct {
	// Plugin is the implementation of the cost source service.
	Plugin Plugin

	// Registry is an optional registry lookup for validating supports requests.
	Registry RegistryLookup

	// PluginInfo is optional plugin metadata returned by GetPluginInfo RPC.
	// If set, the Server will return this information when GetPluginInfo is called.
	// If the Plugin implements PluginInfoProvider interface, that takes precedence.
	// If neither is set, GetPluginInfo returns Unimplemented (graceful degradation).
	PluginInfo *PluginInfo

	// Port is the TCP port to listen on. If 0, it reads from FINFOCUS_PLUGIN_PORT
	// environment variable or assigns a random available port if that's also 0.
	Port int

	// Listener is an optional pre-configured listener.
	// If provided, Port is ignored and this listener is used for serving.
	Listener net.Listener

	// Logger is an optional custom logger. If nil, a default logger is used.
	Logger *zerolog.Logger

	// UnaryInterceptors is an optional list of gRPC unary server interceptors
	// to chain after the built-in TracingUnaryServerInterceptor.
	UnaryInterceptors []grpc.UnaryServerInterceptor

	// Web holds configuration for gRPC-Web and CORS support.
	Web WebConfig

	// Timeouts configures HTTP server timeouts.
	Timeouts *ServerTimeouts

	// MaxBatchSize is the per-plugin batch size limit for BatchCost RPC.
	// Values <= 0 default to DefaultMaxBatchSize. Values > MaxBatchSize are clamped.
	MaxBatchSize int

	// BatchWorkers controls fallback concurrency for BatchCost when the plugin
	// does not implement BatchCostHandler.
	// Values <= 0 default to DefaultBatchWorkers.
	BatchWorkers int
}

// resolvePort determines the port to use with the following priority:
//  1. requested (set from --port flag via ParsePortFlag(), or explicitly configured in ServeConfig.Port)
//  2. FINFOCUS_PLUGIN_PORT env var (via GetPort(), with fallback to PULUMICOST_PLUGIN_PORT)
//  3. 0 (ephemeral port - OS assigns available port)
//
// Note: The generic PORT env var is NOT supported to avoid multi-plugin conflicts.
// When finfocus-core spawns multiple plugins (e.g., aws-public + aws-ce), each
// needs a unique port. Using --port flag allows the core to allocate distinct ports.
func resolvePort(requested int) int {
	if requested > 0 {
		return requested
	}
	// Use centralized GetPort() which reads FINFOCUS_PLUGIN_PORT with legacy fallback
	return GetPort()
}

func listenOnLoopback(ctx context.Context, port int) (net.Listener, *net.TCPAddr, error) {
	address := "127.0.0.1:0"
	if port > 0 {
		address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		closeErr := listener.Close()
		if closeErr != nil {
			return nil, nil, errors.Join(
				errors.New("listener address is not TCP"),
				fmt.Errorf("closing listener: %w", closeErr),
			)
		}
		return nil, nil, errors.New("listener address is not TCP")
	}

	return listener, tcpAddr, nil
}

func announcePort(listener net.Listener, addr *net.TCPAddr) error {
	if _, err := fmt.Fprintf(os.Stdout, "PORT=%d\n", addr.Port); err != nil {
		closeErr := listener.Close()
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("writing port: %w", err),
				fmt.Errorf("closing listener: %w", closeErr),
			)
		}
		return fmt.Errorf("writing port: %w", err)
	}
	return nil
}

// validateCORSConfig checks WebConfig for invalid CORS configurations.
// Returns an error if wildcard is mixed with specific origins or if AllowCredentials
// is used with wildcard origin (which is a security risk per MDN documentation).
func validateCORSConfig(web WebConfig) error {
	if !web.Enabled || len(web.AllowedOrigins) == 0 {
		return nil
	}

	// Check for wildcard mixed with specific origins (undefined behavior)
	hasWildcard := false
	hasSpecific := false
	for _, origin := range web.AllowedOrigins {
		if origin == "*" {
			hasWildcard = true
		} else {
			hasSpecific = true
		}
		// Early exit: once we've found both types, no need to continue scanning
		if hasWildcard && hasSpecific {
			return errors.New("AllowedOrigins cannot mix wildcard '*' with specific origins; " +
				"use either '*' alone or a list of specific origins")
		}
	}

	// AllowCredentials with wildcard origin is a security risk
	// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials
	if web.AllowCredentials && hasWildcard {
		return errors.New("AllowCredentials cannot be used with wildcard origin '*'; " +
			"specify explicit origins instead for security")
	}

	return nil
}

// Serve starts a network server for the provided plugin according to the ServeConfig.
//
// It validates PluginInfo and CORS configuration early, announces the chosen port on stdout,
// constructs the server instance, and selects either gRPC or Connect (gRPC-Web + Connect)
// serving mode based on the Web configuration.
//
// When config.Web.Enabled is false (default), it starts a standard gRPC server.
// When config.Web.Enabled is true, it starts a connect-go server that supports
// gRPC, gRPC-Web, and Connect protocols simultaneously on the same port.
//
// It uses config.Port when > 0; if config.Port is 0 it reads the FINFOCUS_PLUGIN_PORT
// environment variable and falls back to an ephemeral port when none is provided.
// It returns an error if configuration validation fails, a listener cannot be created
// or announced, or if the server fails while serving (including context cancellation).
func Serve(ctx context.Context, config ServeConfig) error {
	// Validate PluginInfo early (before acquiring resources like listeners)
	// This prevents resource leaks if validation fails (T020)
	if config.PluginInfo != nil {
		if valErr := config.PluginInfo.Validate(); valErr != nil {
			return fmt.Errorf("invalid PluginInfo in ServeConfig: %w", valErr)
		}
	}

	// Validate CORS configuration early (before acquiring resources)
	if corsErr := validateCORSConfig(config.Web); corsErr != nil {
		return corsErr
	}

	var listener net.Listener
	var tcpAddr *net.TCPAddr
	var err error

	if config.Listener != nil {
		listener = config.Listener
		var ok bool
		tcpAddr, ok = listener.Addr().(*net.TCPAddr)
		if !ok {
			return errors.New("provided listener address is not TCP")
		}
	} else {
		port := resolvePort(config.Port)
		listener, tcpAddr, err = listenOnLoopback(ctx, port)
		if err != nil {
			return err
		}
	}

	if announceErr := announcePort(listener, tcpAddr); announceErr != nil {
		return announceErr
	}

	// Create the core server that wraps the plugin (pluginInfo set at construction)
	server := NewServerWithOptions(config.Plugin, config.Registry, config.Logger, config.PluginInfo)
	server.maxBatchSize = resolveBatchSize(config.MaxBatchSize)
	server.batchWorkers = resolveBatchWorkers(config.BatchWorkers)

	// Choose serving mode based on WebConfig
	if config.Web.Enabled {
		return serveConnect(ctx, listener, server, config)
	}
	return serveGRPC(ctx, listener, server, config)
}

// serveGRPC starts a standard gRPC server (legacy mode).
func serveGRPC(ctx context.Context, listener net.Listener, server *Server, config ServeConfig) error {
	// Build interceptor chain: tracing first, then user interceptors
	interceptors := make([]grpc.UnaryServerInterceptor, 0, 1+len(config.UnaryInterceptors))
	interceptors = append(interceptors, TracingUnaryServerInterceptor())
	interceptors = append(interceptors, config.UnaryInterceptors...)

	// Create and register server with interceptor chain
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(interceptors...),
	)
	pbc.RegisterCostSourceServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	// Create channels for goroutine coordination
	shutdownComplete := make(chan struct{})
	serverDone := make(chan struct{})

	// Handle context cancellation with proper synchronization
	go func() {
		defer close(shutdownComplete)
		select {
		case <-ctx.Done():
			grpcServer.GracefulStop()
		case <-serverDone:
			// Server already stopped, no need to shut down
			return
		}
	}()

	// Start serving
	err := grpcServer.Serve(listener)
	close(serverDone)

	// Wait for shutdown goroutine to complete (prevents goroutine leak)
	<-shutdownComplete

	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// serveConnect starts an HTTP server that exposes the plugin over Connect, gRPC-Web, and gRPC (using h2c).
// It registers the CostSource service and the gRPC health service, optionally exposes a /healthz endpoint,
// applies CORS when configured, enforces a 1MB request payload limit, and uses the provided timeouts.
// The server shuts down gracefully when ctx is canceled.
// It returns ctx.Err() if shutdown was initiated by the provided context, or the underlying serve error otherwise.
func serveConnect(ctx context.Context, listener net.Listener, server *Server, config ServeConfig) error {
	// Create HTTP mux for routing
	mux := http.NewServeMux()

	// Build connect handler options (currently none; CORS is applied via middleware below)
	var handlerOpts []connect.HandlerOption

	// Create connect handler from our server
	connectHandler := NewConnectHandler(server)
	path, handler := pbcconnect.NewCostSourceServiceHandler(connectHandler, handlerOpts...)
	mux.Handle(path, handler)

	// Detect HealthChecker from plugin
	var customChecker HealthChecker
	if hc, ok := server.plugin.(HealthChecker); ok {
		customChecker = hc
	}

	// Register gRPC health check service (grpc.health.v1.Health/Check)
	// This provides standard gRPC health checking protocol support
	healthChecker := grpchealth.NewStaticChecker(
		// Report CostSourceService as serving
		pbcconnect.CostSourceServiceName,
	)
	healthPath, healthHandler := grpchealth.NewHandler(healthChecker, handlerOpts...)
	mux.Handle(healthPath, healthHandler)

	// Add simple HTTP health endpoint if enabled
	if config.Web.EnableHealthEndpoint {
		mux.Handle("/healthz", HealthHandler(customChecker))
	}

	// Wrap with h2c for HTTP/2 cleartext support (required for gRPC protocol)
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	// Apply CORS if configured
	finalHandler := h2cHandler
	if len(config.Web.AllowedOrigins) > 0 {
		finalHandler = corsMiddleware(h2cHandler, config.Web)
	}

	// Apply payload size limit (1MB) to prevent DoS
	finalHandler = payloadLimitMiddleware(finalHandler, maxPayloadSize)

	// Resolve timeouts with defaults
	timeouts := DefaultServerTimeouts()
	if config.Timeouts != nil {
		timeouts = config.Timeouts.applyDefaults()
	}

	// Create HTTP server with timeouts and limits to prevent DoS attacks
	httpServer := &http.Server{
		Handler:           finalHandler,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		ReadTimeout:       timeouts.ReadTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		IdleTimeout:       timeouts.IdleTimeout,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
	}

	// Create channels for goroutine coordination
	shutdownComplete := make(chan struct{})
	serverDone := make(chan struct{})

	// Handle context cancellation with proper synchronization
	//nolint:gosec // G118: parent ctx is cancelled; fresh context needed for graceful shutdown
	go func() {
		defer close(shutdownComplete)
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.ShutdownTimeout)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
		case <-serverDone:
			// Server already stopped, no need to shut down
			return
		}
	}()

	// Start serving
	err := httpServer.Serve(listener)
	close(serverDone)

	// Wait for shutdown goroutine to complete (prevents goroutine leak)
	<-shutdownComplete

	if err != nil && err != http.ErrServerClosed {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	return nil
}

// resolveHeaderValue computes the header value based on nil/empty/populated semantics.
// If headers is nil, defaultValue is returned. If headers is an empty slice, the empty
// string is returned. Otherwise the slice elements are joined with ", ".
func resolveHeaderValue(headers []string, defaultValue string) string {
	if headers == nil {
		return defaultValue
	}
	if len(headers) == 0 {
		return ""
	}
	return strings.Join(headers, ", ")
}

// payloadLimitMiddleware wraps next and limits the size of the incoming request body.
// It first performs an early Content-Length check to reject obviously oversized requests
// without reading any data, then replaces r.Body with an http.MaxBytesReader that enforces
// the maxBytes limit for requests where Content-Length is missing, malformed, or incorrect.
func payloadLimitMiddleware(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Early Content-Length check to reject oversized requests without reading data.
		// This is more efficient than streaming data only to reject it later.
		if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
			if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil && length > maxBytes {
				http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
				return
			}
		}
		// Fallback: http.MaxBytesReader handles missing/incorrect Content-Length headers
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers to responses based on the provided WebConfig.
// It handles the Origin, Access-Control-Allow-Origin, Access-Control-Allow-Methods,
// Access-Control-Allow-Headers, Access-Control-Expose-Headers, Access-Control-Max-Age, and
// Access-Control-Allow-Credentials headers. It responds to preflight OPTIONS requests
// with HTTP 204 No Content.
func corsMiddleware(next http.Handler, webConfig WebConfig) http.Handler {
	// Pre-compute header values ONCE at middleware construction time (not per-request).
	// This avoids allocations from strings.Join on every HTTP request.
	allowedHeadersValue := resolveHeaderValue(webConfig.AllowedHeaders, DefaultAllowedHeaders)
	exposedHeadersValue := resolveHeaderValue(webConfig.ExposedHeaders, DefaultExposedHeaders)

	// Pre-compute max-age value (use default if not configured)
	maxAge := DefaultMaxAge
	if webConfig.MaxAge != nil {
		maxAge = *webConfig.MaxAge
	}
	maxAgeValue := strconv.Itoa(maxAge)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always set Vary: Origin when CORS is configured to prevent cache pollution.
		// This tells caches that the response varies based on the Origin header,
		// even when no CORS headers are added (e.g., when origin is not allowed).
		w.Header().Set("Vary", "Origin")

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if origin is allowed
		allowed := false
		for _, o := range webConfig.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			next.ServeHTTP(w, r)
			return
		}

		// Set CORS headers (using pre-computed values for zero allocation)
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeadersValue)
		w.Header().Set("Access-Control-Expose-Headers", exposedHeadersValue)

		if webConfig.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", maxAgeValue)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
