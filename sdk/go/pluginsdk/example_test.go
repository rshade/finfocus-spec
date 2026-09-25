// Package pluginsdk_test contains runnable examples for the pluginsdk package.
// These examples appear in godoc and are validated by `go test`.
//
//nolint:testableexamples // Most examples require a running server and cannot have Output comments
package pluginsdk_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ExampleClient_Close demonstrates proper resource cleanup for SDK-owned HTTP clients.
//
// When you create a client using NewConnectClient, NewGRPCClient, or NewGRPCWebClient,
// the SDK creates and owns the HTTP client. You should call Close() when done to release
// connection pool resources.
func ExampleClient_Close() {
	// Create a client with SDK-owned HTTP client
	client := pluginsdk.NewConnectClient("http://localhost:8080")

	// Use the client for requests
	ctx := context.Background()
	name, err := client.Name(ctx)
	if err != nil {
		// Handle error
		return
	}
	fmt.Println("Plugin:", name)

	// Close releases connection pool resources.
	// This is safe to call multiple times.
	client.Close()
}

// ExampleClient_Close_userProvided demonstrates HTTP client ownership when using
// a user-provided HTTPClient.
//
// When you provide your own HTTP client via ClientConfig.HTTPClient, you retain
// ownership and are responsible for its lifecycle. In this case, Client.Close()
// is a no-op.
func ExampleClient_Close_userProvided() {
	// When providing your own HTTP client, you manage its lifecycle
	httpClient := &http.Client{Timeout: 60 * time.Second}

	client := pluginsdk.NewClient(pluginsdk.ClientConfig{
		BaseURL:    "http://localhost:8080",
		Protocol:   pluginsdk.ProtocolConnect,
		HTTPClient: httpClient, // User-provided
	})

	// Use the client...
	ctx := context.Background()
	name, err := client.Name(ctx)
	if err != nil {
		// Handle error
		return
	}
	fmt.Println("Plugin:", name)

	// client.Close() is a no-op here - caller manages httpClient
	client.Close()

	// Caller is responsible for closing the HTTP client
	httpClient.CloseIdleConnections()
}

// ExampleClient_concurrent demonstrates thread-safe concurrent usage of Client.
//
// Client is safe for concurrent use from multiple goroutines. Create once and
// reuse across goroutines rather than creating a new client for each request.
func ExampleClient_concurrent() {
	// Create client once
	client := pluginsdk.NewConnectClient("http://localhost:8080")
	defer client.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Safe to use concurrently from multiple goroutines
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name, _ := client.Name(ctx)
			fmt.Printf("Goroutine %d: %s\n", id, name)
		}(i)
	}

	wg.Wait()
}

// ExampleHighThroughputClientConfig demonstrates configuration for high-throughput scenarios.
//
// Use HighThroughputClientConfig when making many concurrent requests to the same plugin.
// It configures connection pooling for better performance.
func ExampleHighThroughputClientConfig() {
	// Get high-throughput configuration with connection pooling
	cfg := pluginsdk.HighThroughputClientConfig("http://localhost:8080")

	// Create client with optimized settings
	client := pluginsdk.NewClient(cfg)
	defer client.Close()

	ctx := context.Background()

	// Make many requests - connections are reused from the pool
	for range 100 {
		_, _ = client.Name(ctx)
	}
}

// ExampleNewFocusRecordBuilder demonstrates creating FOCUS 1.2/1.3 compliant cost records.
//
// The FocusRecordBuilder provides a fluent API for constructing FinOps FOCUS
// cost records with all mandatory and optional fields.
func ExampleNewFocusRecordBuilder() {
	builder := pluginsdk.NewFocusRecordBuilder()

	// Set mandatory identity fields
	builder.WithIdentity("AWS", "123456789012", "Production Account")

	// Set billing period
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	builder.WithBillingPeriod(monthStart, monthEnd, "USD")

	// Set charge period (same as billing for monthly charges)
	builder.WithChargePeriod(monthStart, monthEnd)

	// Set charge details
	builder.WithChargeDetails(
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
	)

	// Set charge classification (required for FOCUS compliance)
	builder.WithChargeClassification(
		pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		"On-demand EC2 compute usage",
		pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
	)

	// Set usage quantity (required for USAGE charge category)
	builder.WithUsage(720, "hours") // 720 hours = ~1 month

	// Set service information
	builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")

	// Set financial amounts
	builder.WithFinancials(73.0, 80.0, 70.0, "USD", "INV-2025-001")
	builder.WithContractedCost(65.0) // New in FOCUS 1.2

	// Build the record
	record, err := builder.Build()
	if err != nil {
		fmt.Println("Validation error:", err)
		return
	}

	fmt.Println("Created FOCUS record for:", record.GetServiceName())
}

