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
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func pricedEntry(kind, id string, cost float64, currency string, priced bool) *pbc.PricedResource {
	return &pbc.PricedResource{
		Resource: &pbc.ResourceDescriptor{Id: id, Tags: map[string]string{"kind": kind}},
		Cost:     cost,
		Currency: currency,
		Priced:   priced,
	}
}

func pricedNode(id string, cost float64, currency string) *pbc.PricedResource {
	return pricedEntry("node", id, cost, currency, true)
}

// requireInvalidArgument asserts err carries codes.InvalidArgument and a plain
// message (no "rpc error:" prefix).
func requireInvalidArgument(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, strings.HasPrefix(err.Error(), "rpc error:"), "message has status prefix: %s", err)
}

func TestResolveCurrency(t *testing.T) {
	tests := []struct {
		name   string
		priced []*pbc.PricedResource
		want   string
	}{
		{name: "nil", priced: nil, want: "USD"},
		{
			name:   "all empty",
			priced: []*pbc.PricedResource{pricedNode("n1", 1, ""), pricedNode("n2", 1, "")},
			want:   "USD",
		},
		{
			name: "USD empty USD",
			priced: []*pbc.PricedResource{
				pricedNode("n1", 1, "USD"), pricedNode("n2", 1, ""), pricedNode("n3", 1, "USD"),
			},
			want: "USD",
		},
		{name: "single EUR", priced: []*pbc.PricedResource{pricedNode("n1", 1, "EUR")}, want: "EUR"},
		{
			name: "unpriced currency ignored",
			priced: []*pbc.PricedResource{
				pricedEntry("node", "n1", 0, "JPY", false), pricedNode("n2", 1, "USD"),
			},
			want: "USD",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plugintesting.ResolveCurrency(tt.priced)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveCurrencyMixed(t *testing.T) {
	tests := []struct {
		name   string
		priced []*pbc.PricedResource
		want   []string
	}{
		{
			name: "USD empty EUR",
			priced: []*pbc.PricedResource{
				pricedNode("n1", 1, "USD"), pricedNode("n2", 1, ""), pricedNode("n3", 1, "EUR"),
			},
			want: []string{"EUR", "USD"},
		},
		{
			name:   "USD EUR",
			priced: []*pbc.PricedResource{pricedNode("n1", 1, "USD"), pricedNode("n2", 1, "EUR")},
			want:   []string{"EUR", "USD"},
		},
		{
			name:   "case matters",
			priced: []*pbc.PricedResource{pricedNode("n1", 1, "usd"), pricedNode("n2", 1, "USD")},
			want:   []string{"USD", "usd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := plugintesting.ResolveCurrency(tt.priced)
			requireInvalidArgument(t, err)
			require.ErrorIs(t, err, plugintesting.ErrMixedCurrency)
			assert.Contains(t, err.Error(), strings.Join(tt.want, ", "))
		})
	}
}

func TestValidateAllocateRequest(t *testing.T) {
	sharedID := &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
		pricedNode("n1", 1, "USD"),
		pricedEntry("node", "n1", 0, "", false),
	}}

	rejects := []struct {
		name    string
		req     *pbc.AllocateRequest
		wantErr error
	}{
		{name: "nil request", req: nil, wantErr: plugintesting.ErrInvalidAllocateRequest},
		{
			name:    "nil priced element",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", 1, ""), nil}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
		{
			name:    "unpriced with cost",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedEntry("node", "n1", 0.5, "", false)}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
		{
			name:    "negative cost",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", -1, "")}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
		{
			name:    "NaN cost",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", math.NaN(), "")}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
		{
			name:    "Inf cost",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", math.Inf(1), "")}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
		{
			name: "mixed currencies",
			req: &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
				pricedNode("n1", 1, "USD"), pricedNode("n2", 1, "EUR"),
			}},
			wantErr: plugintesting.ErrMixedCurrency,
		},
		{name: "duplicate kind and id", req: sharedID, wantErr: plugintesting.ErrInvalidAllocateRequest},
		{
			name:    "priced node without id",
			req:     &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("", 1, "")}},
			wantErr: plugintesting.ErrInvalidAllocateRequest,
		},
	}
	for _, tt := range rejects {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			err := plugintesting.ValidateAllocateRequest(tt.req)
			requireInvalidArgument(t, err)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}

	accepts := []struct {
		name string
		req  *pbc.AllocateRequest
	}{
		{name: "empty request", req: &pbc.AllocateRequest{}},
		{
			name: "unpriced with zero cost",
			req:  &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedEntry("node", "n1", 0, "", false)}},
		},
		{
			name: "same id different kind",
			req: &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
				pricedNode("n1", 1, ""), pricedEntry("cluster", "n1", 1, "", true),
			}},
		},
		{
			name: "USD empty USD",
			req: &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
				pricedNode("n1", 1, "USD"), pricedNode("n2", 1, ""), pricedNode("n3", 1, "USD"),
			}},
		},
		{
			name: "arbitrary usage",
			req: &pbc.AllocateRequest{Usage: []*pbc.UsageRow{
				nil,
				{Subject: map[string]string{"bogus": "x"}, Metric: "whatever", Amount: -5},
			}},
		},
	}
	for _, tt := range accepts {
		t.Run("accept/"+tt.name, func(t *testing.T) {
			require.NoError(t, plugintesting.ValidateAllocateRequest(tt.req))
		})
	}
}

