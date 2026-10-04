// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testing_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const customProvider = "custom"

// strictCustomPlugin prices only the custom provider. Like a real plugin, it answers
// InvalidArgument for an invalid descriptor, for any other provider, and for an
// actual-cost request that carries no resource descriptor, which the plain MockPlugin
// never does.
type strictCustomPlugin struct {
	*plugintesting.MockPlugin
}

func newStrictCustomPlugin() strictCustomPlugin {
	mock := plugintesting.NewMockPlugin()
	mock.SupportedProviders = []string{customProvider}
	return strictCustomPlugin{MockPlugin: mock}
}

func rejectOtherProviders(resource *pbc.ResourceDescriptor) error {
	if resource == nil {
		return nil
	}
	if err := plugintesting.ValidateResourceDescriptor(resource); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if resource.GetProvider() != customProvider {
		return status.Errorf(codes.InvalidArgument, "unsupported provider: %s", resource.GetProvider())
	}
	return nil
}

func (p strictCustomPlugin) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if err := rejectOtherProviders(req.GetResource()); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetProjectedCost(ctx, req)
}

func (p strictCustomPlugin) GetPricingSpec(
	_ context.Context,
	req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	resource := req.GetResource()
	if err := rejectOtherProviders(resource); err != nil {
		return nil, err
	}
	return &pbc.GetPricingSpecResponse{
		Spec: &pbc.PricingSpec{
			Provider:     resource.GetProvider(),
			ResourceType: resource.GetResourceType(),
			Sku:          resource.GetSku(),
			Region:       resource.GetRegion(),
			BillingMode:  "per_hour",
			RatePerUnit:  0.10,
			Currency:     "USD",
			Unit:         "hour",
			Source:       p.PluginName,
		},
	}, nil
}

func (p strictCustomPlugin) GetActualCost(
	ctx context.Context,
	req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	if req.GetResource() == nil {
		return nil, status.Error(codes.InvalidArgument, "resource is required to price actual cost")
	}
	if err := rejectOtherProviders(req.GetResource()); err != nil {
		return nil, err
	}
	return p.MockPlugin.GetActualCost(ctx, req)
}

func customSample() *pbc.ResourceDescriptor {
	return plugintesting.CreateResourceDescriptor(customProvider, "instance", "standard", "region-1")
}

// failedChecks lists the failing checks of a result, for assertion messages.
func failedChecks(result *plugintesting.ConformanceResult) []string {
	var failed []string
	for _, category := range result.Categories {
		for _, r := range category.Results {
			if !r.Success {
				failed = append(failed, fmt.Sprintf("%s/%s: %v", category.Name, r.Method, r.Error))
			}
		}
	}
	return failed
}

func TestSampleResourceAllLevels(t *testing.T) {
	levels := []plugintesting.ConformanceLevel{
		plugintesting.ConformanceLevelBasic,
		plugintesting.ConformanceLevelStandard,
		plugintesting.ConformanceLevelAdvanced,
	}
	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			result, err := plugintesting.RunConformance(
				newStrictCustomPlugin(), level, plugintesting.WithSampleResource(customSample()))
			require.NoError(t, err)
			assert.Zero(t, result.Summary.Failed, "failed checks: %v", failedChecks(result))
			assert.Positive(t, result.Summary.Passed)
		})
	}
}

func TestSampleResourceDefaultStillAWS(t *testing.T) {
	result, err := plugintesting.RunBasicConformance(newStrictCustomPlugin())
	require.NoError(t, err)
	assert.Positive(t, result.Summary.Failed,
		"the default sample resource must stay aws, which a custom-only plugin rejects")
}

func TestSampleResourceInvalid(t *testing.T) {
	tests := []struct {
		name      string
		level     plugintesting.ConformanceLevel
		resource  *pbc.ResourceDescriptor
		wantField string
	}{
		{
			name:      "empty provider",
			level:     plugintesting.ConformanceLevelBasic,
			resource:  &pbc.ResourceDescriptor{ResourceType: "instance"},
			wantField: "provider",
		},
		{
			name:      "unknown provider",
			level:     plugintesting.ConformanceLevelBasic,
			resource:  &pbc.ResourceDescriptor{Provider: "not-a-provider", ResourceType: "instance"},
			wantField: "provider",
		},
		{
			name:      "empty resource type",
			level:     plugintesting.ConformanceLevelBasic,
			resource:  &pbc.ResourceDescriptor{Provider: customProvider},
			wantField: "resource_type",
		},
		{
			name:      "unknown level",
			level:     plugintesting.ConformanceLevel(99),
			resource:  customSample(),
			wantField: "level",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := plugintesting.RunConformance(
				newStrictCustomPlugin(), tt.level, plugintesting.WithSampleResource(tt.resource))
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.wantField)
		})
	}
}

