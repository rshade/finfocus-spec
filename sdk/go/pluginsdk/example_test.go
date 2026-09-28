// Package pluginsdk_test contains runnable examples for the pluginsdk package.
// These examples appear in godoc and are validated by `go test`.
//
//nolint:testableexamples // Most examples require a running server and cannot have Output comments
package pluginsdk_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ExampleWithCredentials attaches named credentials to one call context.
// The printed line shows the count, that the value round-trips, and that String redacts it.
func ExampleWithCredentials() {
	creds, err := pluginsdk.NewCredentials(map[string]string{"token": "example-value"})
	if err != nil {
		fmt.Println("error")
		return
	}
	ctx := pluginsdk.WithCredentials(context.Background(), creds)
	got, err := pluginsdk.ExtractCredentials(ctx)
	if err != nil {
		fmt.Println("error")
		return
	}
	value, ok := got.Get("token")
	fmt.Println(got.Len(), ok && value == "example-value", strings.Contains(got.String(), "example-value"))
	// Output:
	// 1 true false
}

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

// ExampleNewFocusRecordBuilder demonstrates creating FOCUS 1.2-1.4 compliant cost records.
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

// ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails demonstrates the
// FOCUS 1.4 Cost and Usage columns.
//
// FOCUS 1.4 removes ProviderName, so the record names its provider only through
// WithServiceProvider. WithInvoiceDetailID links the row to an invoice line, and
// WithCommitmentProgramEligibilityDetails lists the commitment programs the charge
// was eligible for. Build rejects eligibility details that are not a JSON object.
func ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails() {
	newBuilder := func() *pluginsdk.FocusRecordBuilder {
		monthStart := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, 0)
		return pluginsdk.NewFocusRecordBuilder().
			WithIdentity("", "123456789012", "Production Account"). // no ProviderName in FOCUS 1.4
			WithServiceProvider("AWS").
			WithBillingPeriod(monthStart, monthEnd, "USD").
			WithChargePeriod(monthStart, monthEnd).
			WithChargeDetails(
				pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
				pbc.FocusPricingCategory_FOCUS_PRICING_CATEGORY_STANDARD,
			).
			WithChargeClassification(
				pbc.FocusChargeClass_FOCUS_CHARGE_CLASS_REGULAR,
				"On-demand EC2 compute usage",
				pbc.FocusChargeFrequency_FOCUS_CHARGE_FREQUENCY_USAGE_BASED,
			).
			WithUsage(720, "hours").
			WithService(pbc.FocusServiceCategory_FOCUS_SERVICE_CATEGORY_COMPUTE, "Amazon EC2").
			WithFinancials(73.0, 80.0, 70.0, "USD", "INV-2026-09").
			WithContractedCost(73.0)
	}

	record, err := newBuilder().
		WithInvoiceDetailID("INV-2026-09-L3").
		WithCommitmentProgramEligibilityDetails(
			`{"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}`,
		).
		Build()
	fmt.Println("build error:", err)
	fmt.Println("invoice detail:", record.GetInvoiceDetailId())
	fmt.Println("eligibility:", record.GetCommitmentProgramEligibilityDetails())

	_, err = newBuilder().
		WithCommitmentProgramEligibilityDetails(`{"CommitmentPrograms":[`).
		Build()
	fmt.Println("truncated JSON rejected:",
		errors.Is(err, pluginsdk.ErrInvalidCommitmentProgramEligibilityDetails))

	// Output:
	// build error: <nil>
	// invoice detail: INV-2026-09-L3
	// eligibility: {"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}
	// truncated JSON rejected: true
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

// exampleAllocPolicy is an allocator's policy document. Every allocator policy
// has a top-level integer "version".
type exampleAllocPolicy struct {
	Version   int     `json:"version"`
	CPUWeight float64 `json:"cpu_weight"`
}

// idleOnlyAllocator is a deliberately simple allocator: it reports each priced
// node's whole cost as that node's idle row, and every other priced resource
// as a cluster row. It embeds BasePlugin for the required cost-source methods.
type idleOnlyAllocator struct {
	*pluginsdk.BasePlugin
}

func (a *idleOnlyAllocator) Allocate(_ context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	if err := pluginsdk.ValidateAllocateRequest(req); err != nil {
		return nil, err // already carries codes.InvalidArgument
	}
	policy := exampleAllocPolicy{Version: 1, CPUWeight: 0.5}
	if err := pluginsdk.DecodePolicy(req.GetPolicyJson(), &policy); err != nil {
		return nil, err
	}
	if policy.Version != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported policy version %d", policy.Version)
	}
	currency, err := pluginsdk.ResolveCurrency(req.GetPriced())
	if err != nil {
		return nil, err
	}

	var rows []*pbc.AllocationRow
	for _, entry := range req.GetPriced() {
		if !entry.GetPriced() {
			continue
		}
		cost := entry.GetCost()
		if entry.GetResource().GetTags()[pluginsdk.SubjectKind] != pluginsdk.KindNode {
			rows = append(rows, &pbc.AllocationRow{
				Subject:   map[string]string{pluginsdk.SubjectKind: pluginsdk.KindCluster},
				TotalCost: cost,
				Currency:  currency,
			})
			continue
		}
		cpu := cost * policy.CPUWeight
		rows = append(rows, &pbc.AllocationRow{
			Subject: map[string]string{
				pluginsdk.SubjectKind: pluginsdk.KindIdle,
				pluginsdk.SubjectNode: entry.GetResource().GetId(),
			},
			CpuCost:   cpu,
			MemCost:   cost - cpu,
			TotalCost: cost,
			Currency:  currency,
		})
	}

	effective, err := json.Marshal(policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode policy: %v", err)
	}
	digest := sha256.Sum256(effective)
	return &pbc.AllocateResponse{
		Rows:                rows,
		EffectivePolicyJson: effective,
		PolicyDigest:        hex.EncodeToString(digest[:]),
	}, nil
}

