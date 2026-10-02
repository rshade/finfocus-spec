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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func newCommitmentHarness(t *testing.T) *plugintesting.ContractCommitmentHarness {
	t.Helper()
	source, err := plugintesting.NewMockContractCommitmentSource(nil)
	if err != nil {
		t.Fatalf("NewMockContractCommitmentSource: %v", err)
	}
	return plugintesting.NewContractCommitmentHarness(source)
}

func TestBufconnHarness(t *testing.T) {
	t.Run("serves registered RPC", func(t *testing.T) {
		h := newCommitmentHarness(t)
		h.Start(t)
		defer h.Stop()
		if _, err := h.Client().GetContractCommitments(context.Background(),
			&pbc.GetContractCommitmentsRequest{}); err != nil {
			t.Fatalf("GetContractCommitments: %v", err)
		}
		_, err := h.Client().GetBillingPeriods(context.Background(), &pbc.GetBillingPeriodsRequest{})
		if status.Code(err) != codes.Unimplemented {
			t.Errorf("GetBillingPeriods code = %v, want Unimplemented", status.Code(err))
		}
	})

	t.Run("stop twice", func(t *testing.T) {
		h := newCommitmentHarness(t)
		h.Start(t)
		h.Stop()
		h.Stop()
	})

	t.Run("stop without start", func(t *testing.T) {
		h := newCommitmentHarness(t)
		if h.Client() != nil {
			t.Error("client before Start = non-nil, want nil")
		}
		h.Stop()
	})
}
