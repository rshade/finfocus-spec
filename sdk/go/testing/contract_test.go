package testing_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestValidateResourceDescriptor(t *testing.T) {
	tests := []struct {
		name        string
		resource    *pbc.ResourceDescriptor
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil resource",
			resource:    nil,
			wantErr:     true,
			errContains: "nil",
		},
		{
			name: "empty provider",
			resource: &pbc.ResourceDescriptor{
				Provider:     "",
				ResourceType: "ec2",
			},
			wantErr:     true,
			errContains: "provider",
		},
		{
			name: "invalid provider",
			resource: &pbc.ResourceDescriptor{
				Provider:     "invalid",
				ResourceType: "ec2",
			},
			wantErr:     true,
			errContains: "provider",
		},
		{
			name: "empty resource_type",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "",
			},
			wantErr:     true,
			errContains: "resource_type",
		},
		{
			name: "valid aws resource",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Sku:          "t3.micro",
				Region:       "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid azure resource",
			resource: &pbc.ResourceDescriptor{
				Provider:     "azure",
				ResourceType: "vm",
				Sku:          "Standard_B1s",
				Region:       "eastus",
			},
			wantErr: false,
		},
		{
			name: "valid gcp resource",
			resource: &pbc.ResourceDescriptor{
				Provider:     "gcp",
				ResourceType: "compute-instance",
				Sku:          "e2-micro",
				Region:       "us-central1",
			},
			wantErr: false,
		},
		{
			name: "valid kubernetes resource",
			resource: &pbc.ResourceDescriptor{
				Provider:     "kubernetes",
				ResourceType: "k8s-namespace",
				Tags: map[string]string{
					"namespace": "default",
				},
			},
			wantErr: false,
		},
		{
			name: "valid custom resource",
			resource: &pbc.ResourceDescriptor{
				Provider:     "custom",
				ResourceType: "my-custom-resource",
			},
			wantErr: false,
		},
		{
			name: "valid with tags",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags: map[string]string{
					"env":  "production",
					"team": "platform",
				},
			},
			wantErr: false,
		},
		{
			name: "too many tags",
			resource: &pbc.ResourceDescriptor{
				Provider:     "aws",
				ResourceType: "ec2",
				Tags:         generateTags(plugintesting.MaxTagCount + 1),
			},
			wantErr:     true,
			errContains: "tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateResourceDescriptor(tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceDescriptor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateTimeRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		start       *timestamppb.Timestamp
		end         *timestamppb.Timestamp
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil start",
			start:       nil,
			end:         timestamppb.New(now),
			wantErr:     true,
			errContains: "start",
		},
		{
			name:        "nil end",
			start:       timestamppb.New(now.Add(-1 * time.Hour)),
			end:         nil,
			wantErr:     true,
			errContains: "end",
		},
		{
			name:        "end before start",
			start:       timestamppb.New(now),
			end:         timestamppb.New(now.Add(-1 * time.Hour)),
			wantErr:     true,
			errContains: "time_range",
		},
		{
			name:        "same start and end",
			start:       timestamppb.New(now),
			end:         timestamppb.New(now),
			wantErr:     true,
			errContains: "time_range",
		},
		{
			name:        "range too short",
			start:       timestamppb.New(now.Add(-30 * time.Minute)),
			end:         timestamppb.New(now),
			wantErr:     true,
			errContains: "short",
		},
		{
			name:        "range too long",
			start:       timestamppb.New(now.Add(-400 * 24 * time.Hour)),
			end:         timestamppb.New(now),
			wantErr:     true,
			errContains: "maximum",
		},
		{
			name:    "valid 24 hour range",
			start:   timestamppb.New(now.Add(-24 * time.Hour)),
			end:     timestamppb.New(now),
			wantErr: false,
		},
		{
			name:    "valid 7 day range",
			start:   timestamppb.New(now.Add(-7 * 24 * time.Hour)),
			end:     timestamppb.New(now),
			wantErr: false,
		},
		{
			name:    "valid 30 day range",
			start:   timestamppb.New(now.Add(-30 * 24 * time.Hour)),
			end:     timestamppb.New(now),
			wantErr: false,
		},
		{
			name:    "valid maximum range (365 days)",
			start:   timestamppb.New(now.Add(-365 * 24 * time.Hour)),
			end:     timestamppb.New(now),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateTimeRange(tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeRange() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateGetActualCostRequest(t *testing.T) {
	now := time.Now()
	validStart := timestamppb.New(now.Add(-24 * time.Hour))
	validEnd := timestamppb.New(now)

	tests := []struct {
		name        string
		req         *pbc.GetActualCostRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil request",
			req:         nil,
			wantErr:     true,
			errContains: "nil",
		},
		{
			name: "empty resource_id",
			req: &pbc.GetActualCostRequest{
				ResourceId: "",
				Start:      validStart,
				End:        validEnd,
			},
			wantErr:     true,
			errContains: "resource_id",
		},
		{
			name: "resource_id too long",
			req: &pbc.GetActualCostRequest{
				ResourceId: strings.Repeat("a", plugintesting.MaxResourceIDLength+1),
				Start:      validStart,
				End:        validEnd,
			},
			wantErr:     true,
			errContains: "resource_id",
		},
		{
			name: "valid request",
			req: &pbc.GetActualCostRequest{
				ResourceId: "i-abc123",
				Start:      validStart,
				End:        validEnd,
			},
			wantErr: false,
		},
		{
			name: "valid request with tags",
			req: &pbc.GetActualCostRequest{
				ResourceId: "i-abc123",
				Start:      validStart,
				End:        validEnd,
				Tags: map[string]string{
					"env": "prod",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateGetActualCostRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGetActualCostRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateEstimateCostRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *pbc.EstimateCostRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil request",
			req:         nil,
			wantErr:     true,
			errContains: "nil",
		},
		{
			name: "empty resource_type",
			req: &pbc.EstimateCostRequest{
				ResourceType: "",
			},
			wantErr:     true,
			errContains: "resource_type",
		},
		{
			name: "valid simple resource_type",
			req: &pbc.EstimateCostRequest{
				ResourceType: "ec2",
			},
			wantErr: false,
		},
		{
			name: "valid Pulumi resource_type format",
			req: &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
			},
			wantErr: false,
		},
		{
			name: "valid with attributes",
			req: &pbc.EstimateCostRequest{
				ResourceType: "aws:ec2/instance:Instance",
				Attributes:   nil, // attributes are optional
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateEstimateCostRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEstimateCostRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateGetRecommendationsRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *pbc.GetRecommendationsRequest
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil request",
			req:         nil,
			wantErr:     true,
			errContains: "nil",
		},
		{
			name:    "empty request is valid",
			req:     &pbc.GetRecommendationsRequest{},
			wantErr: false,
		},
		{
			name: "page_size too large",
			req: &pbc.GetRecommendationsRequest{
				PageSize: plugintesting.MaxPageSize + 1,
			},
			wantErr:     true,
			errContains: "page_size",
		},
		{
			name: "valid page_size",
			req: &pbc.GetRecommendationsRequest{
				PageSize: 100,
			},
			wantErr: false,
		},
		{
			name: "invalid projection_period",
			req: &pbc.GetRecommendationsRequest{
				ProjectionPeriod: "invalid",
			},
			wantErr:     true,
			errContains: "projection_period",
		},
		{
			name: "valid projection_period monthly",
			req: &pbc.GetRecommendationsRequest{
				ProjectionPeriod: "monthly",
			},
			wantErr: false,
		},
		{
			name: "valid projection_period annual",
			req: &pbc.GetRecommendationsRequest{
				ProjectionPeriod: "annual",
			},
			wantErr: false,
		},
		{
			name: "valid with filter",
			req: &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Provider: "aws",
					Region:   "us-east-1",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid filter provider",
			req: &pbc.GetRecommendationsRequest{
				Filter: &pbc.RecommendationFilter{
					Provider: "invalid-provider",
				},
			},
			wantErr:     true,
			errContains: "provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateGetRecommendationsRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGetRecommendationsRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name        string
		tags        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil tags",
			tags:    nil,
			wantErr: false,
		},
		{
			name:    "empty tags",
			tags:    map[string]string{},
			wantErr: false,
		},
		{
			name: "valid tags",
			tags: map[string]string{
				"env":  "production",
				"team": "platform",
			},
			wantErr: false,
		},
		{
			name:        "too many tags",
			tags:        generateTags(plugintesting.MaxTagCount + 1),
			wantErr:     true,
			errContains: "tag",
		},
		{
			name: "key too long",
			tags: map[string]string{
				strings.Repeat("k", plugintesting.MaxTagKeyLength+1): "value",
			},
			wantErr:     true,
			errContains: "key",
		},
		{
			name: "value too long",
			tags: map[string]string{
				"key": strings.Repeat("v", plugintesting.MaxTagValueLength+1),
			},
			wantErr:     true,
			errContains: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateTags(tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestValidateTargetResources(t *testing.T) {
	tests := []struct {
		name        string
		targets     []*pbc.ResourceDescriptor
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil targets (valid - preserves existing behavior)",
			targets: nil,
			wantErr: false,
		},
		{
			name:    "empty targets (valid - preserves existing behavior)",
			targets: []*pbc.ResourceDescriptor{},
			wantErr: false,
		},
		{
			name: "single valid target",
			targets: []*pbc.ResourceDescriptor{
				{
					Provider:     "aws",
					ResourceType: "ec2",
					Sku:          "t3.medium",
					Region:       "us-east-1",
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid targets",
			targets: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2", Sku: "t3.medium", Region: "us-east-1"},
				{Provider: "azure", ResourceType: "vm", Sku: "Standard_B1s", Region: "eastus"},
				{Provider: "gcp", ResourceType: "compute-instance", Sku: "e2-micro", Region: "us-central1"},
			},
			wantErr: false,
		},
		{
			name: "valid targets with tags",
			targets: []*pbc.ResourceDescriptor{
				{
					Provider:     "aws",
					ResourceType: "ec2",
					Tags:         map[string]string{"env": "prod", "team": "platform"},
				},
			},
			wantErr: false,
		},
		{
			name:        "exceeds maximum limit",
			targets:     generateResourceDescriptors(plugintesting.MaxTargetResources + 1),
			wantErr:     true,
			errContains: "exceeds maximum",
		},
		{
			name:    "at maximum limit is valid",
			targets: generateResourceDescriptors(plugintesting.MaxTargetResources),
			wantErr: false,
		},
		{
			name: "invalid resource in list - nil element",
			targets: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				nil,
			},
			wantErr:     true,
			errContains: "target_resources[1]",
		},
		{
			name: "invalid resource in list - empty provider",
			targets: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "", ResourceType: "ec2"},
			},
			wantErr:     true,
			errContains: "target_resources[1]",
		},
		{
			name: "invalid resource in list - invalid provider",
			targets: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "invalid-provider", ResourceType: "ec2"},
			},
			wantErr:     true,
			errContains: "target_resources[1]",
		},
		{
			name: "invalid resource in list - empty resource_type",
			targets: []*pbc.ResourceDescriptor{
				{Provider: "aws", ResourceType: "ec2"},
				{Provider: "aws", ResourceType: ""},
			},
			wantErr:     true,
			errContains: "target_resources[1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateTargetResources(tt.targets)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTargetResources() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("error should contain %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}

// generateResourceDescriptors creates a slice of valid ResourceDescriptor for testing.
func generateResourceDescriptors(count int) []*pbc.ResourceDescriptor {
	providers := []string{"aws", "azure", "gcp", "kubernetes", "custom"}
	resources := make([]*pbc.ResourceDescriptor, count)
	for i := range count {
		resources[i] = &pbc.ResourceDescriptor{
			Provider:     providers[i%len(providers)],
			ResourceType: fmt.Sprintf("resource-%d", i),
		}
	}
	return resources
}

func TestContractTestSuite(t *testing.T) {
	results := plugintesting.RunStandardContractTests()

	passed := 0
	failed := 0

	for _, result := range results {
		if result.Passed {
			passed++
			t.Logf("PASS: %s", result.Name)
		} else {
			failed++
			t.Errorf("FAIL: %s - %s", result.Name, result.Details)
		}
	}

	t.Logf("Contract tests: %d passed, %d failed", passed, failed)

	if failed > 0 {
		t.Errorf("Contract test suite had %d failures", failed)
	}
}

func TestContractError(t *testing.T) {
	err := plugintesting.NewContractError("test_field", "test_value", plugintesting.ErrEmptyProvider)

	if err.Field != "test_field" {
		t.Errorf("expected field 'test_field', got %q", err.Field)
	}
	if err.Value != "test_value" {
		t.Errorf("expected value 'test_value', got %v", err.Value)
	}
	if !errors.Is(err.Unwrap(), plugintesting.ErrEmptyProvider) {
		t.Errorf("expected unwrapped error to be ErrEmptyProvider")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "test_field") {
		t.Errorf("error string should contain field name")
	}
	if !strings.Contains(errStr, "test_value") {
		t.Errorf("error string should contain value")
	}
}

// generateTags creates a map with the specified number of tags.
func generateTags(count int) map[string]string {
	tags := make(map[string]string, count)
	for i := range count {
		tags[fmt.Sprintf("key_%d", i)] = "value"
	}
	return tags
}

// =============================================================================
// Contract Validation Benchmarks
// =============================================================================

// BenchmarkValidateResourceDescriptor benchmarks resource descriptor validation.
func BenchmarkValidateResourceDescriptor(b *testing.B) {
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.micro",
		Region:       "us-east-1",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateResourceDescriptor(resource)
	}
}

// BenchmarkValidateResourceDescriptor_WithTags benchmarks validation with tags.
func BenchmarkValidateResourceDescriptor_WithTags(b *testing.B) {
	resource := &pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.micro",
		Region:       "us-east-1",
		Tags: map[string]string{
			"env":  "production",
			"team": "platform",
			"app":  "web-service",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateResourceDescriptor(resource)
	}
}

// BenchmarkValidateTimeRange benchmarks time range validation.
func BenchmarkValidateTimeRange(b *testing.B) {
	now := time.Now()
	start := timestamppb.New(now.Add(-24 * time.Hour))
	end := timestamppb.New(now)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateTimeRange(start, end)
	}
}

// BenchmarkValidateTags benchmarks tag validation with varying counts.
func BenchmarkValidateTags(b *testing.B) {
	testCases := []struct {
		name     string
		tagCount int
	}{
		{"0Tags", 0},
		{"5Tags", 5},
		{"25Tags", 25},
		{"50Tags", 50},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			tags := generateTags(tc.tagCount)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = plugintesting.ValidateTags(tags)
			}
		})
	}
}