func TestValidateAllocateRequestNamesIndex(t *testing.T) {
	err := plugintesting.ValidateAllocateRequest(&pbc.AllocateRequest{Priced: []*pbc.PricedResource{
		pricedNode("n1", 1, ""), pricedNode("n2", -1, ""),
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "priced[1]")
}

func allocRow(kind string, cpu, mem, total float64) *pbc.AllocationRow {
	return &pbc.AllocationRow{
		Subject:   map[string]string{"kind": kind},
		CpuCost:   cpu,
		MemCost:   mem,
		TotalCost: total,
		Currency:  "USD",
	}
}

func idleRow(node string, cpu, mem float64) *pbc.AllocationRow {
	row := allocRow("__idle__", cpu, mem, cpu+mem)
	row.Subject["node"] = node
	return row
}

func allocResponse(rows ...*pbc.AllocationRow) *pbc.AllocateResponse {
	return &pbc.AllocateResponse{
		Rows:                rows,
		EffectivePolicyJson: []byte(`{"version":1}`),
		PolicyDigest:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

func singleNodeAllocRequest(cost float64) *pbc.AllocateRequest {
	return &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", cost, "USD")}}
}

func TestCheckConservation(t *testing.T) {
	passes := []struct {
		name string
		req  *pbc.AllocateRequest
		resp *pbc.AllocateResponse
	}{
		{
			name: "balanced",
			req:  singleNodeAllocRequest(10),
			resp: allocResponse(allocRow("workload", 3, 3, 6), idleRow("n1", 2, 2)),
		},
		{
			name: "unpriced entry excluded",
			req: &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
				pricedNode("n1", 10, "USD"), pricedEntry("node", "n2", 0, "", false),
			}},
			resp: allocResponse(allocRow("workload", 3, 3, 6), idleRow("n1", 2, 2)),
		},
		{
			name: "nothing priced, zero rows",
			req:  &pbc.AllocateRequest{},
			resp: allocResponse(allocRow("workload", 0, 0, 0)),
		},
		{
			name: "within relative epsilon",
			req:  singleNodeAllocRequest(10),
			resp: allocResponse(allocRow("workload", 5, 5.000005, 10.000005)),
		},
		{
			name: "absolute floor at zero",
			req:  &pbc.AllocateRequest{},
			resp: allocResponse(allocRow("workload", 0, 5e-10, 5e-10)),
		},
		{
			name: "cluster rows with zero portions",
			req:  &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedEntry("cluster", "cp", 3, "USD", true)}},
			resp: allocResponse(allocRow("__cluster__", 0, 0, 3)),
		},
	}
	for _, tt := range passes {
		t.Run("pass/"+tt.name, func(t *testing.T) {
			require.NoError(
				t,
				plugintesting.CheckConservation(tt.req, tt.resp, plugintesting.DefaultConservationEpsilon),
			)
		})
	}

	for _, tc := range []struct {
		name  string
		total float64
		diff  float64
	}{
		{name: "overshoot", total: 10.01, diff: 0.01},
		{name: "shortfall", total: 9.99, diff: -0.01},
	} {
		t.Run("fail/"+tc.name, func(t *testing.T) {
			err := plugintesting.CheckConservation(singleNodeAllocRequest(10),
				allocResponse(allocRow("workload", 0, tc.total, tc.total)),
				plugintesting.DefaultConservationEpsilon)
			var consErr *plugintesting.ConservationError
			require.ErrorAs(t, err, &consErr)
			require.ErrorIs(t, err, plugintesting.ErrConservation)
			assert.InDelta(t, 10, consErr.Expected, 0)
			assert.InDelta(t, tc.total, consErr.Actual, 0)
			assert.InDelta(t, tc.total-10, consErr.Difference, 0)
			assert.InDelta(t, tc.diff, consErr.Difference, 1e-12)
			assert.Equal(t, "USD", consErr.Currency)
			for _, want := range []string{"10", fmt.Sprint(tc.total), fmt.Sprintf("%+g", consErr.Difference)} {
				assert.Contains(t, err.Error(), want)
			}
		})
	}

	fails := []struct {
		name    string
		req     *pbc.AllocateRequest
		resp    *pbc.AllocateResponse
		epsilon float64
	}{
		{
			name: "NaN row total", req: singleNodeAllocRequest(10),
			resp: allocResponse(allocRow("workload", 0, 0, math.NaN())), epsilon: 1e-6,
		},
		{
			name: "Inf priced cost", req: singleNodeAllocRequest(math.Inf(1)),
			resp: allocResponse(allocRow("workload", 0, 0, 10)), epsilon: 1e-6,
		},
		{name: "negative epsilon", req: singleNodeAllocRequest(10), resp: allocResponse(), epsilon: -1},
		{name: "NaN epsilon", req: singleNodeAllocRequest(10), resp: allocResponse(), epsilon: math.NaN()},
		{name: "Inf epsilon", req: singleNodeAllocRequest(10), resp: allocResponse(), epsilon: math.Inf(1)},
		{name: "nil request", req: nil, resp: allocResponse(), epsilon: 1e-6},
		{name: "nil response", req: singleNodeAllocRequest(10), resp: nil, epsilon: 1e-6},
	}
	for _, tt := range fails {
		t.Run("fail/"+tt.name, func(t *testing.T) {
			require.Error(t, plugintesting.CheckConservation(tt.req, tt.resp, tt.epsilon))
		})
	}

	t.Run("fail/mixed currencies", func(t *testing.T) {
		req := &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
			pricedNode("n1", 1, "USD"), pricedNode("n2", 1, "EUR"),
		}}
		err := plugintesting.CheckConservation(req, allocResponse(), plugintesting.DefaultConservationEpsilon)
		require.ErrorIs(t, err, plugintesting.ErrMixedCurrency)
	})
}

