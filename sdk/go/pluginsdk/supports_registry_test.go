//nolint:testpackage // Testing internal Server implementation with mocks
package pluginsdk

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSupports_NoRegistryDelegatesToPlugin(t *testing.T) {
	constructors := []struct {
		name string
		new  func(Plugin) *Server
	}{
		{name: "NewServer", new: NewServer},
		{name: "NewServerWithRegistry nil", new: func(p Plugin) *Server { return NewServerWithRegistry(p, nil) }},
		{
			name: "explicit DefaultRegistryLookup",
			new:  func(p Plugin) *Server { return NewServerWithRegistry(p, &DefaultRegistryLookup{}) },
		},
		{
			name: "NewServerWithOptions nil registry",
			new:  func(p Plugin) *Server { return NewServerWithOptions(p, nil, nil, nil) },
		},
	}

	awsEast := &pbc.ResourceDescriptor{Provider: "aws", Region: "us-east-1", ResourceType: "ec2"}
	typeOnly := &pbc.ResourceDescriptor{ResourceType: "aws:ec2/instance:Instance"}

	cases := []struct {
		name          string
		plugin        Plugin
		resource      *pbc.ResourceDescriptor
		wantSupported bool
		wantReason    string
	}{
		{
			name:          "plugin supports resource",
			plugin:        &mockSupportsPlugin{mockPlugin: mockPlugin{name: "p"}, supported: true},
			resource:      awsEast,
			wantSupported: true,
		},
		{
			name: "plugin declines resource",
			plugin: &mockSupportsPlugin{
				mockPlugin: mockPlugin{name: "p"},
				reason:     "outside compiled region",
			},
			resource:   awsEast,
			wantReason: "outside compiled region",
		},
		{
			name:          "empty provider and region as hosts send today",
			plugin:        &mockSupportsPlugin{mockPlugin: mockPlugin{name: "p"}, supported: true},
			resource:      typeOnly,
			wantSupported: true,
		},
		{
			name:       "plugin without SupportsProvider gets default response",
			plugin:     &mockPlugin{name: "p"},
			resource:   typeOnly,
			wantReason: DefaultSupportsNotImplementedReason,
		},
	}

	for _, ctor := range constructors {
		for _, tc := range cases {
			t.Run(ctor.name+"/"+tc.name, func(t *testing.T) {
				server := ctor.new(tc.plugin)

				resp, err := server.Supports(context.Background(), &pbc.SupportsRequest{Resource: tc.resource})
				require.NoError(t, err)

				assert.Equal(t, tc.wantSupported, resp.GetSupported())
				assert.Equal(t, tc.wantReason, resp.GetReason())
				assert.Equal(t, inferCapabilities(tc.plugin), resp.GetCapabilitiesEnum())
				assert.True(t, resp.GetCapabilities()["supports_projected_costs"])
			})
		}
	}
}

func TestSupports_ConfiguredRegistryStillValidates(t *testing.T) {
	plugin := &mockSupportsPlugin{mockPlugin: mockPlugin{name: "p"}, supported: true}
	server := NewServerWithRegistry(plugin, &mockRegistry{plugins: map[string]string{"aws:us-east-1": "p"}})

	tests := []struct {
		name     string
		resource *pbc.ResourceDescriptor
		wantCode codes.Code
	}{
		{
			name:     "registered",
			resource: &pbc.ResourceDescriptor{Provider: "aws", Region: "us-east-1"},
			wantCode: codes.OK,
		},
		{
			name:     "unregistered region",
			resource: &pbc.ResourceDescriptor{Provider: "aws", Region: "eu-west-1"},
			wantCode: codes.InvalidArgument,
		},
		{name: "empty provider and region", resource: &pbc.ResourceDescriptor{}, wantCode: codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.Supports(context.Background(), &pbc.SupportsRequest{Resource: tt.resource})
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestSupports_NilResourceStillRejectedWithoutRegistry(t *testing.T) {
	server := NewServer(&mockSupportsPlugin{mockPlugin: mockPlugin{name: "p"}, supported: true})

	_, err := server.Supports(context.Background(), &pbc.SupportsRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestServe_SupportsWithoutRegistryReachesPlugin(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, ServeConfig{
			Plugin: &mockSupportsPlugin{
				mockPlugin: mockPlugin{name: "grpc-supports"},
				reason:     "declined by plugin",
			},
			Listener:        listener,
			handshakeWriter: io.Discard,
		})
	}()

	conn, err := grpc.NewClient(listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()
	resp, err := pbc.NewCostSourceServiceClient(conn).Supports(callCtx, &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "aws:ec2/instance:Instance"},
	}, grpc.WaitForReady(true))
	require.NoError(t, err)
	assert.False(t, resp.GetSupported())
	assert.Equal(t, "declined by plugin", resp.GetReason())

	cancel()
	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
