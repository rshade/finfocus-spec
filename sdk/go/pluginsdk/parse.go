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
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// ParseLimits are a parser's advertised formats, input modes and byte limit.
type ParseLimits = pt.ParseLimits

// ParseParseLimits reads parser metadata, returning nil,nil when absent.
func ParseParseLimits(metadata map[string]string) (*ParseLimits, error) {
	return pt.ParseParseLimits(metadata)
}

// WithParseLimits advertises supplied parser limits. PluginInfo.Validate rejects
// malformed values; this option does not silently repair configuration.
func WithParseLimits(formats, inputModes []string, maxInputBytes int64) PluginInfoOption {
	return func(info *PluginInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]string)
		}
		for k, v := range pt.FormatParseLimits(formats, inputModes, maxInputBytes) {
			info.Metadata[k] = v
		}
	}
}

// ValidateParseRequest validates syntax and advertised support without file I/O.
func ValidateParseRequest(req *pbc.ParseRequest, limits *ParseLimits) error {
	return pt.ValidateParseRequest(req, limits)
}

// ValidateParseResponse validates descriptor, source, unknown and warning structure.
func ValidateParseResponse(req *pbc.ParseRequest, resp *pbc.ParseResponse) error {
	return pt.ValidateParseResponse(req, resp)
}

func dispatchParse(
	ctx context.Context,
	provider CostParseProvider,
	limits *ParseLimits,
	req *pbc.ParseRequest,
) (*pbc.ParseResponse, error) {
	if provider == nil {
		return nil, status.Error(codes.Unimplemented, "parsing is not implemented")
	}
	if err := ValidateParseRequest(req, limits); err != nil {
		return nil, err
	}
	resp, err := provider.Parse(ctx, req)
	if err != nil {
		fallback, normalized := handlerStatus(err, "Parse handler failed")
		if fallback {
			return nil, status.Error(codes.Unimplemented, "parsing is not implemented")
		}
		return nil, normalized
	}
	if err = ValidateParseResponse(req, resp); err != nil {
		return nil, status.Error(codes.Internal, "Parse handler returned invalid response")
	}
	return resp, nil
}

type costParseGRPCServer struct {
	pbc.UnimplementedCostParseServiceServer

	provider CostParseProvider
	limits   *ParseLimits
}

// Parse applies shared request, handler-error and response validation.
func (s *costParseGRPCServer) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return dispatchParse(ctx, s.provider, s.limits, req)
}

type costParseConnectHandler struct {
	provider CostParseProvider
	limits   *ParseLimits
}

// Parse adapts the shared dispatch result to Connect.
func (h *costParseConnectHandler) Parse(
	ctx context.Context,
	req *connect.Request[pbc.ParseRequest],
) (*connect.Response[pbc.ParseResponse], error) {
	resp, err := dispatchParse(ctx, h.provider, h.limits, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}

func configuredParseLimits(ctx context.Context, config ServeConfig) (*ParseLimits, error) {
	if _, ok := config.Plugin.(CostParseProvider); !ok {
		return nil, nil //nolint:nilnil // No parser means no configured parse limits.
	}
	var metadata map[string]string
	if config.PluginInfo != nil {
		metadata = config.PluginInfo.Metadata
	}
	provider, ok := config.Plugin.(PluginInfoProvider)
	if !ok {
		return requiredParseLimits(metadata)
	}
	info, err := provider.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	if err != nil {
		fallback, normalized := handlerStatus(err, "GetPluginInfo handler failed")
		if fallback {
			return requiredParseLimits(metadata)
		}
		return nil, normalized
	}
	if info == nil {
		return nil, fmt.Errorf("%w: nil plugin information", pt.ErrInvalidParseLimits)
	}
	return requiredParseLimits(info.GetMetadata())
}

func requiredParseLimits(metadata map[string]string) (*ParseLimits, error) {
	limits, err := ParseParseLimits(metadata)
	if err != nil {
		return nil, err
	}
	if limits == nil {
		return nil, fmt.Errorf("%w: registered parser requires parse metadata", pt.ErrInvalidParseLimits)
	}
	return limits, nil
}

// validateParseAdvertisement runs after capability inference so both configured
// and provider-owned discovery responses obey the same advertisement gate.
func validateParseAdvertisement(metadata map[string]string, capabilities []pbc.PluginCapability) error {
	limits, err := ParseParseLimits(metadata)
	if err != nil {
		return err
	}
	if limits == nil && slices.Contains(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE) {
		return fmt.Errorf("%w: COST_PARSE requires parse metadata", pt.ErrInvalidParseLimits)
	}
	return nil
}