// validAllocPair is one priced node n1 (cost 10) and one cluster resource
// (cost 3), with a workload row, the node's idle row, and a cluster row.
func validAllocPair() (*pbc.AllocateRequest, *pbc.AllocateResponse) {
	req := &pbc.AllocateRequest{Priced: []*pbc.PricedResource{
		pricedNode("n1", 10, "USD"),
		pricedEntry("cluster", "cp", 3, "USD", true),
	}}
	return req, allocResponse(
		allocRow("workload", 3, 3, 6),
		idleRow("n1", 2, 2),
		allocRow("__cluster__", 0, 0, 3),
	)
}

func TestValidateAllocateResponse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req, resp := validAllocPair()
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})

	rejects := []struct {
		name   string
		mutate func(resp *pbc.AllocateResponse) *pbc.AllocateResponse
	}{
		{name: "nil response", mutate: func(*pbc.AllocateResponse) *pbc.AllocateResponse { return nil }},
		{name: "empty digest", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.PolicyDigest = ""
			return r
		}},
		{name: "empty effective policy", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.EffectivePolicyJson = nil
			return r
		}},
		{name: "row without kind", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			delete(r.GetRows()[0].GetSubject(), "kind")
			return r
		}},
		{name: "row with node kind", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].Subject["kind"] = "node"
			return r
		}},
		{name: "idle row without node", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			delete(r.GetRows()[1].GetSubject(), "node")
			return r
		}},
		{name: "negative cpu", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].CpuCost, r.Rows[0].MemCost = -1, 7
			return r
		}},
		{name: "negative mem", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].CpuCost, r.Rows[0].MemCost = 7, -1
			return r
		}},
		{name: "negative total", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[2].TotalCost = -3
			return r
		}},
		{name: "NaN total", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].TotalCost = math.NaN()
			return r
		}},
		{name: "total differs from portions", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0] = allocRow("workload", 1, 1, 3)
			return r
		}},
		{name: "empty row currency", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].Currency = ""
			return r
		}},
		{name: "wrong row currency", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].Currency = "EUR"
			return r
		}},
		{name: "missing idle row", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows = []*pbc.AllocationRow{r.GetRows()[0], r.GetRows()[2]}
			return r
		}},
		{name: "two idle rows", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows = append(r.Rows, idleRow("n1", 0, 0))
			return r
		}},
		{name: "method id without resource id", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].AllocatedMethodId = "proportional"
			return r
		}},
		{name: "whitespace method id without resource id", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[0].AllocatedMethodId = " "
			return r
		}},
		{name: "method and details no resource", mutate: func(r *pbc.AllocateResponse) *pbc.AllocateResponse {
			r.Rows[2].AllocatedMethodId = "proportional"
			r.Rows[2].AllocatedMethodDetails = "split by request"
			return r
		}},
	}
	for _, tt := range rejects {
		t.Run("reject/"+tt.name, func(t *testing.T) {
			req, resp := validAllocPair()
			err := plugintesting.ValidateAllocateResponse(req, tt.mutate(resp))
			require.ErrorIs(t, err, plugintesting.ErrInvalidAllocateResponse)
		})
	}

	t.Run("reject/method without resource names row and field", func(t *testing.T) {
		req, resp := validAllocPair()
		resp.Rows[1].AllocatedMethodId = "proportional"
		err := plugintesting.ValidateAllocateResponse(req, resp)
		require.ErrorIs(t, err, plugintesting.ErrInvalidAllocateResponse)
		assert.Contains(t, err.Error(), "rows[1]")
		assert.Contains(t, err.Error(), "allocated_method_id")
	})
	t.Run("accept/provenance combinations", func(t *testing.T) {
		combos := []struct{ method, details, resource string }{
			{"", "", ""},
			{"", "", "n1"},
			{"", "split by request", ""},
			{"", "split by request", "n1"},
			{"proportional", "", "n1"},
			{"proportional", "split by request", "n1"},
			{" ", "", " "},
		}
		for _, c := range combos {
			req, resp := validAllocPair()
			for _, row := range resp.GetRows() {
				row.AllocatedMethodId = c.method
				row.AllocatedMethodDetails = c.details
				row.AllocatedResourceId = c.resource
			}
			require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp), "%+v", c)
		}
	})
	t.Run("accept/total within tolerance", func(t *testing.T) {
		req, resp := validAllocPair()
		resp.Rows[0].TotalCost = 6 + 1e-12
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})
	t.Run("accept/no idle row for unpriced node", func(t *testing.T) {
		req, resp := validAllocPair()
		req.Priced = append(req.Priced, pricedEntry("node", "n2", 0, "", false))
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})
	t.Run("accept/empty currencies resolve to USD", func(t *testing.T) {
		req, resp := validAllocPair()
		for _, entry := range req.GetPriced() {
			entry.Currency = ""
		}
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})
	t.Run("reject/names row index", func(t *testing.T) {
		req, resp := validAllocPair()
		resp.Rows[2].Currency = "EUR"
		assert.ErrorContains(t, plugintesting.ValidateAllocateResponse(req, resp), "rows[2]")
	})
}

