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
	"testing"

	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func benchWindowRequest() *pbc.GetContractCommitmentsRequest {
	return &pbc.GetContractCommitmentsRequest{Start: ts(date(2025, 3, 1)), End: ts(date(2025, 4, 1))}
}

// TestContractCommitmentValidatorsAllocationFree proves the zero-allocation
// claims in the docs on valid input.
func TestContractCommitmentValidatorsAllocationFree(t *testing.T) {
	c := commitment("cc-1", date(2025, 1, 1), date(2026, 1, 1))
	req := benchWindowRequest()
	page50 := &pbc.GetContractCommitmentsResponse{Commitments: commitments(50), TotalCount: 50}
	page64 := &pbc.GetContractCommitmentsResponse{Commitments: commitments(64), TotalCount: 64}
	req64 := &pbc.GetContractCommitmentsRequest{Start: req.GetStart(), End: req.GetEnd(), PageSize: 64}

	checks := map[string]func(){
		"commitment": func() { _ = plugintesting.ValidateContractCommitment(c) },
		"request":    func() { _ = plugintesting.ValidateGetContractCommitmentsRequest(req) },
		"window": func() {
			_ = plugintesting.ContractCommitmentMatchesWindow(c, req.GetStart(), req.GetEnd())
		},
		"response 50": func() { _ = plugintesting.ValidateGetContractCommitmentsResponse(req, page50) },
		"response 64": func() { _ = plugintesting.ValidateGetContractCommitmentsResponse(req64, page64) },
	}
	for name, fn := range checks {
		t.Run(name, func(t *testing.T) {
			fn()
			if allocs := testing.AllocsPerRun(100, fn); allocs != 0 {
				t.Errorf("%s: %v allocs per run, want 0", name, allocs)
			}
		})
	}
}

func BenchmarkValidateContractCommitment(b *testing.B) {
	c := commitment("cc-1", date(2025, 1, 1), date(2026, 1, 1))
	b.ReportAllocs()
	for b.Loop() {
		if err := plugintesting.ValidateContractCommitment(c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateGetContractCommitmentsRequest(b *testing.B) {
	req := benchWindowRequest()
	b.ReportAllocs()
	for b.Loop() {
		if err := plugintesting.ValidateGetContractCommitmentsRequest(req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContractCommitmentMatchesWindow(b *testing.B) {
	c := commitment("cc-1", date(2025, 1, 1), date(2026, 1, 1))
	req := benchWindowRequest()
	b.ReportAllocs()
	for b.Loop() {
		if !plugintesting.ContractCommitmentMatchesWindow(c, req.GetStart(), req.GetEnd()) {
			b.Fatal("expected a match")
		}
	}
}

func BenchmarkValidateGetContractCommitmentsResponse(b *testing.B) {
	for _, n := range []int{50, 1000} {
		req := &pbc.GetContractCommitmentsRequest{Start: ts(date(2025, 3, 1)), End: ts(date(2025, 4, 1)),
			PageSize: int32(n)}
		resp := &pbc.GetContractCommitmentsResponse{Commitments: commitments(n), TotalCount: int32(n)}
		b.Run(fmt.Sprintf("page_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := plugintesting.ValidateGetContractCommitmentsResponse(req, resp); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPaginateContractCommitments(b *testing.B) {
	all := commitments(500)
	_, token, _, err := plugintesting.PaginateContractCommitments(all, 50, "")
	require.NoError(b, err)
	b.ReportAllocs()
	for b.Loop() {
		if _, _, _, pageErr := plugintesting.PaginateContractCommitments(all, 50, token); pageErr != nil {
			b.Fatal(pageErr)
		}
	}
}

func BenchmarkMockContractCommitmentSource(b *testing.B) {
	source, err := plugintesting.NewMockContractCommitmentSource(commitments(200))
	require.NoError(b, err)
	req := &pbc.GetContractCommitmentsRequest{Start: ts(date(2025, 3, 1)), End: ts(date(2025, 4, 1))}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, callErr := source.GetContractCommitments(ctx, req); callErr != nil {
			b.Fatal(callErr)
		}
	}
}
