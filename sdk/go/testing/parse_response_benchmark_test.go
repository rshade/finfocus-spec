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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func BenchmarkCostParseResponse(b *testing.B) {
	req := parseRequest()
	cases := []struct {
		name     string
		response *pbc.ParseResponse
		valid    bool
	}{
		{"valid", parseResponse(), true},
		{"rejected", &pbc.ParseResponse{Resources: []*pbc.ParsedResource{nil}}, false},
		{"nested_unknown", parseResponse(), true},
		{"warning_heavy", parseResponse(), true},
		{"descriptor_near_limit", parseResponse(), true},
	}
	cases[2].response.GetResources()[0].UnknownProperties = []string{"Sizing.Cpu"}
	for range 128 {
		cases[3].response.Warnings = append(
			cases[3].response.GetWarnings(),
			&pbc.ParseWarning{Severity: 2, Code: "resource_skipped", Message: "unsupported", SourceId: "Skipped"},
		)
	}
	cases[4].response.GetResources()[0].GetResource().Attributes = &structpb.Struct{
		Fields: map[string]*structpb.Value{"Data": structpb.NewStringValue(strings.Repeat("x", 65500))},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			require.Equal(b, c.valid, pt.ValidateParseResponse(req, c.response) == nil)
			before := proto.Clone(c.response)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = pt.ValidateParseResponse(req, c.response)
			}
			b.StopTimer()
			require.True(b, proto.Equal(before, c.response))
		})
	}
}