// BenchmarkValidateGetActualCostRequest benchmarks actual cost request validation.
func BenchmarkValidateGetActualCostRequest(b *testing.B) {
	now := time.Now()
	req := &pbc.GetActualCostRequest{
		ResourceId: "i-abc123",
		Start:      timestamppb.New(now.Add(-24 * time.Hour)),
		End:        timestamppb.New(now),
		Tags: map[string]string{
			"env": "production",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateGetActualCostRequest(req)
	}
}

// BenchmarkValidateEstimateCostRequest benchmarks estimate cost request validation.
func BenchmarkValidateEstimateCostRequest(b *testing.B) {
	req := &pbc.EstimateCostRequest{
		ResourceType: "aws:ec2/instance:Instance",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateEstimateCostRequest(req)
	}
}

// BenchmarkValidateGetRecommendationsRequest benchmarks recommendations request validation.
func BenchmarkValidateGetRecommendationsRequest(b *testing.B) {
	req := &pbc.GetRecommendationsRequest{
		PageSize:         100,
		ProjectionPeriod: "monthly",
		Filter: &pbc.RecommendationFilter{
			Provider: "aws",
			Region:   "us-east-1",
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateGetRecommendationsRequest(req)
	}
}

// BenchmarkRunStandardContractTests benchmarks running the full contract test suite.
func BenchmarkRunStandardContractTests(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.RunStandardContractTests()
	}
}

// BenchmarkValidateSupportsRequest benchmarks support request validation.
func BenchmarkValidateSupportsRequest(b *testing.B) {
	req := &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = plugintesting.ValidateSupportsRequest(req)
	}
}

