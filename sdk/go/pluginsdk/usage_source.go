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

package pluginsdk

import (
	"context"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// usageSourceGRPCServer adapts a UsageSourceProvider to the generated gRPC
// server interface, so plugins need not embed the Unimplemented server.
type usageSourceGRPCServer struct {
	pbc.UnimplementedUsageSourceServiceServer

	provider UsageSourceProvider
}

func (s *usageSourceGRPCServer) GetStats(
	ctx context.Context, req *pbc.GetStatsRequest,
) (*pbc.GetStatsResponse, error) {
	return s.provider.GetStats(ctx, req)
}

// usageSourceConnectHandler adapts a UsageSourceProvider to the generated
// Connect handler interface, preserving gRPC status codes via toConnectError.
type usageSourceConnectHandler struct {
	provider UsageSourceProvider
}

func (h *usageSourceConnectHandler) GetStats(
	ctx context.Context, req *connect.Request[pbc.GetStatsRequest],
) (*connect.Response[pbc.GetStatsResponse], error) {
	resp, err := h.provider.GetStats(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}

// warnUsageSourceCapabilities logs a warning when a usage source relies on
// inferred capabilities. Inference always adds the four pricing capabilities
// from the required Plugin interface, so a usage-only plugin would otherwise
// advertise pricing it does not offer. Plugins that implement
// PluginInfoProvider control their own capabilities and are not warned.
func warnUsageSourceCapabilities(logger *zerolog.Logger, plugin Plugin, info *PluginInfo) {
	if _, ok := plugin.(UsageSourceProvider); !ok {
		return
	}
	if _, ok := plugin.(PluginInfoProvider); ok {
		return
	}
	if info != nil && len(info.Capabilities) > 0 {
		return
	}
	logger.Warn().
		Str("capability", pbc.PluginCapability_PLUGIN_CAPABILITY_USAGE_STATS.String()).
		Msg("usage source relies on inferred capabilities, which include pricing capabilities; " +
			"usage-only plugins should set PluginInfo.Capabilities explicitly")
}
