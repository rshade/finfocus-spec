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

package testing

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// MockContractCommitmentSource is the reference producer for
// SupplementalDatasetService.GetContractCommitments. It serves a fixed list of
// commitments, filtered by the request window and paged, and passes
// RunContractCommitmentConformance. It is safe for concurrent use.
//
// It is a separate type rather than a MockPlugin method so that MockPlugin's
// inferred capabilities and served services do not change.
type MockContractCommitmentSource struct {
	commitments []*pbc.ContractCommitment
}

// NewMockContractCommitmentSource returns a source serving deep copies of
// commitments in the given order. It returns an error, and no source, if any
// commitment is nil, fails ValidateContractCommitment, or repeats a
// contract_commitment_id, so the source never serves data that fails the SDK's
// own validators.
func NewMockContractCommitmentSource(commitments []*pbc.ContractCommitment) (*MockContractCommitmentSource, error) {
	seen := make(map[string]int, len(commitments))
	copies := make([]*pbc.ContractCommitment, 0, len(commitments))
	for i, c := range commitments {
		if err := ValidateContractCommitment(c); err != nil {
			return nil, fmt.Errorf("commitments[%d]: %w", i, err)
		}
		id := c.GetContractCommitmentId()
		if first, dup := seen[id]; dup {
			return nil, fmt.Errorf("commitments[%d]: duplicates contract_commitment_id %q of commitments[%d]",
				i, id, first)
		}
		seen[id] = i
		copies = append(copies, proto.CloneOf(c))
	}
	return &MockContractCommitmentSource{commitments: copies}, nil
}

// GetContractCommitments validates req with ValidateGetContractCommitmentsRequest,
// keeps the commitments matching its window, and returns the requested page
// (PaginateContractCommitments) with the matching total. Returned records are
// copies. Invalid requests and malformed page tokens fail with
// codes.InvalidArgument.
func (m *MockContractCommitmentSource) GetContractCommitments(
	_ context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	if err := ValidateGetContractCommitmentsRequest(req); err != nil {
		return nil, err
	}
	matching := make([]*pbc.ContractCommitment, 0, len(m.commitments))
	for _, c := range m.commitments {
		if ContractCommitmentMatchesWindow(c, req.GetStart(), req.GetEnd()) {
			matching = append(matching, c)
		}
	}
	page, next, total, err := PaginateContractCommitments(matching, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	out := make([]*pbc.ContractCommitment, len(page))
	for i, c := range page {
		out[i] = proto.CloneOf(c)
	}
	return &pbc.GetContractCommitmentsResponse{Commitments: out, NextPageToken: next, TotalCount: total}, nil
}