func TestHarnessSampleResourceCopies(t *testing.T) {
	t.Run("default harness", func(t *testing.T) {
		harness := plugintesting.NewTestHarness(plugintesting.NewMockPlugin())
		defer harness.Stop()
		got := harness.SampleResource()
		assert.Equal(t, "aws", got.GetProvider())

		got.Provider = customProvider
		assert.Equal(t, "aws", harness.SampleResource().GetProvider())
	})

	t.Run("configured suite", func(t *testing.T) {
		sample := customSample()
		option := plugintesting.WithSampleResource(sample)
		sample.ResourceType = "changed-before-apply"

		config := plugintesting.DefaultSuiteConfig()
		option(&config)
		sample.Provider = "aws"

		var seen, types []string
		suite := plugintesting.NewConformanceSuiteWithConfig(config)
		suite.AddTest(plugintesting.ConformanceSuiteTest{
			Name:     "SampleResource_Copy",
			Category: plugintesting.CategoryRPCCorrectness,
			MinLevel: plugintesting.ConformanceLevelBasic,
			TestFunc: func(h *plugintesting.TestHarness) plugintesting.TestResult {
				first := h.SampleResource()
				seen = append(seen, first.GetProvider())
				types = append(types, first.GetResourceType())
				first.Provider = "aws"
				seen = append(seen, h.SampleResource().GetProvider())
				return plugintesting.TestResult{Success: true}
			},
		})
		_, err := suite.Run(plugintesting.NewMockPlugin())
		require.NoError(t, err)
		assert.Equal(t, []string{customProvider, customProvider}, seen)
		assert.Equal(t, []string{"instance"}, types)
	})
}

func TestDefaultSampleResourceValid(t *testing.T) {
	sample := plugintesting.DefaultSampleResource()
	require.NoError(t, plugintesting.ValidateResourceDescriptor(sample))
	assert.Equal(t, "aws", sample.GetProvider())
	assert.Equal(t, "ec2", sample.GetResourceType())
	assert.Equal(t, "t3.micro", sample.GetSku())
	assert.Equal(t, "us-east-1", sample.GetRegion())

	sample.Provider = customProvider
	assert.Equal(t, "aws", plugintesting.DefaultSampleResource().GetProvider())
}

// TestRunnersStoreAsFuncValues guards source compatibility: the level runners keep
// their signatures, so code that stores them in typed function values still compiles.
func TestRunnersStoreAsFuncValues(t *testing.T) {
	runners := []func(pbc.CostSourceServiceServer) (*plugintesting.ConformanceResult, error){
		plugintesting.RunBasicConformance,
		plugintesting.RunStandardConformance,
		plugintesting.RunAdvancedConformance,
	}
	assert.Len(t, runners, 3)
}

func TestActualCostChecksSendResource(t *testing.T) {
	result, err := plugintesting.RunConformance(
		newStrictCustomPlugin(),
		plugintesting.ConformanceLevelStandard,
		plugintesting.WithSampleResource(customSample()),
	)
	require.NoError(t, err)

	var actual []plugintesting.TestResult
	for _, category := range result.Categories {
		for _, r := range category.Results {
			if r.Method == plugintesting.MethodGetActualCost {
				actual = append(actual, r)
			}
		}
	}
	require.Len(t, actual, 4, "invalid time range, plain, billing-account, and with-resource checks")
	for _, r := range actual {
		assert.True(t, r.Success, "%s: %v", r.Details, r.Error)
	}
}

func TestSampleResourceAtTagLimit(t *testing.T) {
	sample := customSample()
	sample.Tags = map[string]string{}
	for i := range plugintesting.MaxTagCount {
		sample.Tags[fmt.Sprintf("key-%02d", i)] = "value"
	}

	result, err := plugintesting.RunConformance(
		newStrictCustomPlugin(), plugintesting.ConformanceLevelStandard, plugintesting.WithSampleResource(sample))
	require.NoError(t, err)
	assert.Zero(t, result.Summary.Failed, "failed checks: %v", failedChecks(result))
}

func TestRunConformanceOptionCannotRaiseLevel(t *testing.T) {
	raise := func(c *plugintesting.SuiteConfig) { c.TargetLevel = plugintesting.ConformanceLevelAdvanced }

	result, err := plugintesting.RunConformance(newStrictCustomPlugin(), plugintesting.ConformanceLevelBasic,
		plugintesting.WithSampleResource(customSample()), raise)
	require.NoError(t, err)
	assert.Equal(t, plugintesting.ConformanceLevelBasic, result.LevelAchieved)
	assert.NotContains(t, result.Categories, plugintesting.CategoryPerformance)
	assert.NotContains(t, result.Categories, plugintesting.CategoryConcurrency)
}
