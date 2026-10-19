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

package testing

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const parseScenarioTimeout = 10 * time.Second
const parseDeterminismCalls = 100

// ParseServer is satisfied by a parser provider or generated gRPC server.
type ParseServer interface {
	Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error)
}

type parseAdapter struct {
	pbc.UnimplementedCostParseServiceServer

	impl ParseServer
}

// Parse forwards to the provider under test without changing its behavior.
func (a *parseAdapter) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	return a.impl.Parse(ctx, req)
}

// CostParseHarness serves a parser over the shared in-memory gRPC transport.
type CostParseHarness struct {
	bufconnHarness[pbc.CostParseServiceClient]
}

// NewCostParseHarness creates a provider-neutral parser harness. Call Start before Client.
func NewCostParseHarness(impl ParseServer) *CostParseHarness {
	return &CostParseHarness{
		newBufconnHarness(
			func(server *grpc.Server) { pbc.RegisterCostParseServiceServer(server, &parseAdapter{impl: impl}) },
			pbc.NewCostParseServiceClient,
		),
	}
}

// Client returns the generated parser client after Start.
func (h *CostParseHarness) Client() pbc.CostParseServiceClient { return h.client }

// CostParseConformanceConfig supplies advertised limits and caller-owned valid
// fixtures. ValidRequest is required; mode-specific fixtures enable more coverage.
// Path fixtures must refer to accessible caller-owned roots with stable dependencies.
type CostParseConformanceConfig struct {
	Limits         ParseLimits
	ValidRequest   *pbc.ParseRequest
	ContentRequest *pbc.ParseRequest
	PathRequest    *pbc.ParseRequest
}

// RunCostParseConformance runs structural and error scenarios over real gRPC.
// It clones fixtures before mutation, bounds each call to ten seconds, and never
// allocates an advertised-limit-plus-one payload or prescribes vendor contents.
// Binary/JSON conflicting oneof members are tested separately by Serve tests.
func RunCostParseConformance(t *testing.T, server ParseServer, config CostParseConformanceConfig) {
	t.Helper()
	if err := validateConformanceConfig(config); err != nil {
		t.Fatalf("invalid conformance fixture/configuration: %v", err)
	}
	crossCheckParseMetadata(t, server, &config.Limits)
	harness := NewCostParseHarness(server)
	harness.Start(t)
	defer harness.Stop()
	client := harness.Client()
	t.Run("format_rejected", func(t *testing.T) {
		req := proto.CloneOf(config.ValidRequest)
		for n := 0; ; n++ {
			req.Format = fmt.Sprintf("unsupported_%d", n)
			if !slices.Contains(config.Limits.Formats, req.GetFormat()) {
				break
			}
		}
		_, err := conformanceParse(client, req)
		requireParseCode(t, err, codes.FailedPrecondition, UnsupportedFormatReason)
	})
	t.Run("input_mode_xor", func(t *testing.T) {
		req := proto.CloneOf(config.ValidRequest)
		req.Input = nil
		_, err := conformanceParse(client, req)
		requireParseCode(t, err, codes.InvalidArgument, "")
	})
	t.Run("unsupported_input_mode", func(t *testing.T) {
		runUnsupportedParseMode(t, client, config)
	})
	t.Run("path_traversal_rejected", func(t *testing.T) {
		fixture := conformanceContentFixture(config)
		if fixture == nil {
			t.Skip("no caller-provided content-mode fixture")
		}
		req := proto.CloneOf(fixture)
		req.GetFiles().GetFiles()[0].Path = "../escape"
		_, err := conformanceParse(client, req)
		requireParseCode(t, err, codes.InvalidArgument, "")
	})
	for _, name := range []string{"source_ids_unique", "unknown_not_absent", "descriptors_valid"} {
		t.Run(name, func(t *testing.T) {
			for _, req := range conformanceParseFixtures(config) {
				requireValidConformanceParse(t, client, req)
			}
		})
	}
	t.Run("deterministic", func(t *testing.T) {
		for _, req := range conformanceParseFixtures(config) {
			requireDeterministicParse(t, client, req)
		}
	})
}

func validateConformanceConfig(config CostParseConformanceConfig) error {
	if err := ValidateParseRequest(config.ValidRequest, &config.Limits); err != nil {
		return fmt.Errorf("valid_request: %w", err)
	}
	for _, fixture := range []struct {
		name    string
		req     *pbc.ParseRequest
		content bool
	}{
		{"content_request", config.ContentRequest, true},
		{"path_request", config.PathRequest, false},
	} {
		if fixture.req == nil {
			continue
		}
		if err := ValidateParseRequest(fixture.req, &config.Limits); err != nil {
			return fmt.Errorf("%s: %w", fixture.name, err)
		}
		if fixture.content != (fixture.req.GetFiles() != nil) {
			return fmt.Errorf("%w: %s selects the wrong input mode", ErrInvalidParseRequest, fixture.name)
		}
	}
	return nil
}

