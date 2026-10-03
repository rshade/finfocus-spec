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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// descriptorCapturingPlugin records the ResourceDescriptor each RPC receives.
type descriptorCapturingPlugin struct {
	*plugintesting.MockPlugin

	got *pbc.ResourceDescriptor
}

func (p *descriptorCapturingPlugin) Supports(
	_ context.Context, req *pbc.SupportsRequest,
) (*pbc.SupportsResponse, error) {
	p.got = req.GetResource()
	return &pbc.SupportsResponse{}, nil
}

func (p *descriptorCapturingPlugin) GetProjectedCost(
	_ context.Context, req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	p.got = req.GetResource()
	return &pbc.GetProjectedCostResponse{}, nil
}

func (p *descriptorCapturingPlugin) GetPricingSpec(
	_ context.Context, req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	p.got = req.GetResource()
	return &pbc.GetPricingSpecResponse{}, nil
}

func (p *descriptorCapturingPlugin) BatchCost(
	_ context.Context, req *pbc.BatchCostRequest,
) (*pbc.BatchCostResponse, error) {
	p.got = req.GetResources()[0]
	return &pbc.BatchCostResponse{}, nil
}

func (p *descriptorCapturingPlugin) GetRecommendations(
	_ context.Context, req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	p.got = req.GetTargetResources()[0]
	return &pbc.GetRecommendationsResponse{}, nil
}

// cronJobAttributes nests a container CPU request ten segments deep, the depth
// of a Kubernetes CronJob: spec.jobTemplate.spec.template.spec.containers.0.resources.requests.cpu.
func cronJobAttributes(t *testing.T) *structpb.Struct {
	t.Helper()
	attrs, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"schedule": "*/5 * * * *",
			"jobTemplate": map[string]any{
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "worker",
									"resources": map[string]any{
										"requests": map[string]any{"cpu": "250m", "memory": "128Mi"},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	return attrs
}

// TestResourceDescriptorAttributesDelivered checks that nested attributes reach the
// plugin intact on every RPC that carries a ResourceDescriptor (SC-001).
func TestResourceDescriptorAttributesDelivered(t *testing.T) {
	plugin := &descriptorCapturingPlugin{MockPlugin: plugintesting.NewMockPlugin()}
	harness := plugintesting.NewTestHarness(plugin)
	harness.Start(t)
	defer harness.Stop()

	sent := &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:batch/v1:CronJob",
		Tags:         map[string]string{"app": "worker"},
		Attributes:   cronJobAttributes(t),
	}
	client := harness.Client()
	ctx := context.Background()

	calls := []struct {
		name string
		call func() error
	}{
		{"Supports", func() error {
			_, err := client.Supports(ctx, &pbc.SupportsRequest{Resource: sent})
			return err
		}},
		{"GetProjectedCost", func() error {
			_, err := client.GetProjectedCost(ctx, &pbc.GetProjectedCostRequest{Resource: sent})
			return err
		}},
		{"GetPricingSpec", func() error {
			_, err := client.GetPricingSpec(ctx, &pbc.GetPricingSpecRequest{Resource: sent})
			return err
		}},
		{"BatchCost", func() error {
			_, err := client.BatchCost(ctx, &pbc.BatchCostRequest{Resources: []*pbc.ResourceDescriptor{sent}})
			return err
		}},
		{"GetRecommendations", func() error {
			_, err := client.GetRecommendations(ctx, &pbc.GetRecommendationsRequest{
				TargetResources: []*pbc.ResourceDescriptor{sent},
			})
			return err
		}},
	}

	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			plugin.got = nil
			require.NoError(t, tc.call())
			require.NotNil(t, plugin.got)
			require.Empty(t, cmp.Diff(sent, plugin.got, protocmp.Transform()), "descriptor changed in transit")

			cpu := plugin.got.GetAttributes().GetFields()["spec"].GetStructValue().
				GetFields()["jobTemplate"].GetStructValue().
				GetFields()["spec"].GetStructValue().
				GetFields()["template"].GetStructValue().
				GetFields()["spec"].GetStructValue().
				GetFields()["containers"].GetListValue().GetValues()[0].GetStructValue().
				GetFields()["resources"].GetStructValue().
				GetFields()["requests"].GetStructValue().
				GetFields()["cpu"].GetStringValue()
			require.Equal(t, "250m", cpu)
		})
	}
}