// largeAllocPair builds n priced nodes, each with one workload row and one idle row.
func largeAllocPair(n int) (*pbc.AllocateRequest, *pbc.AllocateResponse) {
	req := &pbc.AllocateRequest{Priced: make([]*pbc.PricedResource, 0, n)}
	rows := make([]*pbc.AllocationRow, 0, 2*n)
	for i := range n {
		node := fmt.Sprintf("n%d", i)
		req.Priced = append(req.Priced, pricedNode(node, 10, "USD"))
		rows = append(rows, allocRow("workload", 3, 3, 6), idleRow(node, 2, 2))
	}
	return req, allocResponse(rows...)
}

func BenchmarkCheckConservation(b *testing.B) {
	for _, n := range []int{500, 5000} {
		req, resp := largeAllocPair(n)
		b.Run(fmt.Sprintf("rows=%d", 2*n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := plugintesting.CheckConservation(
					req,
					resp,
					plugintesting.DefaultConservationEpsilon,
				); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkValidateAllocateResponse(b *testing.B) {
	for _, n := range []int{500, 5000} {
		req, resp := largeAllocPair(n)
		b.Run(fmt.Sprintf("rows=%d", 2*n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := plugintesting.ValidateAllocateResponse(req, resp); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestValidateAllocateResponseProvenanceAddsNoAllocations(t *testing.T) {
	req, plain := largeAllocPair(64)
	_, tagged := largeAllocPair(64)
	for _, row := range tagged.GetRows() {
		row.AllocatedMethodId = "proportional"
		row.AllocatedMethodDetails = "split by request"
		row.AllocatedResourceId = "n0"
	}
	measure := func(resp *pbc.AllocateResponse) float64 {
		return testing.AllocsPerRun(50, func() {
			if err := plugintesting.ValidateAllocateResponse(req, resp); err != nil {
				t.Fatal(err)
			}
		})
	}
	assert.InDelta(t, measure(plain), measure(tagged), 0, "provenance must not add allocations")
}

func allocWindow() (*timestamppb.Timestamp, *timestamppb.Timestamp) {
	return &timestamppb.Timestamp{Seconds: 1_790_000_000}, &timestamppb.Timestamp{Seconds: 1_790_086_400}
}

func TestValidateAllocateRequestWindow(t *testing.T) {
	start, end := allocWindow()
	base := func() *pbc.AllocateRequest {
		return &pbc.AllocateRequest{Priced: []*pbc.PricedResource{pricedNode("n1", 10, "USD")}}
	}

	t.Run("no window is valid", func(t *testing.T) {
		require.NoError(t, plugintesting.ValidateAllocateRequest(base()))
	})
	t.Run("full window is valid", func(t *testing.T) {
		req := base()
		req.Start, req.End = start, end
		require.NoError(t, plugintesting.ValidateAllocateRequest(req))
	})
	t.Run("start equal to end is valid", func(t *testing.T) {
		req := base()
		req.Start, req.End = start, start
		require.NoError(t, plugintesting.ValidateAllocateRequest(req))
	})

	rejects := []struct {
		name       string
		start, end *timestamppb.Timestamp
		wantMsg    string
	}{
		{name: "only start", start: start, wantMsg: "start and end must be set together"},
		{name: "only end", end: end, wantMsg: "start and end must be set together"},
		{name: "start after end", start: end, end: start, wantMsg: "start is after end"},
		{
			name:    "start after end by nanos",
			start:   &timestamppb.Timestamp{Seconds: 1_790_000_000, Nanos: 2},
			end:     &timestamppb.Timestamp{Seconds: 1_790_000_000, Nanos: 1},
			wantMsg: "start is after end",
		},
	}
	for _, tt := range rejects {
		t.Run(tt.name, func(t *testing.T) {
			req := base()
			req.Start, req.End = tt.start, tt.end
			err := plugintesting.ValidateAllocateRequest(req)
			requireInvalidArgument(t, err)
			require.ErrorIs(t, err, plugintesting.ErrInvalidAllocateRequest)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestValidateAllocateResponseWindowEcho(t *testing.T) {
	start, end := allocWindow()
	windowed := func() (*pbc.AllocateRequest, *pbc.AllocateResponse) {
		req, resp := validAllocPair()
		req.Start, req.End = start, end
		return req, resp
	}

	t.Run("missing echo is accepted for older allocators", func(t *testing.T) {
		req, resp := windowed()
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})
	t.Run("matching echo is accepted", func(t *testing.T) {
		req, resp := windowed()
		resp.Start = &timestamppb.Timestamp{Seconds: start.GetSeconds()}
		resp.End = &timestamppb.Timestamp{Seconds: end.GetSeconds()}
		require.NoError(t, plugintesting.ValidateAllocateResponse(req, resp))
	})

	rejects := []struct {
		name      string
		unwindow  bool
		respStart *timestamppb.Timestamp
		respEnd   *timestamppb.Timestamp
		wantField string
	}{
		{name: "different start", respStart: end, respEnd: end, wantField: "start"},
		{
			name:      "end differs by nanos",
			respStart: start,
			respEnd:   &timestamppb.Timestamp{Seconds: end.GetSeconds(), Nanos: 1},
			wantField: "end",
		},
		{name: "only start echoed", respStart: start, wantField: "end"},
		{name: "window the request did not have", unwindow: true, respStart: start, respEnd: end, wantField: "start"},
	}
	for _, tt := range rejects {
		t.Run(tt.name, func(t *testing.T) {
			req, resp := windowed()
			if tt.unwindow {
				req.Start, req.End = nil, nil
			}
			resp.Start, resp.End = tt.respStart, tt.respEnd
			err := plugintesting.ValidateAllocateResponse(req, resp)
			require.ErrorIs(t, err, plugintesting.ErrInvalidAllocateResponse)
			assert.Contains(t, err.Error(), tt.wantField)
		})
	}
}
