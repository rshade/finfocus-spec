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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func parseResponse() *pbc.ParseResponse {
	return &pbc.ParseResponse{
		Resources: []*pbc.ParsedResource{
			{
				SourceId: "Item",
				Resource: &pbc.ResourceDescriptor{Provider: "custom", ResourceType: "fixture:compute:Instance"},
			},
		},
	}
}
func TestCostParseResponse(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pbc.ParseResponse)
		valid  bool
	}{
		{"valid", func(*pbc.ParseResponse) {}, true},
		{"empty", func(r *pbc.ParseResponse) { r.Resources = nil }, true},
		{"nil entry", func(r *pbc.ParseResponse) { r.Resources[0] = nil }, false},
		{"nil descriptor", func(r *pbc.ParseResponse) { r.Resources[0].Resource = nil }, false},
		{"bad descriptor", func(r *pbc.ParseResponse) { r.Resources[0].Resource.Provider = "" }, false},
		{"empty source", func(r *pbc.ParseResponse) { r.Resources[0].SourceId = "" }, false},
		{
			"duplicate",
			func(r *pbc.ParseResponse) { r.Resources = append(r.GetResources(), r.GetResources()[0]) },
			false,
		},
		{
			"unknown missing ancestor",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Sizing.Cpu"} },
			true,
		},
		{"empty unknown", func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{""} }, false},
		{
			"duplicate unknown",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Cpu", "Cpu"} },
			false,
		},
		{
			"empty segment",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Sizing..Cpu"} },
			false,
		},
		{
			"list indexing",
			func(r *pbc.ParseResponse) { r.Resources[0].UnknownProperties = []string{"Items[0]"} },
			false,
		},
		{"nil warning", func(r *pbc.ParseResponse) { r.Warnings = []*pbc.ParseWarning{nil} }, false},
		{"document warning", func(r *pbc.ParseResponse) {
			r.Warnings = []*pbc.ParseWarning{{Severity: 1, Code: "notice", Message: "hello"}}
		}, true},
		{"skipped no source", func(r *pbc.ParseResponse) {
			r.Warnings = []*pbc.ParseWarning{{Severity: 2, Code: "resource_skipped", Message: "skip"}}
		}, false},
		{"skipped source", func(r *pbc.ParseResponse) {
			r.Warnings = []*pbc.ParseWarning{
				{Severity: 2, Code: "resource_skipped", Message: "skip", SourceId: "Other"},
			}
		}, true},
		{"bad severity", func(r *pbc.ParseResponse) {
			r.Warnings = []*pbc.ParseWarning{{Severity: 3, Code: "notice", Message: "hello"}}
		}, false},
		{
			"zero severity",
			func(r *pbc.ParseResponse) { r.Warnings = []*pbc.ParseWarning{{Code: "notice", Message: "hello"}} },
			false,
		},
		{
			"empty code",
			func(r *pbc.ParseResponse) { r.Warnings = []*pbc.ParseWarning{{Severity: 1, Message: "hello"}} },
			false,
		},
		{
			"empty message",
			func(r *pbc.ParseResponse) { r.Warnings = []*pbc.ParseWarning{{Severity: 1, Code: "notice"}} },
			false,
		},
		{
			"action zero",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_UNSPECIFIED },
			true,
		},
		{
			"action create",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_CREATE },
			true,
		},
		{
			"action update",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_UPDATE },
			true,
		},
		{
			"action delete",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_DELETE },
			true,
		},
		{
			"action replace",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_REPLACE },
			true,
		},
		{
			"action no change",
			func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction_PARSE_ACTION_NO_CHANGE },
			true,
		},
		{"action unknown number", func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction(6) }, false},
		{"action negative number", func(r *pbc.ParseResponse) { r.Resources[0].Action = pbc.ParseAction(-1) }, false},
		{"expired", func(r *pbc.ParseResponse) { r.ExpiresAt = &timestamppb.Timestamp{Seconds: 1} }, true},
		{"invalid timestamp", func(r *pbc.ParseResponse) { r.ExpiresAt = &timestamppb.Timestamp{Nanos: -1} }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := parseResponse()
			c.change(r)
			before := proto.Clone(r)
			err := pt.ValidateParseResponse(parseRequest(), r)
			require.Equal(t, c.valid, err == nil, "validation error: %v", err)
			if err != nil {
				require.ErrorIs(t, err, pt.ErrInvalidParseResponse)
			}
			require.True(t, proto.Equal(r, before), "mutated response")
		})
	}
	for name, v := range map[string]*structpb.Value{"null": structpb.NewNullValue(), "scalar": structpb.NewNumberValue(0), "list": structpb.NewListValue(&structpb.ListValue{}), "empty object": structpb.NewStructValue(&structpb.Struct{})} {
		t.Run(name, func(t *testing.T) {
			r := parseResponse()
			r.Resources[0].Resource.Attributes = &structpb.Struct{Fields: map[string]*structpb.Value{"Sizing": v}}
			r.Resources[0].UnknownProperties = []string{"Sizing.Cpu"}
			err := pt.ValidateParseResponse(parseRequest(), r)
			if (err == nil) != (name == "empty object") {
				t.Fatalf("ancestor: %v", err)
			}
			r.Resources[0].UnknownProperties = []string{"Sizing"}
			if pt.ValidateParseResponse(parseRequest(), r) == nil {
				t.Fatal("present leaf allowed")
			}
			r.Resources[0].UnknownProperties = []string{"sizing"}
			require.NoError(t, pt.ValidateParseResponse(parseRequest(), r))
		})
	}
	if pt.ValidateParseResponse(nil, parseResponse()) == nil || pt.ValidateParseResponse(parseRequest(), nil) == nil {
		t.Fatal("nil values")
	}
}
