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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func costParsePartialRequest(content string) *pbc.ParseRequest {
	return costParsePolicyFiles(&pbc.ParseFile{Path: "partial.fixture", Content: []byte(content)})
}

func costParsePartialDescriptor(t *testing.T) *pbc.ResourceDescriptor {
	t.Helper()
	attributes, err := structpb.NewStruct(map[string]any{
		"Sizing": map[string]any{"Cpu": 4.0, "Memory": 0.0},
		"Label":  "", "sizing": map[string]any{"Cpu": 8.0},
	})
	require.NoError(t, err)
	return &pbc.ResourceDescriptor{
		Provider: "custom", ResourceType: "custom:compute:Instance",
		Attributes: attributes, Tags: map[string]string{"environment": "test"},
	}
}

func TestCostParseMockPartialResults(t *testing.T) {
	descriptor := costParsePartialDescriptor(t)
	before := proto.CloneOf(descriptor)
	parser := pt.NewMockCostParser(pt.WithParseResource(descriptor))
	req := costParsePartialRequest(
		"resource LogicalDatabase\nresource Deployment/api\nunknown Deployment/api Sizing.Cpu\nskip Service/unsupported\nnotice Some resources need review\n",
	)
	requestBefore := proto.CloneOf(req)
	resp, err := parser.Parse(context.Background(), req)
	require.NoError(t, err)
	require.NoError(t, pt.ValidateParseResponse(req, resp))
	require.Len(t, resp.GetResources(), 2)
	resolved := resp.GetResources()[0]
	require.Equal(t, "LogicalDatabase", resolved.GetSourceId())
	require.Empty(t, resolved.GetUnknownProperties())
	require.True(t, proto.Equal(before, resolved.GetResource()), "resolved descriptor changed")
	partial := resp.GetResources()[1]
	require.Equal(t, "Deployment/api", partial.GetSourceId())
	require.Equal(t, []string{"Sizing.Cpu"}, partial.GetUnknownProperties())
	fields := partial.GetResource().GetAttributes().GetFields()
	sizing := fields["Sizing"].GetStructValue().GetFields()
	require.NotContains(t, sizing, "Cpu", "unknown values must be absent, including null")
	require.Contains(t, sizing, "Memory")
	require.Zero(t, sizing["Memory"].GetNumberValue(), "known zero must remain present")
	require.Contains(t, fields, "Label")
	require.Empty(t, fields["Label"].GetStringValue(), "known empty string must remain present")
	require.InDelta(t, 8, fields["sizing"].GetStructValue().GetFields()["Cpu"].GetNumberValue(), 0.000001,
		"unknown paths are case sensitive")
	require.Len(t, resp.GetWarnings(), 2)
	skipped := resp.GetWarnings()[0]
	require.Equal(t, pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING, skipped.GetSeverity())
	require.Equal(t, "resource_skipped", skipped.GetCode())
	require.Equal(t, "Service/unsupported", skipped.GetSourceId())
	require.NotEmpty(t, skipped.GetMessage())
	notice := resp.GetWarnings()[1]
	require.Equal(t, pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_INFO, notice.GetSeverity())
	require.Equal(t, "document_notice", notice.GetCode())
	require.Empty(t, notice.GetSourceId())
	require.Equal(t, "Some resources need review", notice.GetMessage())
	require.True(t, proto.Equal(descriptor, before), "unknown removal mutated caller descriptor")
	require.True(t, proto.Equal(req, requestBefore), "Parse mutated caller request")

	again, err := parser.Parse(context.Background(), req)
	require.NoError(t, err)
	require.True(t, proto.Equal(resp, again), "partial results changed across calls")
	pristine, err := parser.Parse(context.Background(), costParsePartialRequest("resource Fresh\n"))
	require.NoError(t, err)
	require.Len(t, pristine.GetResources(), 1)
	require.True(t, proto.Equal(before, pristine.GetResources()[0].GetResource()),
		"unknown removal changed the configured descriptor for future calls")
}

func TestCostParseMockPartialEmptyAndSkipped(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		warnings int
	}{
		{"empty", "", 0},
		{"blank", "\n  \n", 0},
		{"all skipped", "skip LogicalBucket\nskip Service/api\n", 2},
		{"document notice", "notice Nothing priceable\n", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := costParsePartialRequest(tc.content)
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			require.NoError(t, err)
			require.NoError(t, pt.ValidateParseResponse(req, resp))
			require.Empty(t, resp.GetResources())
			require.Len(t, resp.GetWarnings(), tc.warnings)
			if tc.name == "all skipped" {
				for i, source := range []string{"LogicalBucket", "Service/api"} {
					warning := resp.GetWarnings()[i]
					require.Equal(t, source, warning.GetSourceId())
					require.Equal(t, "resource_skipped", warning.GetCode())
					require.Equal(t, pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING, warning.GetSeverity())
					require.NotEmpty(t, warning.GetMessage())
				}
			}
		})
	}
}

