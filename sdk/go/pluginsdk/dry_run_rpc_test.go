//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// dryRunRPCPlugin implements DryRunHandler with a configurable result.
type dryRunRPCPlugin struct {
	mockPlugin

	resp *pbc.DryRunResponse
	err  error
	got  *pbc.DryRunRequest
}

func (p *dryRunRPCPlugin) HandleDryRun(
	_ context.Context,
	req *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	p.got = req
	return p.resp, p.err
}

func TestServer_DryRun(t *testing.T) {
	resource := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "aws:ec2/instance:Instance"}
	supported := NewDryRunResponse(WithResourceTypeSupported(true))

	tests := []struct {
		name     string
		plugin   Plugin
		req      *pbc.DryRunRequest
		wantCode codes.Code
		want     *pbc.DryRunResponse
	}{
		{
			name:     "delegates to DryRunHandler",
			plugin:   &dryRunRPCPlugin{resp: supported},
			req:      &pbc.DryRunRequest{Resource: resource},
			wantCode: codes.OK,
			want:     supported,
		},
		{
			name:     "plugin without DryRunHandler is Unimplemented",
			plugin:   &mockPlugin{name: "p"},
			req:      &pbc.DryRunRequest{Resource: resource},
			wantCode: codes.Unimplemented,
		},
		{
			name:     "nil resource is InvalidArgument",
			plugin:   &dryRunRPCPlugin{resp: supported},
			req:      &pbc.DryRunRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "handler error is Internal",
			plugin:   &dryRunRPCPlugin{err: errors.New("config unreadable")},
			req:      &pbc.DryRunRequest{Resource: resource},
			wantCode: codes.Internal,
		},
		{
			name:     "nil handler response is Internal",
			plugin:   &dryRunRPCPlugin{},
			req:      &pbc.DryRunRequest{Resource: resource},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewServer(tt.plugin).DryRun(context.Background(), tt.req)

			require.Equal(t, tt.wantCode, status.Code(err), "error: %v", err)
			if tt.want != nil {
				assert.Same(t, tt.want, resp)
			}
		})
	}
}

func TestServer_DryRun_PassesRequestThrough(t *testing.T) {
	plugin := &dryRunRPCPlugin{resp: NewDryRunResponse()}
	req := &pbc.DryRunRequest{
		Resource:             &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"},
		SimulationParameters: map[string]string{"region": "us-west-2"},
	}

	_, err := NewServer(plugin).DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Same(t, req, plugin.got)
}
