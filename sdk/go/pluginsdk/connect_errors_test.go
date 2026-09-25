package pluginsdk_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

// optionalRPCConnectPlugin implements DryRunHandler and ResolveResourceTypesProvider.
type optionalRPCConnectPlugin struct {
	connectTestPlugin
}

func (p *optionalRPCConnectPlugin) HandleDryRun(
	_ context.Context,
	_ *pbc.DryRunRequest,
) (*pbc.DryRunResponse, error) {
	return pluginsdk.NewDryRunResponse(pluginsdk.WithResourceTypeSupported(true)), nil
}

func (p *optionalRPCConnectPlugin) ResolveResourceTypes(
	_ context.Context,
	req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
	mappings := make(map[string]*pbc.ResourceTypeMapping, len(req.GetSourceTypes()))
	for _, src := range req.GetSourceTypes() {
		mappings[src] = &pbc.ResourceTypeMapping{PulumiToken: "aws:ec2/instance:Instance", Supported: true}
	}
	return &pbc.ResolveResourceTypesResponse{Mappings: mappings}, nil
}

func serveConnectForTest(t *testing.T, plugin pluginsdk.Plugin) pbcconnect.CostSourceServiceClient {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web:      pluginsdk.WebConfig{Enabled: true, EnableHealthEndpoint: true},
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(5 * time.Second):
			t.Error("server did not shut down in time")
		}
	})

	addr := listener.Addr().String()
	waitForServer(t, addr)
	return pbcconnect.NewCostSourceServiceClient(http.DefaultClient, "http://"+addr)
}

func TestConnectHandler_PreservesStatusCodes(t *testing.T) {
	client := serveConnectForTest(t, &connectTestPlugin{name: "codes"})
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
		want connect.Code
	}{
		{
			name: "Supports nil resource is InvalidArgument",
			call: func() error {
				_, err := client.Supports(ctx, connect.NewRequest(&pbc.SupportsRequest{}))
				return err
			},
			want: connect.CodeInvalidArgument,
		},
		{
			name: "GetBudgets without provider is Unimplemented",
			call: func() error {
				_, err := client.GetBudgets(ctx, connect.NewRequest(&pbc.GetBudgetsRequest{}))
				return err
			},
			want: connect.CodeUnimplemented,
		},
		{
			name: "GetPluginInfo on legacy plugin is Unimplemented",
			call: func() error {
				_, err := client.GetPluginInfo(ctx, connect.NewRequest(&pbc.GetPluginInfoRequest{}))
				return err
			},
			want: connect.CodeUnimplemented,
		},
		{
			name: "DryRun without handler is Unimplemented",
			call: func() error {
				_, err := client.DryRun(ctx, connect.NewRequest(&pbc.DryRunRequest{
					Resource: &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"},
				}))
				return err
			},
			want: connect.CodeUnimplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			require.Error(t, err)
			assert.Equal(t, tt.want, connect.CodeOf(err), "error: %v", err)
		})
	}
}

func TestConnectHandler_OptionalRPCsReachPlugin(t *testing.T) {
	client := serveConnectForTest(t, &optionalRPCConnectPlugin{connectTestPlugin{name: "optional"}})
	ctx := context.Background()

	dryRun, err := client.DryRun(ctx, connect.NewRequest(&pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"},
	}))
	require.NoError(t, err)
	assert.True(t, dryRun.Msg.GetResourceTypeSupported())

	resolved, err := client.ResolveResourceTypes(ctx, connect.NewRequest(&pbc.ResolveResourceTypesRequest{
		SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
		SourceTypes:  []string{"aws_instance"},
	}))
	require.NoError(t, err)
	require.Contains(t, resolved.Msg.GetMappings(), "aws_instance")
	assert.Equal(t, "aws:ec2/instance:Instance", resolved.Msg.GetMappings()["aws_instance"].GetPulumiToken())
}