func TestCostParseMockPartialInvalidSources(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"duplicate resources", "resource LogicalId\nresource LogicalId\n"},
		{"duplicate skips", "skip Service/api\nskip Service/api\n"},
		{"resource then skip", "resource LogicalId\nskip LogicalId\n"},
		{"skip then resource", "skip LogicalId\nresource LogicalId\n"},
		{"unknown missing source", "unknown Missing Sizing.Cpu\n"},
		{"unknown forward source", "unknown LogicalId Sizing.Cpu\nresource LogicalId\n"},
		{"unknown skipped source", "skip LogicalId\nunknown LogicalId Sizing.Cpu\n"},
		{"duplicate unknown", "resource LogicalId\nunknown LogicalId Sizing.Cpu\nunknown LogicalId Sizing.Cpu\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := costParsePartialRequest(tc.content)
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
			require.Nil(t, resp)
			require.ErrorContains(t, err, "partial.fixture:")
		})
	}
}

func TestCostParseMockPartialUnknownAncestors(t *testing.T) {
	cases := []struct {
		name  string
		value *structpb.Value
		valid bool
	}{
		{"missing ancestor", nil, true},
		{"object ancestor", structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
			"Cpu": structpb.NewNumberValue(4),
		}}), true},
		{"null ancestor", structpb.NewNullValue(), false},
		{"scalar ancestor", structpb.NewNumberValue(0), false},
		{"list ancestor", structpb.NewListValue(&structpb.ListValue{}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			descriptor := costParsePartialDescriptor(t)
			if tc.value == nil {
				delete(descriptor.GetAttributes().GetFields(), "Sizing")
			} else {
				descriptor.GetAttributes().GetFields()["Sizing"] = tc.value
			}
			before := proto.CloneOf(descriptor)
			req := costParsePartialRequest("resource NativeId\nunknown NativeId Sizing.Cpu\n")
			resp, err := pt.NewMockCostParser(pt.WithParseResource(descriptor)).Parse(context.Background(), req)
			require.True(t, proto.Equal(descriptor, before), "caller attributes changed")
			if !tc.valid {
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				require.Nil(t, resp)
				return
			}
			require.NoError(t, err)
			require.NoError(t, pt.ValidateParseResponse(req, resp))
			require.Len(t, resp.GetResources(), 1)
			require.Equal(t, []string{"Sizing.Cpu"}, resp.GetResources()[0].GetUnknownProperties())
		})
	}
}

func TestCostParseMockActions(t *testing.T) {
	ctx := context.Background()
	parser := pt.NewMockCostParser()
	cases := []struct {
		name    string
		content string
		want    []pbc.ParseAction
		wantErr bool
	}{
		{"no action lines leave zero", "resource A\nresource B\n", []pbc.ParseAction{0, 0}, false},
		{"create", "resource A\naction A create\n", []pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_CREATE}, false},
		{"update", "resource A\naction A update\n", []pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_UPDATE}, false},
		{"delete", "resource A\naction A delete\n", []pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_DELETE}, false},
		{"replace", "resource A\naction A replace\n", []pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_REPLACE}, false},
		{
			"no change",
			"resource A\naction A no_change\n",
			[]pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_NO_CHANGE},
			false,
		},
		{
			"mixed",
			"resource A\nresource B\naction B delete\n",
			[]pbc.ParseAction{pbc.ParseAction_PARSE_ACTION_UNSPECIFIED, pbc.ParseAction_PARSE_ACTION_DELETE},
			false,
		},
		{"unknown name", "resource A\naction A explode\n", nil, true},
		{"unspecified name", "resource A\naction A unspecified\n", nil, true},
		{"unknown source", "action A create\n", nil, true},
		{"skipped source", "skip A\naction A create\n", nil, true},
		{"duplicate action", "resource A\naction A create\naction A delete\n", nil, true},
		{"missing action name", "resource A\naction A\n", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := costParsePartialRequest(c.content)
			resp, err := parser.Parse(ctx, req)
			if c.wantErr {
				require.Error(t, err)
				require.Equal(t, codes.InvalidArgument, status.Code(err))
				return
			}
			require.NoError(t, err)
			require.NoError(t, pt.ValidateParseResponse(req, resp))
			got := make([]pbc.ParseAction, len(resp.GetResources()))
			for i, r := range resp.GetResources() {
				got[i] = r.GetAction()
			}
			require.Equal(t, c.want, got)
		})
	}
}
