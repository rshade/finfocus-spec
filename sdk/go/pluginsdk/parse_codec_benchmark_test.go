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
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// BenchmarkCostParseBinaryGuard includes RPC transport costs. Compare the
// non_parse case to isolate the delegation behavior; no codec is registered globally.
func BenchmarkCostParseBinaryGuard(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(b, err)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	logger := zerolog.Nop()
	plugin := newCostParseTestPlugin(
		costParseFunc(
			func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) { return &pbc.ParseResponse{}, nil },
		),
	)
	go func() {
		done <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{Plugin: plugin, Listener: listener, Logger: &logger, PluginInfo: pluginsdk.NewPluginInfo("benchmark-parser", "1.0.0", pluginsdk.WithParseLimits([]string{"fixture"}, []string{"content"}, 65536))})
	}()
	b.Cleanup(func() { cancel(); require.NoError(b, <-done) })
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, conn.Close()) })
	for _, name := range []string{"normal", "rejected", "near_limit", "non_parse"} {
		b.Run(name, func(b *testing.B) {
			req := costParseFixtureRequest()
			method := pbc.CostParseService_Parse_FullMethodName
			result := proto.Message(&pbc.ParseResponse{})
			expected := codes.OK
			if name == "near_limit" {
				req.GetFiles().GetFiles()[0].Content = []byte(strings.Repeat("x", 65536))
			}
			raw, marshalErr := proto.Marshal(req)
			require.NoError(b, marshalErr)
			if name == "rejected" {
				raw = append(raw, costParseWireField(3, []byte("root"))...)
				expected = codes.InvalidArgument
			}
			if name == "non_parse" {
				raw = nil
				method = pbc.CostSourceService_GetPluginInfo_FullMethodName
				result = &pbc.GetPluginInfoResponse{}
			}
			call := func() error { return conn.Invoke(ctx, method, raw, result, grpc.ForceCodec(costParseRawCodec{})) }
			require.Equal(b, expected, status.Code(call()))
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if got := status.Code(call()); got != expected {
					b.Fatalf("code %v, want %v", got, expected)
				}
			}
		})
	}
}
