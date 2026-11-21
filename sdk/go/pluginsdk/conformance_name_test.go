//nolint:testpackage // Uses the internal conformanceMockPlugin fixture.
package pluginsdk

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// infoProviderPlugin reports infoName through PluginInfoProvider and its own name through Name().
type infoProviderPlugin struct {
	conformanceMockPlugin

	infoName string
}

func (p *infoProviderPlugin) GetPluginInfo(
	_ context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{Name: p.infoName, Version: "v1.0.0", SpecVersion: SpecVersion}, nil
}

func countNameMismatches(result *ConformanceResult) int {
	var count int
	for _, category := range result.Categories {
		for _, r := range category.Results {
			if !r.Success && errors.Is(r.Error, plugintesting.ErrPluginNameMismatch) {
				count++
			}
		}
	}
	return count
}

func TestRunConformanceNameConsistency(t *testing.T) {
	tests := []struct {
		name string
		run  func() (*ConformanceResult, error)
		want int
	}{
		{
			name: "provider names agree",
			run: func() (*ConformanceResult, error) {
				return RunConformance(&infoProviderPlugin{conformanceMockPlugin{name: "same"}, "same"},
					ConformanceLevelBasic, WithExpectedName("same"))
			},
		},
		{
			name: "provider plugin info name differs from Name",
			run: func() (*ConformanceResult, error) {
				return RunConformance(&infoProviderPlugin{conformanceMockPlugin{name: "aws-public"}, "aws"},
					ConformanceLevelBasic, WithExpectedName("aws-public"))
			},
			want: 1,
		},
		{
			name: "plugin info drift is not checked without WithExpectedName",
			run: func() (*ConformanceResult, error) {
				return RunBasicConformance(&infoProviderPlugin{conformanceMockPlugin{name: "aws-public"}, "aws"})
			},
		},
		{
			name: "configured plugin info name differs from Name",
			run: func() (*ConformanceResult, error) {
				server := NewServerWithOptions(&conformanceMockPlugin{name: "aws-public"}, nil, nil,
					NewPluginInfo("aws", "v1.0.0"))
				return plugintesting.RunConformance(server, ConformanceLevelBasic,
					plugintesting.WithExpectedName("aws-public"))
			},
			want: 1,
		},
		{
			name: "plugin without plugin info is not compared",
			run: func() (*ConformanceResult, error) {
				return RunConformance(&conformanceMockPlugin{name: "no-info"}, ConformanceLevelBasic,
					WithExpectedName("no-info"))
			},
		},
		{
			name: "expected name differs from Name",
			run: func() (*ConformanceResult, error) {
				return RunConformance(&conformanceMockPlugin{name: "y"}, ConformanceLevelBasic, WithExpectedName("x"))
			},
			want: 1,
		},
		{
			name: "expected name matches",
			run: func() (*ConformanceResult, error) {
				return RunConformance(&conformanceMockPlugin{name: "x"}, ConformanceLevelBasic, WithExpectedName("x"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.run()
			require.NoError(t, err)
			assert.Equal(t, tt.want, countNameMismatches(result))
		})
	}
}
