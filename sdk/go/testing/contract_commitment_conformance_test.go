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
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// conformanceFixture is ten 2025 commitments plus one with no period bounds.
func conformanceFixture() []*pbc.ContractCommitment {
	out := commitments(10)
	return append(out, commitment("cc-open", time.Time{}, time.Time{}))
}

func newFixtureSource(t *testing.T) *plugintesting.MockContractCommitmentSource {
	t.Helper()
	source, err := plugintesting.NewMockContractCommitmentSource(conformanceFixture())
	require.NoError(t, err)
	return source
}

func TestNewMockContractCommitmentSourceRejectsBadConfig(t *testing.T) {
	invalid := commitment("cc-bad", date(2025, 1, 1), date(2026, 1, 1))
	invalid.BillingCurrency = ""

	tests := []struct {
		name    string
		input   []*pbc.ContractCommitment
		wantMsg string
	}{
		{name: "nil entry", input: []*pbc.ContractCommitment{commitment("a", time.Time{}, time.Time{}), nil},
			wantMsg: "commitments[1]"},
		{name: "invalid entry", input: []*pbc.ContractCommitment{invalid}, wantMsg: "billing_currency is required"},
		{name: "duplicate id", input: []*pbc.ContractCommitment{
			commitment("dup", time.Time{}, time.Time{}), commitment("dup", time.Time{}, time.Time{}),
		}, wantMsg: `duplicates contract_commitment_id "dup"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := plugintesting.NewMockContractCommitmentSource(tt.input)
			require.Error(t, err)
			assert.Nil(t, source)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestMockContractCommitmentSource(t *testing.T) {
	ctx := context.Background()

	t.Run("copies its input", func(t *testing.T) {
		input := commitments(2)
		source, err := plugintesting.NewMockContractCommitmentSource(input)
		require.NoError(t, err)
		input[0].BillingCurrency = ""

		resp, err := source.GetContractCommitments(ctx, &pbc.GetContractCommitmentsRequest{})
		require.NoError(t, err)
		assert.Equal(t, "USD", resp.GetCommitments()[0].GetBillingCurrency())

		resp.GetCommitments()[0].BillingCurrency = ""
		again, err := source.GetContractCommitments(ctx, &pbc.GetContractCommitmentsRequest{})
		require.NoError(t, err)
		assert.Equal(t, "USD", again.GetCommitments()[0].GetBillingCurrency())
	})

	t.Run("filters by window", func(t *testing.T) {
		old := commitment("cc-2020", date(2020, 1, 1), date(2021, 1, 1))
		source, err := plugintesting.NewMockContractCommitmentSource(append(commitments(3), old))
		require.NoError(t, err)

		req := &pbc.GetContractCommitmentsRequest{Start: ts(date(2025, 6, 1)), End: ts(date(2025, 7, 1))}
		resp, err := source.GetContractCommitments(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.GetCommitments(), 3)
		assert.Equal(t, int32(3), resp.GetTotalCount())
		require.NoError(t, plugintesting.ValidateGetContractCommitmentsResponse(req, resp))
	})

	t.Run("pages", func(t *testing.T) {
		source := newFixtureSource(t)
		req := &pbc.GetContractCommitmentsRequest{PageSize: 4}
		resp, err := source.GetContractCommitments(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.GetCommitments(), 4)
		assert.NotEmpty(t, resp.GetNextPageToken())
		assert.Equal(t, int32(11), resp.GetTotalCount())
	})

	t.Run("rejects invalid requests", func(t *testing.T) {
		source := newFixtureSource(t)
		for _, req := range []*pbc.GetContractCommitmentsRequest{
			{Start: ts(date(2025, 1, 1))},
			{Start: ts(date(2025, 2, 1)), End: ts(date(2025, 1, 1))},
			{PageSize: -1},
			{PageToken: "!not-a-token!"},
		} {
			_, err := source.GetContractCommitments(ctx, req)
			assert.Equal(t, codes.InvalidArgument, status.Code(err), "request %v", req)
		}
	})
}

func TestRunContractCommitmentConformance_ReferenceProducer(t *testing.T) {
	plugintesting.RunContractCommitmentConformance(t, newFixtureSource(t))
}

func TestRunContractCommitmentConformance_EmptySource(t *testing.T) {
	source, err := plugintesting.NewMockContractCommitmentSource(nil)
	require.NoError(t, err)
	plugintesting.RunContractCommitmentConformance(t, source)
}

// brokenSource wraps the reference producer and injects one defect by
// rewriting the request before, or the response after, the inner call.
type brokenSource struct {
	inner      *plugintesting.MockContractCommitmentSource
	reversed   *plugintesting.MockContractCommitmentSource
	editReq    func(req *pbc.GetContractCommitmentsRequest)
	editResp   func(resp *pbc.GetContractCommitmentsResponse)
	reverseOn1 bool
}

func (b *brokenSource) GetContractCommitments(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	req = proto.Clone(req).(*pbc.GetContractCommitmentsRequest)
	if b.editReq != nil {
		b.editReq(req)
	}
	source := b.inner
	if b.reverseOn1 && req.GetPageSize() == 1 {
		source = b.reversed
	}
	resp, err := source.GetContractCommitments(ctx, req)
	if err != nil {
		return nil, err
	}
	if b.editResp != nil {
		b.editResp(resp)
	}
	return resp, nil
}

func TestRunContractCommitmentConformance_BrokenSources(t *testing.T) {
	reversedFixture := conformanceFixture()
	slices.Reverse(reversedFixture)
	reversed, err := plugintesting.NewMockContractCommitmentSource(reversedFixture)
	require.NoError(t, err)

	tests := []struct {
		name       string
		source     *brokenSource
		wantFailed string
	}{
		{name: "ignores window", wantFailed: "window_filter", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) { r.Start, r.End = nil, nil },
		}},
		{name: "invalid record", wantFailed: "full_walk", source: &brokenSource{
			editResp: func(r *pbc.GetContractCommitmentsResponse) {
				if len(r.GetCommitments()) > 0 {
					r.GetCommitments()[0].BillingCurrency = ""
				}
			},
		}},
		{name: "duplicates across pages", wantFailed: "full_walk", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) {
				r.PageToken = ""
				if r.GetPageSize() == plugintesting.MaxPageSize {
					r.PageSize = 3
				}
			},
			editResp: func(r *pbc.GetContractCommitmentsResponse) { r.NextPageToken = "next" },
		}},
		{name: "ignores page size", wantFailed: "stable_order", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) { r.PageSize = plugintesting.MaxPageSize },
		}},
		{name: "accepts one-bound window", wantFailed: "window_one_bound", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) {
				if (r.GetStart() == nil) != (r.GetEnd() == nil) {
					r.Start, r.End = nil, nil
				}
			},
		}},
		{name: "accepts inverted window", wantFailed: "window_inverted", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) {
				if r.GetStart() != nil && r.GetEnd() != nil && r.GetEnd().AsTime().Before(r.GetStart().AsTime()) {
					r.Start, r.End = r.GetEnd(), r.GetStart()
				}
			},
		}},
		{name: "accepts negative page size", wantFailed: "negative_page_size", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) {
				if r.GetPageSize() < 0 {
					r.PageSize = 0
				}
			},
		}},
		{name: "accepts malformed token", wantFailed: "malformed_page_token", source: &brokenSource{
			editReq: func(r *pbc.GetContractCommitmentsRequest) {
				if r.GetPageToken() == "!not-a-token!" {
					r.PageToken = ""
				}
			},
		}},
		{name: "wrong total", wantFailed: "full_walk", source: &brokenSource{
			editResp: func(r *pbc.GetContractCommitmentsResponse) { r.TotalCount++ },
		}},
		{name: "unstable order", wantFailed: "stable_order", source: &brokenSource{reverseOn1: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.source.inner = newFixtureSource(t)
			tt.source.reversed = reversed
			harness := plugintesting.NewContractCommitmentHarness(tt.source)
			harness.Start(t)
			defer harness.Stop()

			results := plugintesting.RunContractCommitmentScenariosForTest(context.Background(), harness.Client())
			require.Contains(t, results, tt.wantFailed)
			assert.Error(t, results[tt.wantFailed], "scenario %s must catch %s", tt.wantFailed, tt.name)
		})
	}
}

func TestRunContractCommitmentScenarios_ReferencePassesAll(t *testing.T) {
	harness := plugintesting.NewContractCommitmentHarness(newFixtureSource(t))
	harness.Start(t)
	defer harness.Stop()

	results := plugintesting.RunContractCommitmentScenariosForTest(context.Background(), harness.Client())
	assert.Len(t, results, 7)
	for name, err := range results {
		assert.NoError(t, err, name)
	}
}