// ExampleAllocatorProvider shows an allocation-only plugin. It declares
// PLUGIN_CAPABILITY_ALLOCATION explicitly so hosts do not mistake it for a
// pricing plugin, and hosts verify its output with CheckConservation. A real
// main would pass config to pluginsdk.Run.
func ExampleAllocatorProvider() {
	plugin := &idleOnlyAllocator{BasePlugin: pluginsdk.NewBasePlugin("idle-only")}

	config := pluginsdk.ServeConfig{
		Plugin: plugin,
		PluginInfo: pluginsdk.NewPluginInfo("idle-only", "v1.0.0",
			pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ALLOCATION),
		),
	}
	fmt.Println(config.PluginInfo.Capabilities)

	req := &pbc.AllocateRequest{
		Priced: []*pbc.PricedResource{{
			Resource: &pbc.ResourceDescriptor{Id: "n1", Tags: map[string]string{"kind": "node"}},
			Cost:     10,
			Currency: "USD",
			Priced:   true,
		}},
		PolicyJson: []byte(`{"cpu_weight":0.25}`),
	}
	resp, _ := plugin.Allocate(context.Background(), req)
	for _, row := range resp.GetRows() {
		fmt.Printf("%s %s cpu=%.2f mem=%.2f total=%.2f %s\n", row.GetSubject()[pluginsdk.SubjectKind],
			row.GetSubject()[pluginsdk.SubjectNode], row.GetCpuCost(), row.GetMemCost(), row.GetTotalCost(),
			row.GetCurrency())
	}
	fmt.Println(pluginsdk.CheckConservation(req, resp, pluginsdk.DefaultConservationEpsilon))

	_, err := plugin.Allocate(context.Background(), &pbc.AllocateRequest{PolicyJson: []byte(`{"cpu":1}`)})
	fmt.Println(status.Code(err), err)

	// Output:
	// [PLUGIN_CAPABILITY_ALLOCATION]
	// __idle__ n1 cpu=2.50 mem=7.50 total=10.00 USD
	// <nil>
	// InvalidArgument invalid allocation policy: unknown field "cpu"
}

// ExampleDecodePolicy applies a partial policy document onto defaults: nested
// objects merge field by field, arrays replace the default wholesale, and
// unknown fields are rejected with their JSON path.
func ExampleDecodePolicy() {
	type nodeSplit struct {
		CPUWeight float64 `json:"cpu_weight"`
		MemWeight float64 `json:"mem_weight"`
	}
	type policy struct {
		Version    int       `json:"version"`
		NodeSplit  nodeSplit `json:"node_split"`
		Namespaces []string  `json:"namespaces"`
	}

	p := policy{
		Version:    1,
		NodeSplit:  nodeSplit{CPUWeight: 0.5, MemWeight: 0.5},
		Namespaces: []string{"default", "kube-system"},
	}
	err := pluginsdk.DecodePolicy([]byte(`{"node_split":{"cpu_weight":0.7},"namespaces":["payments"]}`), &p)
	fmt.Printf("%+v %v\n", p, err)

	err = pluginsdk.DecodePolicy([]byte(`{"node_split":{"cpu":1}}`), &p)
	fmt.Println(err)

	// Output:
	// {Version:1 NodeSplit:{CPUWeight:0.7 MemWeight:0.5} Namespaces:[payments]} <nil>
	// invalid allocation policy: unknown field "node_split.cpu"
}

// ExampleWithProjectedCostBreakdown reports an EC2 instance's compute and root
// volume as separate components of cost_per_month, then validates the response.
func ExampleWithProjectedCostBreakdown() {
	resp := pluginsdk.NewGetProjectedCostResponse(
		pluginsdk.WithProjectedCostDetails(0.0104, "USD", 8.392, "On-demand Linux + 8GB gp2 root"),
		pluginsdk.WithProjectedCostBreakdown(map[string]float64{
			"compute":     7.592,
			"root_volume": 0.80,
		}),
	)

	breakdown := resp.GetCostBreakdown()
	for _, name := range slices.Sorted(maps.Keys(breakdown)) {
		fmt.Printf("%s: %.3f %s\n", name, breakdown[name], resp.GetCurrency())
	}
	fmt.Println("valid:", pluginsdk.ValidateGetProjectedCostResponse(resp))

	resp.CostBreakdown["root_volume"] = 1.408
	err := pluginsdk.ValidateGetProjectedCostResponse(resp)
	fmt.Println(errors.Is(err, pluginsdk.ErrCostBreakdownSumMismatch), err)

	// Output:
	// compute: 7.592 USD
	// root_volume: 0.800 USD
	// valid: <nil>
	// true GetProjectedCostResponse: cost_breakdown does not sum to cost_per_month: sum 9, cost_per_month 8.392, tolerance 0.01
}
