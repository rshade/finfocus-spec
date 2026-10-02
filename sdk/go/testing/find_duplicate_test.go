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
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func distinctKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	return keys
}

func withKey(keys []string, i int, key string) []string {
	out := append([]string(nil), keys...)
	out[i] = key
	return out
}

func identity(s string) string { return s }

func TestFindDuplicate(t *testing.T) {
	tests := []struct {
		name      string
		list      []string
		wantI     int
		wantFirst int
		wantOK    bool
	}{
		{name: "empty", list: nil},
		{name: "one record", list: []string{"a"}},
		{name: "distinct at pairwise limit", list: distinctKeys(plugintesting.PairwiseDuplicateLimitForTest)},
		{name: "distinct above pairwise limit", list: distinctKeys(plugintesting.PairwiseDuplicateLimitForTest + 1)},
		{
			name:  "duplicate at pairwise limit",
			list:  withKey(distinctKeys(plugintesting.PairwiseDuplicateLimitForTest), 40, "k7"),
			wantI: 40, wantFirst: 7, wantOK: true,
		},
		{
			name:  "duplicate above pairwise limit",
			list:  withKey(distinctKeys(plugintesting.PairwiseDuplicateLimitForTest+1), 40, "k7"),
			wantI: 40, wantFirst: 7, wantOK: true,
		},
		{
			name:  "lowest later index wins",
			list:  []string{"a", "b", "c", "b", "a"},
			wantI: 3, wantFirst: 1, wantOK: true,
		},
		{
			name:  "matched to earliest occurrence",
			list:  []string{"x", "a", "y", "a", "a"},
			wantI: 3, wantFirst: 1, wantOK: true,
		},
		{
			name:  "empty string key",
			list:  []string{"", "a", ""},
			wantI: 2, wantFirst: 0, wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, first, ok := plugintesting.FindDuplicateForTest(tt.list, identity)
			if i != tt.wantI || first != tt.wantFirst || ok != tt.wantOK {
				t.Errorf("findDuplicate = (%d, %d, %v), want (%d, %d, %v)",
					i, first, ok, tt.wantI, tt.wantFirst, tt.wantOK)
			}
		})
	}
}

func TestFindDuplicateCompositeKey(t *testing.T) {
	const s = int64(1735689600)
	period := func(issuer string, seconds int64, nanos int32, endSeconds int64) *pbc.BillingPeriod {
		return &pbc.BillingPeriod{
			InvoiceIssuerName:  issuer,
			BillingPeriodStart: &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos},
			BillingPeriodEnd:   &timestamppb.Timestamp{Seconds: endSeconds},
		}
	}

	_, _, ok := plugintesting.FindDuplicateForTest(
		[]*pbc.BillingPeriod{period("A", s, 0, s+10), period("A", s, 0, s+20)},
		plugintesting.BillingPeriodIdentityForTest,
	)
	if !ok {
		t.Error("same issuer and start with a different end: want duplicate")
	}

	_, _, ok = plugintesting.FindDuplicateForTest(
		[]*pbc.BillingPeriod{period("A", s, 0, s+10), period("A", s, 1, s+10)},
		plugintesting.BillingPeriodIdentityForTest,
	)
	if ok {
		t.Error("start differing only in nanos: want no duplicate")
	}
}

func TestFindDuplicateAllocationFree(t *testing.T) {
	commitments := make([]*pbc.ContractCommitment, plugintesting.PairwiseDuplicateLimitForTest)
	periods := make([]*pbc.BillingPeriod, plugintesting.PairwiseDuplicateLimitForTest)
	for i := range plugintesting.PairwiseDuplicateLimitForTest {
		commitments[i] = &pbc.ContractCommitment{ContractCommitmentId: fmt.Sprintf("cc-%d", i)}
		periods[i] = &pbc.BillingPeriod{
			InvoiceIssuerName:  fmt.Sprintf("issuer-%d", i),
			BillingPeriodStart: &timestamppb.Timestamp{Seconds: int64(i)},
		}
	}
	checks := map[string]func(){
		"string key": func() {
			_, _, _ = plugintesting.FindDuplicateForTest(commitments, (*pbc.ContractCommitment).GetContractCommitmentId)
		},
		"composite key": func() {
			_, _, _ = plugintesting.FindDuplicateForTest(periods, plugintesting.BillingPeriodIdentityForTest)
		},
	}
	for name, fn := range checks {
		t.Run(name, func(t *testing.T) {
			if allocs := testing.AllocsPerRun(100, fn); allocs != 0 {
				t.Errorf("%v allocs per run, want 0", allocs)
			}
		})
	}
}