func runUnsupportedParseMode(t *testing.T, client pbc.CostParseServiceClient, config CostParseConformanceConfig) {
	t.Helper()
	if slices.Contains(config.Limits.InputModes, parseInputModeContent) &&
		slices.Contains(config.Limits.InputModes, parseInputModePath) {
		t.Skip("both input modes are advertised")
	}
	req := proto.CloneOf(config.ValidRequest)
	if slices.Contains(config.Limits.InputModes, parseInputModeContent) {
		// This root is only sent for an unadvertised mode; accepted path-mode
		// scenarios always use an accessible fixture supplied by the caller.
		req.Input = &pbc.ParseRequest_RootPath{RootPath: t.TempDir()}
	} else {
		req.Input = &pbc.ParseRequest_Files{
			Files: &pbc.ParseFileSet{Files: []*pbc.ParseFile{{Path: "probe.fixture"}}},
		}
	}
	_, err := conformanceParse(client, req)
	requireParseCode(t, err, codes.FailedPrecondition, UnsupportedInputModeReason)
}

func requireDeterministicParse(t *testing.T, client pbc.CostParseServiceClient, req *pbc.ParseRequest) {
	t.Helper()
	baseline := requireValidConformanceParse(t, client, req)
	baseline.ExpiresAt = nil
	for range parseDeterminismCalls - 1 {
		response := requireValidConformanceParse(t, client, req)
		response.ExpiresAt = nil
		if !proto.Equal(baseline, response) {
			t.Fatal("resources or warnings changed for identical input and dependencies")
		}
	}
}

func conformanceContentFixture(config CostParseConformanceConfig) *pbc.ParseRequest {
	if config.ContentRequest != nil {
		return config.ContentRequest
	}
	if config.ValidRequest.GetFiles() != nil {
		return config.ValidRequest
	}
	return nil
}

func conformanceParseFixtures(config CostParseConformanceConfig) []*pbc.ParseRequest {
	fixtures := []*pbc.ParseRequest{config.ValidRequest}
	for _, req := range []*pbc.ParseRequest{config.ContentRequest, config.PathRequest} {
		if req != nil && !proto.Equal(req, config.ValidRequest) {
			fixtures = append(fixtures, req)
		}
	}
	return fixtures
}

func conformanceParse(client pbc.CostParseServiceClient, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), parseScenarioTimeout)
	defer cancel()
	return client.Parse(ctx, proto.CloneOf(req))
}

func requireValidConformanceParse(
	t *testing.T,
	client pbc.CostParseServiceClient,
	req *pbc.ParseRequest,
) *pbc.ParseResponse {
	t.Helper()
	response, err := conformanceParse(client, req)
	if err != nil {
		t.Fatalf("Parse valid caller fixture: %v", err)
	}
	if err = ValidateParseResponse(req, response); err != nil {
		t.Fatalf("invalid parser response: %v", err)
	}
	return response
}

func requireParseCode(t *testing.T, err error, want codes.Code, reason string) {
	t.Helper()
	if status.Code(err) != want {
		t.Fatalf("Parse status %v, want %v: %v", status.Code(err), want, err)
	}
	if reason == "" {
		return
	}
	for _, detail := range status.Convert(err).Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetReason() == reason &&
			info.GetDomain() == ParseErrorDomain {
			return
		}
	}
	t.Fatalf("Parse error missing ErrorInfo reason %s domain %s: %v", reason, ParseErrorDomain, err)
}

func crossCheckParseMetadata(t *testing.T, server ParseServer, limits *ParseLimits) {
	t.Helper()
	provider, ok := server.(interface {
		GetPluginInfo(context.Context, *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error)
	})
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), parseScenarioTimeout)
	defer cancel()
	info, err := provider.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	if status.Code(err) == codes.Unimplemented {
		return
	}
	if err != nil {
		t.Fatalf("parser discovery: %v", err)
	}
	advertised, err := ParseParseLimits(info.GetMetadata())
	if err != nil || advertised == nil {
		t.Fatalf("invalid parser discovery metadata: %v", err)
	}
	formats := slices.Clone(limits.Formats)
	modes := slices.Clone(limits.InputModes)
	slices.Sort(formats)
	slices.Sort(modes)
	slices.Sort(advertised.Formats)
	slices.Sort(advertised.InputModes)
	if limits.MaxInputBytes != advertised.MaxInputBytes || !slices.Equal(formats, advertised.Formats) ||
		!slices.Equal(modes, advertised.InputModes) {
		t.Fatal("conformance limits differ from served parser metadata")
	}
	// Empty raw capabilities request interface inference from Serve.
	if len(info.GetCapabilities()) > 0 &&
		!slices.Contains(info.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE) {
		t.Fatal("parser discovery omits COST_PARSE")
	}
}
