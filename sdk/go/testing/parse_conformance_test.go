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

package testing_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// costParseObservedServer records the actual deadlines and request sizes received
// over the harness, without requiring any vendor or synthetic fixture syntax.
type costParseObservedServer struct {
	mu            sync.Mutex
	limits        pt.ParseLimits
	calls         int
	badDeadline   bool
	oversizeProbe bool
	roots         []string
}

func (s *costParseObservedServer) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	deadline, present := ctx.Deadline()
	s.mu.Lock()
	s.calls++
	s.badDeadline = s.badDeadline || !present || time.Until(deadline) <= 0 || time.Until(deadline) > 11*time.Second
	s.oversizeProbe = s.oversizeProbe || proto.Size(req) > 4096
	if req.GetRootPath() != "" {
		s.roots = append(s.roots, req.GetRootPath())
	}
	s.mu.Unlock()
	if err := pt.ValidateParseRequest(req, &s.limits); err != nil {
		return nil, err
	}
	return &pbc.ParseResponse{Resources: []*pbc.ParsedResource{{
		SourceId: "NativeLogicalId", Resource: &pbc.ResourceDescriptor{
			Provider: "custom", ResourceType: "custom:compute:Instance",
		},
	}}}, nil
}

func TestCostParseConformanceHarness(t *testing.T) {
	parser := pt.NewMockCostParser()
	harness := pt.NewCostParseHarness(parser)
	harness.Start(t)
	defer harness.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := costParseMockRequest()
	want, err := parser.Parse(ctx, req)
	require.NoError(t, err)
	got, err := harness.Client().Parse(ctx, req)
	require.NoError(t, err)
	require.True(t, proto.Equal(want, got), "harness changed source-linked descriptors")
}

func TestCostParseConformanceReference(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "input.fixture"), []byte("resource NativeId\n"), 0o600))
	content := costParsePartialRequest("resource NativeId\n")
	path := &pbc.ParseRequest{Format: "fixture", Input: &pbc.ParseRequest_RootPath{RootPath: root}}
	for _, mode := range []string{"both", "content", "path", "both without content fixture"} {
		t.Run(mode, func(t *testing.T) {
			limits := costParsePolicyLimits(1024)
			config := pt.CostParseConformanceConfig{
				Limits: limits, ValidRequest: content, ContentRequest: content, PathRequest: path,
			}
			switch mode {
			case "content":
				limits.InputModes = []string{"content"}
				config.PathRequest = nil
			case "both without content fixture":
				config.ValidRequest = path
				config.ContentRequest = nil
			case "path":
				limits.InputModes = []string{"path"}
				config.ValidRequest = path
				config.ContentRequest = nil
			}
			config.Limits = limits
			contentBefore := proto.CloneOf(content)
			pathBefore := proto.CloneOf(path)
			pt.RunCostParseConformance(t, pt.NewMockCostParser(pt.WithMockParseLimits(limits)), config)
			require.True(t, proto.Equal(content, contentBefore), "conformance mutated caller content fixture")
			require.True(t, proto.Equal(path, pathBefore), "conformance mutated caller root fixture")
		})
	}
}

func TestCostParseConformanceCallerFixturesAndDeadlines(t *testing.T) {
	root := t.TempDir()
	limits := pt.ParseLimits{
		Formats:       []string{"native"},
		InputModes:    []string{"content", "path"},
		MaxInputBytes: math.MaxInt64,
	}
	options, err := structpb.NewStruct(map[string]any{"native_parameter": "keep"})
	require.NoError(t, err)
	content := &pbc.ParseRequest{
		Format: "native", Options: options,
		Input: &pbc.ParseRequest_Files{Files: &pbc.ParseFileSet{Files: []*pbc.ParseFile{{
			Path: "vendor.native", Content: []byte{0, 255, 65},
		}}}},
	}
	path := &pbc.ParseRequest{Format: "native", Input: &pbc.ParseRequest_RootPath{RootPath: root}}
	contentBefore := proto.CloneOf(content)
	pathBefore := proto.CloneOf(path)
	server := &costParseObservedServer{limits: limits}
	pt.RunCostParseConformance(t, server, pt.CostParseConformanceConfig{
		Limits: limits, ValidRequest: content, ContentRequest: content, PathRequest: path,
	})
	require.True(t, proto.Equal(contentBefore, content), "caller content or options changed")
	require.True(t, proto.Equal(pathBefore, path), "caller path changed")
	server.mu.Lock()
	defer server.mu.Unlock()
	require.GreaterOrEqual(t, server.calls, 100, "determinism must exercise 100 calls")
	require.False(t, server.badDeadline, "every parse call must have a 10-second deadline")
	require.False(t, server.oversizeProbe, "conformance must not allocate advertised-limit-plus-one payloads")
	require.NotEmpty(t, server.roots, "caller-owned path fixture was never exercised")
	for _, gotRoot := range server.roots {
		require.Equal(t, root, gotRoot, "conformance invented a root instead of using the caller fixture")
	}
}

func TestCostParseConformanceMalformedResponses(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*pbc.ParseResponse)
	}{
		{
			"duplicate sources",
			func(r *pbc.ParseResponse) { r.Resources = append(r.GetResources(), proto.CloneOf(r.GetResources()[0])) },
		},
		{"invalid descriptor", func(r *pbc.ParseResponse) { r.GetResources()[0].GetResource().Provider = "" }},
		{"present unknown", func(r *pbc.ParseResponse) {
			r.GetResources()[0].UnknownProperties = []string{"Cpu"}
			r.GetResources()[0].GetResource().Attributes = &structpb.Struct{
				Fields: map[string]*structpb.Value{"Cpu": structpb.NewNullValue()},
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := costParseMockRequest()
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			require.NoError(t, err)
			tc.mutate(resp)
			require.ErrorIs(t, pt.ValidateParseResponse(req, resp), pt.ErrInvalidParseResponse)
		})
	}
}

// Empty raw capabilities ask Serve to infer from implemented interfaces.
type costParseInferredInfoServer struct{ *pt.MockCostParser }

func (s *costParseInferredInfoServer) GetPluginInfo(
	ctx context.Context,
	req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	info, err := s.MockCostParser.GetPluginInfo(ctx, req)
	if info != nil {
		info.Capabilities = nil
	}
	return info, err
}
func TestCostParseConformanceInferredCapabilities(t *testing.T) {
	limits := costParsePolicyLimits(1024)
	request := costParseMockRequest()
	pt.RunCostParseConformance(
		t,
		&costParseInferredInfoServer{pt.NewMockCostParser(pt.WithMockParseLimits(limits))},
		pt.CostParseConformanceConfig{Limits: limits, ValidRequest: request, ContentRequest: request},
	)
}