// ExampleFocusRecordBuilder_WithAllocation demonstrates FOCUS 1.3 split cost allocation.
//
// Use allocation methods when distributing shared infrastructure costs
// across multiple workloads or cost centers.
func ExampleFocusRecordBuilder_WithAllocation() {
	builder := pluginsdk.NewFocusRecordBuilder()

	// Basic identity and billing (required)
	builder.WithIdentity("AWS", "123456789012", "Shared Services")
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	builder.WithBillingPeriod(monthStart, monthEnd, "USD")
	builder.WithChargePeriod(monthStart, monthEnd)
	builder.WithChargeDetails(
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
	)
	builder.WithChargeClassification(
		pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
		"Shared infrastructure compute usage",
		pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
	)
	builder.WithUsage(720, "hours") // 720 hours = ~1 month
	builder.WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2")
	builder.WithFinancials(100.0, 100.0, 100.0, "USD", "")
	builder.WithContractedCost(100.0)

	// FOCUS 1.3: Allocate shared costs to specific workloads
	builder.WithAllocation("proportional-cpu", "Costs split by CPU utilization percentage")
	builder.WithAllocatedResource("workload-frontend-001", "Frontend Application")
	builder.WithAllocatedTags(map[string]string{
		"team":        "frontend",
		"environment": "production",
	})

	record, err := builder.Build()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Allocated to: %s (%s)\n",
		record.GetAllocatedResourceName(),
		record.GetAllocatedMethodId())
}

// ExampleResourceMatcher demonstrates resource filtering configuration.
//
// ResourceMatcher helps plugins declare which resources they support.
// Configure it during plugin initialization before calling Serve().
func ExampleResourceMatcher() {
	matcher := pluginsdk.NewResourceMatcher()

	// Add supported providers
	matcher.AddProvider("aws")
	matcher.AddProvider("azure")

	// Add supported resource types
	matcher.AddResourceType("aws:ec2/instance:Instance")
	matcher.AddResourceType("aws:rds/instance:Instance")
	matcher.AddResourceType("azure:compute/virtualMachine:VirtualMachine")

	// Check if a resource is supported (in plugin's Supports() method)
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "aws:ec2/instance:Instance",
	}

	if matcher.Supports(resource) {
		fmt.Println("Resource is supported")
	}

	// Output: Resource is supported
}

// clusterUsageSource is a usage-only plugin. BasePlugin supplies the Plugin
// cost methods, so GetStats is the only method the author writes.
type clusterUsageSource struct {
	*pluginsdk.BasePlugin
}

func (s *clusterUsageSource) GetStats(
	_ context.Context, _ *pbc.GetStatsRequest,
) (*pbc.GetStatsResponse, error) {
	return &pbc.GetStatsResponse{
		Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
		Rows: []*pbc.UsageRow{
			{
				Subject: map[string]string{
					pluginsdk.SubjectKind:      pluginsdk.KindWorkload,
					pluginsdk.SubjectNamespace: "payments",
					pluginsdk.SubjectPod:       "api-7d9f",
					pluginsdk.SubjectNode:      "ip-10-0-1-5",
				},
				Metric: pluginsdk.MetricCPURequest,
				Amount: 0.5,
				Unit:   pluginsdk.UnitCore,
			},
			{
				Subject: map[string]string{
					pluginsdk.SubjectKind: pluginsdk.KindNode,
					pluginsdk.SubjectNode: "ip-10-0-1-5",
				},
				Metric: pluginsdk.MetricCPUAllocatable,
				Amount: 1.93,
				Unit:   pluginsdk.UnitCore,
			},
		},
	}, nil
}

// Example_usageSource shows a usage-only plugin. It declares
// PLUGIN_CAPABILITY_USAGE_STATS explicitly so hosts do not mistake it for a
// pricing plugin. A real main would pass config to pluginsdk.Run.
func Example_usageSource() {
	plugin := &clusterUsageSource{BasePlugin: pluginsdk.NewBasePlugin("k8s-usage")}

	config := pluginsdk.ServeConfig{
		Plugin: plugin,
		PluginInfo: pluginsdk.NewPluginInfo("k8s-usage", "v1.0.0",
			pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS),
		),
	}
	fmt.Println(config.PluginInfo.Capabilities)

	resp, _ := plugin.GetStats(context.Background(), &pbc.GetStatsRequest{})
	for _, row := range resp.GetRows() {
		fmt.Printf("%s %s=%.2f %s\n",
			row.GetSubject()[pluginsdk.SubjectKind], row.GetMetric(), row.GetAmount(), row.GetUnit())
	}

	// Output:
	// [PLUGIN_CAPABILITY_USAGE_STATS]
	// workload cpu_request=0.50 core
	// node cpu_allocatable=1.93 core
}
