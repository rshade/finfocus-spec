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

package pluginsdk_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// decliningChild is a plugin whose GetProjectedCost answers with a status.
type decliningChild struct {
	clientTestPlugin

	err error
}

func (c *decliningChild) GetProjectedCost(
	context.Context, *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return nil, c.err
}

// routerPlugin forwards GetProjectedCost to a child through pluginsdk.Client and returns
// the client's error unchanged, as a routing plugin does.
type routerPlugin struct {
	pbc.UnimplementedCostSourceServiceServer

	child *pluginsdk.Client
}

func (r *routerPlugin) GetProjectedCost(
	ctx context.Context, req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	return r.child.GetProjectedCost(ctx, req)
}

func serveChild(t *testing.T, childErr error) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   &decliningChild{clientTestPlugin: clientTestPlugin{name: "child"}, err: childErr},
			Listener: listener,
			Web:      pluginsdk.WebConfig{Enabled: true, EnableHealthEndpoint: true},
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(time.Second):
			t.Error("child did not shut down in time")
		}
	})
	addr := listener.Addr().String()
	waitForServer(t, addr)
	return addr
}

// TestClientErrorKeepsStatusThroughRouter shows that a routing plugin returning a
// pluginsdk.Client error from a gRPC handler keeps the child's status code.
func TestClientErrorKeepsStatusThroughRouter(t *testing.T) {
	newClients := map[string]func(string) *pluginsdk.Client{
		"connect": pluginsdk.NewConnectClient,
		"grpc":    pluginsdk.NewGRPCClient,
	}
	for protocol, newClient := range newClients {
		t.Run(protocol, func(t *testing.T) {
			childErr := status.Error(codes.InvalidArgument, "resource.sku is required")
			child := newClient("http://" + serveChild(t, childErr))
			t.Cleanup(child.Close)

			harness := plugintesting.NewTestHarness(&routerPlugin{child: child})
			harness.Start(t)
			defer harness.Stop()

			_, err := harness.Client().GetProjectedCost(context.Background(), &pbc.GetProjectedCostRequest{})
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err), "got %v", err)
			assert.Contains(t, err.Error(), "resource.sku is required")
		})
	}
}
