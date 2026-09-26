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
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// Literals rather than pluginsdk constants: sdk/go/testing cannot import pluginsdk.
func workloadRow(metric string, amount float64) *pbc.UsageRow {
	return &pbc.UsageRow{
		Subject: map[string]string{"kind": "workload", "namespace": "payments", "pod": "api", "node": "n1"},
		Metric:  metric,
		Amount:  amount,
		Unit:    "core",
	}
}

func nodeRow(node, metric string) *pbc.UsageRow {
	return &pbc.UsageRow{
		Subject: map[string]string{"kind": "node", "node": node},
		Metric:  metric,
		Amount:  2,
		Unit:    "core",
	}
}

func priceableNode(id string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Id:   id,
		Tags: map[string]string{"kind": "node", "provider_id": "aws:///us-east-1a/i-1", "capacity_type": "spot"},
	}
}

func validStatsResponse() *pbc.GetStatsResponse {
	return &pbc.GetStatsResponse{
		Mode:      pbc.StatsMode_STATS_MODE_RUN_RATE,
		Rows:      []*pbc.UsageRow{workloadRow("cpu_request", 0.5), nodeRow("n1", "cpu_allocatable")},
		Priceable: []*pbc.ResourceDescriptor{priceableNode("n1")},
	}
}

// singleRow returns a valid response whose only row has the given subject.
func singleRow(subject map[string]string, amount float64) *pbc.GetStatsResponse {
	return &pbc.GetStatsResponse{
		Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
		Rows: []*pbc.UsageRow{{Subject: subject, Metric: "cpu_request", Amount: amount, Unit: "core"}},
	}
}

func TestValidateStatsResponse(t *testing.T) {
	workload := map[string]string{"kind": "workload", "pod": "a"}
	withKey := func(key, value string) map[string]string {
		subject := map[string]string{"kind": "workload"}
		subject[key] = value
		return subject
	}

	rejecting := []struct {
		name  string
		resp  *pbc.GetStatsResponse
		wants []string
	}{
		{name: "V1 nil response", resp: nil, wants: []string{"nil"}},
		{
			name:  "V2 unspecified mode",
			resp:  &pbc.GetStatsResponse{Mode: pbc.StatsMode_STATS_MODE_UNSPECIFIED},
			wants: []string{"STATS_MODE_UNSPECIFIED"},
		},
		{
			name:  "V3 missing kind",
			resp:  singleRow(map[string]string{"pod": "a"}, 1),
			wants: []string{"rows[0]", "kind"},
		},
		{name: "V4 idle kind", resp: singleRow(withKey("kind", "__idle__"), 1), wants: []string{"rows[0]", "__idle__"}},
		{
			name:  "V4 cluster kind",
			resp:  singleRow(withKey("kind", "__cluster__"), 1),
			wants: []string{"rows[0]", "__cluster__"},
		},
		{name: "V4 pvc kind", resp: singleRow(withKey("kind", "pvc"), 1), wants: []string{"rows[0]", "pvc"}},
		{
			name:  "V10 node row without node key",
			resp:  singleRow(map[string]string{"kind": "node"}, 1),
			wants: []string{"rows[0]", `"node"`},
		},
		{
			name:  "V10 node row with empty node",
			resp:  singleRow(map[string]string{"kind": "node", "node": ""}, 1),
			wants: []string{"rows[0]", `"node"`},
		},
		{
			name:  "V5 misspelled key",
			resp:  singleRow(withKey("namespcae", "x"), 1),
			wants: []string{"rows[0]", "namespcae"},
		},
		{
			name:  "V5 bare label prefix",
			resp:  singleRow(withKey("label.", "x"), 1),
			wants: []string{"rows[0]", `"label."`},
		},
		{name: "V6 negative amount", resp: singleRow(workload, -1), wants: []string{"rows[0]", "-1"}},
		{name: "V6 NaN amount", resp: singleRow(workload, math.NaN()), wants: []string{"rows[0]", "NaN"}},
		{name: "V6 infinite amount", resp: singleRow(workload, math.Inf(1)), wants: []string{"rows[0]", "Inf"}},
		{
			name: "V7 duplicate subject and metric",
			resp: &pbc.GetStatsResponse{
				Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
				Rows: []*pbc.UsageRow{workloadRow("cpu_request", 1), workloadRow("cpu_request", 2)},
			},
			wants: []string{"rows[1]", "cpu_request"},
		},
		{
			name: "V8 nil priceable entry",
			resp: func() *pbc.GetStatsResponse {
				r := validStatsResponse()
				r.Priceable = []*pbc.ResourceDescriptor{nil}
				return r
			}(),
			wants: []string{"priceable[0]", "nil"},
		},
		{
			name: "V8 empty priceable id",
			resp: func() *pbc.GetStatsResponse {
				r := validStatsResponse()
				r.Priceable = []*pbc.ResourceDescriptor{priceableNode("")}
				return r
			}(),
			wants: []string{"priceable[0]", "id"},
		},
		{
			name: "V9 priceable node without rows",
			resp: func() *pbc.GetStatsResponse {
				r := validStatsResponse()
				r.Priceable = append(r.Priceable, priceableNode("n2"))
				return r
			}(),
			wants: []string{"priceable[1]", "n2"},
		},
	}

	for _, tt := range rejecting {
		t.Run("reject "+tt.name, func(t *testing.T) {
			err := plugintesting.ValidateStatsResponse(tt.resp)
			require.ErrorIs(t, err, plugintesting.ErrInvalidStatsResponse)
			for _, want := range tt.wants {
				assert.Contains(t, err.Error(), want)
			}
		})
	}

	historical := validStatsResponse()
	historical.Mode = pbc.StatsMode_STATS_MODE_HISTORICAL
	for _, row := range historical.GetRows() {
		row.Amount *= 24
		row.Unit = "core-hours"
	}
	historical.Rows = append(historical.Rows, &pbc.UsageRow{
		Subject: map[string]string{"kind": "node", "node": "n1"},
		Metric:  "mem_allocatable",
		Amount:  172.8,
		Unit:    "GiB-hours",
	})

	accepting := []struct {
		name string
		resp *pbc.GetStatsResponse
	}{
		{name: "valid response", resp: validStatsResponse()},
		{
			name: "unpriceable Fargate-style node",
			resp: &pbc.GetStatsResponse{
				Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
				Rows: []*pbc.UsageRow{nodeRow("fargate-ip-10-0-9-9", "cpu_allocatable")},
			},
		},
		{
			name: "control plane without node rows",
			resp: func() *pbc.GetStatsResponse {
				r := validStatsResponse()
				r.Priceable = append(r.Priceable, &pbc.ResourceDescriptor{
					Id: "eks-prod", Tags: map[string]string{"kind": "cluster"},
				})
				return r
			}(),
		},
		{
			name: "priceable without kind tag",
			resp: func() *pbc.GetStatsResponse {
				r := validStatsResponse()
				r.Priceable = append(r.Priceable, &pbc.ResourceDescriptor{Id: "nat-gw-1"})
				return r
			}(),
		},
		{name: "label subject key", resp: singleRow(withKey("label.app.kubernetes.io/name", "api"), 1)},
		{name: "empty rows in run-rate", resp: &pbc.GetStatsResponse{Mode: pbc.StatsMode_STATS_MODE_RUN_RATE}},
		{
			name: "custom metric and unit",
			resp: &pbc.GetStatsResponse{
				Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
				Rows: []*pbc.UsageRow{{Subject: workload, Metric: "gpu_request", Amount: 1, Unit: "gpu"}},
			},
		},
		{
			name: "same subject with two metrics",
			resp: &pbc.GetStatsResponse{
				Mode: pbc.StatsMode_STATS_MODE_RUN_RATE,
				Rows: []*pbc.UsageRow{workloadRow("cpu_request", 1), workloadRow("mem_request", 1)},
			},
		},
		{name: "zero amount", resp: singleRow(workload, 0)},
		{name: "historical units", resp: historical},
	}

	for _, tt := range accepting {
		t.Run("accept "+tt.name, func(t *testing.T) {
			assert.NoError(t, plugintesting.ValidateStatsResponse(tt.resp))
		})
	}
}

