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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestCostParseFreshnessAdvisory(t *testing.T) {
	cases := []struct {
		name   string
		expiry *timestamppb.Timestamp
	}{
		{"absent", nil},
		{"expired", &timestamppb.Timestamp{Seconds: 1, Nanos: 123456789}},
		{"future", &timestamppb.Timestamp{Seconds: 253402300799, Nanos: 987654321}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := pt.NewMockCostParser(pt.WithParseExpiration(func() *timestamppb.Timestamp { return tc.expiry }))
			req := costParseMockRequest()
			resp, err := parser.Parse(context.Background(), req)
			require.NoError(t, err)
			require.NoError(
				t,
				pt.ValidateParseResponse(req, resp),
				"expiry is advisory, including already-expired values",
			)
			require.True(t, proto.Equal(tc.expiry, resp.GetExpiresAt()), "expiry lost seconds or nanoseconds")
		})
	}
}

func TestCostParseFreshnessPureInput(t *testing.T) {
	parser := pt.NewMockCostParser()
	req := costParsePartialRequest(
		"resource First\nunknown First Sizing.Cpu\nresource Second\nskip Other\nnotice Input only\n",
	)
	var baseline *pbc.ParseResponse
	for range 100 {
		resp, err := parser.Parse(context.Background(), req)
		require.NoError(t, err)
		require.NoError(t, pt.ValidateParseResponse(req, resp))
		require.Nil(t, resp.GetExpiresAt(), "pure input parsing must not invent an expiration")
		if baseline == nil {
			baseline = proto.CloneOf(resp)
		}
		require.True(t, proto.Equal(baseline, resp), "resources, warnings, or ordering changed")
	}
}

func TestCostParseFreshnessAdvancement(t *testing.T) {
	expiry := &timestamppb.Timestamp{Seconds: 1700000000, Nanos: 100}
	parser := pt.NewMockCostParser(pt.WithParseExpiration(func() *timestamppb.Timestamp { return expiry }))
	req := costParsePartialRequest("resource First\nunknown First Sizing.Cpu\nskip Other\nnotice Dependency result\n")
	var baseline *pbc.ParseResponse
	var previous *timestamppb.Timestamp
	for range 100 {
		expiry.Nanos++
		resp, err := parser.Parse(context.Background(), req)
		require.NoError(t, err)
		require.NoError(t, pt.ValidateParseResponse(req, resp))
		require.True(t, proto.Equal(expiry, resp.GetExpiresAt()))
		if previous != nil {
			require.Greater(t, resp.GetExpiresAt().GetNanos(), previous.GetNanos(), "advisory expiration may advance")
		}
		previous = proto.CloneOf(resp.GetExpiresAt())
		stable := proto.CloneOf(resp)
		stable.ExpiresAt = nil
		if baseline == nil {
			baseline = stable
		}
		require.True(t, proto.Equal(baseline, stable), "expiration advancement changed semantic results")
		// The callback owns this timestamp; mutating the response must not change it.
		resp.GetExpiresAt().Nanos = 999
		require.NotEqual(t, int32(999), expiry.GetNanos(), "parser shared callback-owned expiration state")
	}
}
