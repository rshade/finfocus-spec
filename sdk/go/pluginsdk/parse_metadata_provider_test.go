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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

type costParseMetadataPlugin struct {
	*pluginsdk.BasePlugin

	info *pbc.GetPluginInfoResponse
}

func (p *costParseMetadataPlugin) GetPluginInfo(
	context.Context,
	*pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return p.info, nil
}
func TestCostParseProviderMetadataGate(t *testing.T) {
	for _, name := range []string{"partial without capability", "declared missing", "inferred missing", "valid"} {
		t.Run(name, func(t *testing.T) {
			info := &pbc.GetPluginInfoResponse{Name: "metadata", Version: "1.0.0", SpecVersion: pluginsdk.SpecVersion}
			plugin := &costParseMetadataPlugin{BasePlugin: pluginsdk.NewBasePlugin("metadata"), info: info}
			var impl pluginsdk.Plugin = plugin
			switch name {
			case "partial without capability":
				info.Metadata = map[string]string{"parse_formats": "fixture"}
			case "declared missing":
				info.Capabilities = []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE}
			case "inferred missing":
				impl = &costParseMetadataParser{costParseMetadataPlugin: plugin}
			case "valid":
				info.Capabilities = []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE}
				info.Metadata = pt.FormatParseLimits([]string{"fixture"}, []string{"content"}, 1024)
			}
			before := proto.Clone(info)
			resp, err := pluginsdk.NewServer(impl).GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			if name == "valid" {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Equal(t, codes.Internal, status.Code(err))
				require.Nil(t, resp)
			}
			require.True(t, proto.Equal(info, before))
		})
	}
}

type costParseMetadataParser struct{ *costParseMetadataPlugin }

func (*costParseMetadataParser) Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return &pbc.ParseResponse{}, nil
}