type fixedUsageSource struct {
	resp *pbc.GetStatsResponse
	err  error
}

func (s *fixedUsageSource) GetStats(context.Context, *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	return s.resp, s.err
}

func TestUsageSourceHarness(t *testing.T) {
	t.Run("returns implementation response", func(t *testing.T) {
		want := validStatsResponse()
		h := plugintesting.NewUsageSourceHarness(&fixedUsageSource{resp: want})
		h.Start(t)
		defer h.Stop()

		got, err := h.Client().GetStats(context.Background(), &pbc.GetStatsRequest{})
		require.NoError(t, err)
		assert.True(t, proto.Equal(want, got), "harness response differs from implementation")
	})

	t.Run("preserves error code", func(t *testing.T) {
		h := plugintesting.NewUsageSourceHarness(&fixedUsageSource{
			err: status.Error(codes.PermissionDenied, "cannot list pods"),
		})
		h.Start(t)
		defer h.Stop()

		_, err := h.Client().GetStats(context.Background(), &pbc.GetStatsRequest{})
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Equal(t, "cannot list pods", status.Convert(err).Message())
	})
}

func BenchmarkValidateStatsResponse(b *testing.B) {
	const nodes, podsPerNode = 3, 20
	resp := &pbc.GetStatsResponse{Mode: pbc.StatsMode_STATS_MODE_RUN_RATE}
	for n := range nodes {
		node := fmt.Sprintf("ip-10-0-%d-5", n)
		for p := range podsPerNode {
			subject := map[string]string{
				"kind": "workload", "cluster": "c1", "namespace": "payments",
				"controller_kind": "Deployment", "controller": "api",
				"pod": fmt.Sprintf("api-%d-%d", n, p), "node": node,
			}
			resp.Rows = append(resp.Rows,
				&pbc.UsageRow{Subject: subject, Metric: "cpu_request", Amount: 0.25, Unit: "core"},
				&pbc.UsageRow{Subject: subject, Metric: "mem_request", Amount: 0.5, Unit: "GiB"},
			)
		}
		resp.Rows = append(resp.Rows, nodeRow(node, "cpu_allocatable"), nodeRow(node, "mem_allocatable"))
		resp.Priceable = append(resp.Priceable, priceableNode(node))
	}
	require.NoError(b, plugintesting.ValidateStatsResponse(resp))

	b.ReportAllocs()
	for b.Loop() {
		_ = plugintesting.ValidateStatsResponse(resp)
	}
}
