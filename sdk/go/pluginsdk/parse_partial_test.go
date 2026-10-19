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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	ptest "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func costParsePartialFixture(t *testing.T) *pbc.ParseResponse {
	t.Helper()
	attributes, err := structpb.NewStruct(map[string]any{
		"Replicas": 0, "Zone": "", "Enabled": false, "Sizing": map[string]any{},
	})
	require.NoError(t, err)
	resp := costParseFixtureResponse()
	resp.Resources[0].Resource.Attributes = attributes
	resp.Resources[0].UnknownProperties = []string{"Sizing.Cpu", "Future.Nested", "zone"}
	resp.Warnings = []*pbc.ParseWarning{
		{Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_INFO,
			Code: "document_notice", Message: "Document contains a synthetic example."},
		{Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING,
			Code: "resource_skipped", Message: "Unsupported resource was skipped.", SourceId: "Unsupported"},
		{Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING,
			Code: "unresolved_property", Message: "CPU depends on a runtime value.", SourceId: "Instance"},
	}
	return resp
}

func TestCostParsePartial_AttributesSourcesAndWarnings(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			expected := costParsePartialFixture(t)
			before := proto.Clone(expected)
			plugin := newCostParseTestPlugin(
				costParseFunc(func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
					return expected, nil
				}),
			)
			_, call := startCostParseServer(t, plugin, tr.web)
			req := costParseFixtureRequest()
			got, err := callCostParse(t, call, req)
			require.NoError(t, err)
			require.NoError(t, ptest.ValidateParseResponse(req, got))
			assert.True(t, proto.Equal(expected, got), "transport changed partial results or diagnostics")
			assert.True(t, proto.Equal(before, expected), "Serve mutated provider output")
			require.Len(t, got.GetResources(), 1)
			resource := got.GetResources()[0]
			assert.Equal(t, "Instance", resource.GetSourceId())
			assert.Equal(t, []string{"Sizing.Cpu", "Future.Nested", "zone"}, resource.GetUnknownProperties())
			fields := resource.GetResource().GetAttributes().GetFields()
			require.Contains(t, fields, "Replicas")
			assert.Zero(t, fields["Replicas"].GetNumberValue())
			require.Contains(t, fields, "Zone")
			assert.Empty(t, fields["Zone"].GetStringValue())
			require.Contains(t, fields, "Enabled")
			assert.False(t, fields["Enabled"].GetBoolValue())
			assert.NotContains(t, fields, "zone", "attribute path casing must be preserved")
			assert.NotContains(t, fields["Sizing"].GetStructValue().GetFields(), "Cpu")
			assert.NotContains(t, fields, "Future")
			require.Len(t, got.GetWarnings(), 3)
			assert.Equal(t, "resource_skipped", got.GetWarnings()[1].GetCode())
			assert.Equal(t, "Unsupported", got.GetWarnings()[1].GetSourceId())
			for _, parsed := range got.GetResources() {
				assert.NotEqual(
					t,
					got.GetWarnings()[1].GetSourceId(),
					parsed.GetSourceId(),
					"skipped source was returned",
				)
			}
		})
	}
}

func TestCostParsePartial_EmptyAndAllSkippedSuccess(t *testing.T) {
	for _, tr := range allocTransports {
		for _, scenario := range []string{"empty", "all_skipped", "document_diagnostic_only"} {
			t.Run(tr.name+"/"+scenario, func(t *testing.T) {
				expected := &pbc.ParseResponse{}
				switch scenario {
				case "all_skipped":
					expected.Warnings = []*pbc.ParseWarning{
						{
							Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING,
							Code:     "resource_skipped",
							Message:  "No priceable resource for this source.",
							SourceId: "Unsupported",
						},
					}
				case "document_diagnostic_only":
					expected.Warnings = []*pbc.ParseWarning{{
						Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_INFO,
						Code:     "empty_document", Message: "Document has no resources.",
					}}
				}
				plugin := newCostParseTestPlugin(
					costParseFunc(func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
						return expected, nil
					}),
				)
				_, call := startCostParseServer(t, plugin, tr.web)
				req := costParseFixtureRequest()
				got, err := callCostParse(t, call, req)
				require.NoError(t, err)
				assert.Empty(t, got.GetResources())
				assert.True(t, proto.Equal(expected, got))
				require.NoError(t, ptest.ValidateParseResponse(req, got))
			})
		}
	}
}

func TestCostParsePartial_InvalidProviderOutputIsInternal(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pbc.ParseResponse)
	}{
		{
			"unknown_present_zero",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Replicas"} },
		},
		{"unknown_present_empty", func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Zone"} }},
		{
			"unknown_present_false",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Enabled"} },
		},
		{"unknown_present_null", func(r *pbc.ParseResponse) {
			r.Resources[0].Resource.Attributes.Fields["Future"] = structpb.NewNullValue()
			r.Resources[0].UnknownProperties = []string{"Future"}
		}},
		{"unknown_null_ancestor", func(r *pbc.ParseResponse) {
			r.Resources[0].Resource.Attributes.Fields["Sizing"] = structpb.NewNullValue()
		}},
		{"unknown_scalar_ancestor", func(r *pbc.ParseResponse) {
			r.Resources[0].Resource.Attributes.Fields["Sizing"] = structpb.NewNumberValue(0)
		}},
		{"unknown_list_ancestor", func(r *pbc.ParseResponse) {
			r.Resources[0].Resource.Attributes.Fields["Sizing"] = structpb.NewListValue(&structpb.ListValue{})
		}},
		{
			"unknown_duplicate",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Absent", "Absent"} },
		},
		{
			"unknown_empty_segment",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Sizing..Cpu"} },
		},
		{"warning_unspecified_severity", func(r *pbc.ParseResponse) {
			r.Warnings[0].Severity = pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_UNSPECIFIED
		}},
		{"warning_unknown_severity", func(r *pbc.ParseResponse) { r.Warnings[0].Severity = 99 }},
		{"warning_empty_code", func(r *pbc.ParseResponse) { r.Warnings[0].Code = "" }},
		{"warning_empty_message", func(r *pbc.ParseResponse) { r.Warnings[0].Message = "" }},
		{"skipped_without_source", func(r *pbc.ParseResponse) { r.Warnings[1].SourceId = "" }},
		{"duplicate_source", func(r *pbc.ParseResponse) {
			r.Resources = append(r.Resources, proto.Clone(r.GetResources()[0]).(*pbc.ParsedResource))
		}},
	}
	for _, tr := range allocTransports {
		for _, tc := range cases {
			t.Run(tr.name+"/"+tc.name, func(t *testing.T) {
				resp := costParsePartialFixture(t)
				tc.change(resp)
				before := proto.Clone(resp)
				plugin := newCostParseTestPlugin(
					costParseFunc(func(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error) {
						return resp, nil
					}),
				)
				_, call := startCostParseServer(t, plugin, tr.web)
				_, err := callCostParse(t, call, costParseFixtureRequest())
				code, _ := allocWireCode(t, err, tr.web)
				assert.Equal(t, codes.Internal, code, "provider contract failures must not be blamed on request")
				assert.True(t, proto.Equal(before, resp), "invalid provider response was mutated")
			})
		}
	}
}