func TestValidateGetStatsRequest(t *testing.T) {
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	ts := timestamppb.New

	tests := []struct {
		name      string
		req       *pbc.GetStatsRequest
		wantErr   error
		wantField string
	}{
		{name: "nil request", req: nil, wantErr: plugintesting.ErrNilRequest},
		{
			name:      "only start set",
			req:       &pbc.GetStatsRequest{Start: ts(base)},
			wantErr:   plugintesting.ErrNilEndTime,
			wantField: "end",
		},
		{
			name:      "only end set",
			req:       &pbc.GetStatsRequest{End: ts(base)},
			wantErr:   plugintesting.ErrNilStartTime,
			wantField: "start",
		},
		{
			name:    "start after end",
			req:     &pbc.GetStatsRequest{Start: ts(base.Add(time.Hour)), End: ts(base)},
			wantErr: plugintesting.ErrInvertedStatsWindow,
		},
		{name: "run-rate request", req: &pbc.GetStatsRequest{}},
		{name: "start before end", req: &pbc.GetStatsRequest{Start: ts(base), End: ts(base.Add(time.Hour))}},
		{name: "start equals end", req: &pbc.GetStatsRequest{Start: ts(base), End: ts(base)}},
		{
			name: "five-minute window",
			req:  &pbc.GetStatsRequest{Start: ts(base), End: ts(base.Add(5 * time.Minute))},
		},
		{name: "empty scope", req: &pbc.GetStatsRequest{Scope: ""}},
		{name: "unknown metric", req: &pbc.GetStatsRequest{Metrics: []string{"gpu_seconds"}}},
		{
			name: "namespace and label selector",
			req: &pbc.GetStatsRequest{
				Selector: map[string]string{"namespace": "payments", "app": "api"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateGetStatsRequest(tt.req)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateGetStatsRequest() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateGetStatsRequest() = %v, want %v", err, tt.wantErr)
			}
			if tt.wantField == "" {
				return
			}
			var contractErr *plugintesting.ContractError
			if !errors.As(err, &contractErr) {
				t.Fatalf("error %v is not a *ContractError", err)
			}
			if contractErr.Field != tt.wantField {
				t.Errorf("ContractError.Field = %q, want %q", contractErr.Field, tt.wantField)
			}
		})
	}
}
