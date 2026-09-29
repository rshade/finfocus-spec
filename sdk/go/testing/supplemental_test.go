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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

// commitment returns a valid SPEND commitment whose commitment period is
// [start, end); zero times leave that bound unset.
func commitment(id string, start, end time.Time) *pbc.ContractCommitment {
	discount := 0.0
	upfront := 0.0
	created := ts(date(2025, 1, 1))
	c := &pbc.ContractCommitment{
		ContractCommitmentId:            id,
		ContractId:                      "contract-1",
		ContractCommitmentCategory:      pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND,
		ContractCommitmentType:          "Savings Plan",
		ContractCommitmentCost:          1200,
		BillingCurrency:                 "USD",
		ContractCommitmentApplicability: `{"IsGlobalScope":true}`,
		ContractCommitmentBenefitCategory: pbc.
			FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_DISCOUNT,
		ContractCommitmentCreated:            created,
		ContractCommitmentDiscountPercentage: &discount,
		ContractCommitmentDurationType:       "1 Year",
		ContractCommitmentFulfillmentInterval: pbc.
			FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_MONTHLY,
		ContractCommitmentLastUpdated: created,
		ContractCommitmentLifecycleStatus: pbc.
			FocusContractCommitmentLifecycleStatus_FOCUS_CONTRACT_COMMITMENT_LIFECYCLE_STATUS_ACTIVE,
		ContractCommitmentModel: pbc.FocusContractCommitmentModel_FOCUS_CONTRACT_COMMITMENT_MODEL_CONTINUOUS,
		ContractCommitmentOfferCategory: pbc.
			FocusContractCommitmentOfferCategory_FOCUS_CONTRACT_COMMITMENT_OFFER_CATEGORY_PUBLIC,
		ContractCommitmentPaymentInterval: pbc.
			FocusContractCommitmentPaymentInterval_FOCUS_CONTRACT_COMMITMENT_PAYMENT_INTERVAL_MONTHLY,
		ContractCommitmentPaymentModel: pbc.
			FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_NO_UPFRONT,
		ContractCommitmentPaymentUpfrontPercentage: &upfront,
		InvoiceIssuerName:                          "Example Issuer",
		ServiceProviderName:                        "Example Provider",
	}
	if !start.IsZero() {
		c.ContractCommitmentPeriodStart = ts(start)
	}
	if !end.IsZero() {
		c.ContractCommitmentPeriodEnd = ts(end)
	}
	return c
}

func commitments(n int) []*pbc.ContractCommitment {
	out := make([]*pbc.ContractCommitment, n)
	for i := range out {
		out[i] = commitment(fmt.Sprintf("cc-%04d", i), date(2025, 1, 1), date(2026, 1, 1))
	}
	return out
}

// assertInvalidArgument checks the 052 error pattern: a plain message without
// the "rpc error:" prefix, the given sentinel, and codes.InvalidArgument.
func assertInvalidArgument(t *testing.T, err error, sentinel error, contains string) {
	t.Helper()
	require.Error(t, err)
	require.ErrorIs(t, err, sentinel)
	assert.NotContains(t, err.Error(), "rpc error:")
	assert.Contains(t, err.Error(), contains)
	st, ok := status.FromError(err)
	require.True(t, ok, "error must carry a gRPC status")
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestValidateContractCommitment(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *pbc.ContractCommitment)
		wantMsg string
	}{
		{name: "valid spend", mutate: func(*pbc.ContractCommitment) {}},
		{name: "valid usage with zero cost", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentCategory = pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE
			c.ContractCommitmentCost = 0
			c.ContractCommitmentQuantity = 8760
			c.ContractCommitmentUnit = "Hours"
		}},
		{name: "valid equal period bounds", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentPeriodEnd = c.GetContractCommitmentPeriodStart()
		}},
		{name: "missing id", mutate: func(c *pbc.ContractCommitment) { c.ContractCommitmentId = "" },
			wantMsg: "contract_commitment_id is required"},
		{name: "missing contract id", mutate: func(c *pbc.ContractCommitment) { c.ContractId = "" },
			wantMsg: "contract_id is required"},
		{name: "missing currency", mutate: func(c *pbc.ContractCommitment) { c.BillingCurrency = "" },
			wantMsg: "billing_currency is required"},
		{name: "unspecified category", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentCategory = pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_UNSPECIFIED
		}, wantMsg: "contract_commitment_category must be SPEND or USAGE"},
		{name: "invalid currency", mutate: func(c *pbc.ContractCommitment) { c.BillingCurrency = "XXX1" },
			wantMsg: "billing_currency must be a valid ISO 4217 currency code"},
		{name: "inverted commitment period", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentPeriodEnd = ts(date(2024, 1, 1))
		}, wantMsg: "contract_commitment_period_end"},
		{name: "inverted contract period", mutate: func(c *pbc.ContractCommitment) {
			c.ContractPeriodStart = ts(date(2026, 1, 1))
			c.ContractPeriodEnd = ts(date(2025, 1, 1))
		}, wantMsg: "contract_period_end"},
		{name: "negative cost", mutate: func(c *pbc.ContractCommitment) { c.ContractCommitmentCost = -1 },
			wantMsg: "contract_commitment_cost must be non-negative"},
		{name: "negative quantity", mutate: func(c *pbc.ContractCommitment) { c.ContractCommitmentQuantity = -1 },
			wantMsg: "contract_commitment_quantity must be non-negative"},
		{name: "NaN cost", mutate: func(c *pbc.ContractCommitment) { c.ContractCommitmentCost = math.NaN() },
			wantMsg: "contract_commitment_cost must be finite"},
		{name: "infinite quantity", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentQuantity = math.Inf(1)
		}, wantMsg: "contract_commitment_quantity must be finite"},
		{name: "usage may omit billing currency", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentCategory = pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE
			c.BillingCurrency = ""
		}},
		{name: "discount zero is present", mutate: func(c *pbc.ContractCommitment) {
			zero := 0.0
			c.ContractCommitmentDiscountPercentage = &zero
		}},
		{name: "discount required for discount benefit", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentDiscountPercentage = nil
		}, wantMsg: "contract_commitment_discount_percentage is required"},
		{name: "availability rejects discount", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentBenefitCategory = pbc.
				FocusContractCommitmentBenefitCategory_FOCUS_CONTRACT_COMMITMENT_BENEFIT_CATEGORY_AVAILABILITY
		}, wantMsg: "contract_commitment_discount_percentage must be null"},
		{name: "applicability must be an object", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentApplicability = `["global"]`
		}, wantMsg: "contract_commitment_applicability must be a JSON object"},
		{name: "full period requires discontinuous", mutate: func(c *pbc.ContractCommitment) {
			c.ContractCommitmentFulfillmentInterval = pbc.
				FocusContractCommitmentFulfillmentInterval_FOCUS_CONTRACT_COMMITMENT_FULFILLMENT_INTERVAL_FULL_PERIOD
		}, wantMsg: "contract_commitment_model must be DISCONTINUOUS"},
		{name: "all upfront requires one-time and 1", mutate: func(c *pbc.ContractCommitment) {
			one := 1.0
			c.ContractCommitmentPaymentModel = pbc.
				FocusContractCommitmentPaymentModel_FOCUS_CONTRACT_COMMITMENT_PAYMENT_MODEL_ALL_UPFRONT
			c.ContractCommitmentPaymentUpfrontPercentage = &one
		}, wantMsg: "contract_commitment_payment_interval must be ONE_TIME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := commitment("cc-1", date(2025, 1, 1), date(2026, 1, 1))
			tt.mutate(c)
			err := plugintesting.ValidateContractCommitment(c)
			if tt.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitment, tt.wantMsg)
			assert.True(t, strings.HasPrefix(err.Error(), tt.wantMsg),
				"message %q must be the builder's message without a prefix", err.Error())
		})
	}

	t.Run("nil", func(t *testing.T) {
		err := plugintesting.ValidateContractCommitment(nil)
		assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitment, "contract commitment is nil")
	})
}

func TestValidateGetContractCommitmentsRequest(t *testing.T) {
	start := date(2025, 1, 1)
	tests := []struct {
		name    string
		req     *pbc.GetContractCommitmentsRequest
		wantMsg string
	}{
		{name: "empty", req: &pbc.GetContractCommitmentsRequest{}},
		{name: "window", req: &pbc.GetContractCommitmentsRequest{
			Start: ts(start), End: ts(start.Add(time.Hour)), PageSize: 10, PageToken: "abc",
		}},
		{name: "page size above maximum is clamped, not rejected",
			req: &pbc.GetContractCommitmentsRequest{PageSize: plugintesting.MaxPageSize + 1}},
		{name: "nil", wantMsg: "request is nil"},
		{name: "only start", req: &pbc.GetContractCommitmentsRequest{Start: ts(start)},
			wantMsg: "start and end must both be set or both be unset"},
		{name: "only end", req: &pbc.GetContractCommitmentsRequest{End: ts(start)},
			wantMsg: "start and end must both be set or both be unset"},
		{name: "end equals start", req: &pbc.GetContractCommitmentsRequest{Start: ts(start), End: ts(start)},
			wantMsg: "end must be after start"},
		{name: "end before start", req: &pbc.GetContractCommitmentsRequest{
			Start: ts(start), End: ts(start.Add(-time.Hour)),
		}, wantMsg: "end must be after start"},
		{name: "invalid start timestamp", req: &pbc.GetContractCommitmentsRequest{
			Start: &timestamppb.Timestamp{Seconds: 1, Nanos: -1}, End: ts(start),
		}, wantMsg: "start:"},
		{name: "invalid end timestamp", req: &pbc.GetContractCommitmentsRequest{
			Start: ts(start), End: &timestamppb.Timestamp{Seconds: math.MaxInt64},
		}, wantMsg: "end:"},
		{name: "negative page size", req: &pbc.GetContractCommitmentsRequest{PageSize: -1},
			wantMsg: "page_size must not be negative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateGetContractCommitmentsRequest(tt.req)
			if tt.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitmentsRequest, tt.wantMsg)
			assert.True(t, strings.HasPrefix(err.Error(), plugintesting.ErrInvalidContractCommitmentsRequest.Error()))
		})
	}
}

func TestContractCommitmentMatchesWindow(t *testing.T) {
	ws, we := ts(date(2025, 3, 1)), ts(date(2025, 4, 1))
	contract := func(start, end time.Time) *pbc.ContractCommitment {
		c := commitment("cc", time.Time{}, time.Time{})
		c.ContractPeriodStart, c.ContractPeriodEnd = ts(start), ts(end)
		return c
	}

	tests := []struct {
		name string
		c    *pbc.ContractCommitment
		want bool
	}{
		{name: "overlaps", c: commitment("cc", date(2025, 1, 1), date(2026, 1, 1)), want: true},
		{name: "inside", c: commitment("cc", date(2025, 3, 10), date(2025, 3, 20)), want: true},
		{name: "before", c: commitment("cc", date(2024, 1, 1), date(2025, 1, 1)), want: false},
		{name: "after", c: commitment("cc", date(2025, 5, 1), date(2026, 1, 1)), want: false},
		{name: "ends at window start", c: commitment("cc", date(2025, 1, 1), date(2025, 3, 1)), want: false},
		{name: "starts at window end", c: commitment("cc", date(2025, 4, 1), date(2026, 1, 1)), want: false},
		{name: "open end", c: commitment("cc", date(2020, 1, 1), time.Time{}), want: true},
		{name: "open start", c: commitment("cc", time.Time{}, date(2025, 3, 2)), want: true},
		{name: "open start ended before", c: commitment("cc", time.Time{}, date(2025, 2, 1)), want: false},
		{name: "no bounds", c: commitment("cc", time.Time{}, time.Time{}), want: true},
		{name: "contract period fallback overlaps", c: contract(date(2025, 1, 1), date(2027, 1, 1)), want: true},
		{name: "contract period fallback misses", c: contract(date(2023, 1, 1), date(2024, 1, 1)), want: false},
		{name: "commitment period wins over contract", c: func() *pbc.ContractCommitment {
			c := contract(date(2025, 1, 1), date(2027, 1, 1))
			c.ContractCommitmentPeriodEnd = ts(date(2025, 2, 1))
			return c
		}(), want: false},
		{name: "nil commitment", c: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, plugintesting.ContractCommitmentMatchesWindow(tt.c, ws, we))
		})
	}

	t.Run("no window matches everything", func(t *testing.T) {
		c := commitment("cc", date(1990, 1, 1), date(1991, 1, 1))
		assert.True(t, plugintesting.ContractCommitmentMatchesWindow(c, nil, nil))
	})
}

func TestPaginateContractCommitments(t *testing.T) {
	all := commitments(5)

	t.Run("default page size", func(t *testing.T) {
		many := commitments(plugintesting.DefaultPageSize + 1)
		page, next, total, err := plugintesting.PaginateContractCommitments(many, 0, "")
		require.NoError(t, err)
		assert.Len(t, page, plugintesting.DefaultPageSize)
		assert.NotEmpty(t, next)
		assert.Equal(t, int32(plugintesting.DefaultPageSize+1), total)
	})

	t.Run("clamps to maximum", func(t *testing.T) {
		many := commitments(plugintesting.MaxPageSize + 1)
		page, next, _, err := plugintesting.PaginateContractCommitments(many, plugintesting.MaxPageSize+500, "")
		require.NoError(t, err)
		assert.Len(t, page, plugintesting.MaxPageSize)
		assert.NotEmpty(t, next)
	})

	t.Run("walks every page once", func(t *testing.T) {
		var got []string
		token := ""
		pages := 0
		for {
			page, next, total, err := plugintesting.PaginateContractCommitments(all, 2, token)
			require.NoError(t, err)
			assert.Equal(t, int32(5), total)
			for _, c := range page {
				got = append(got, c.GetContractCommitmentId())
			}
			pages++
			if next == "" {
				break
			}
			token = next
		}
		assert.Equal(t, 3, pages)
		assert.Equal(t, []string{"cc-0000", "cc-0001", "cc-0002", "cc-0003", "cc-0004"}, got)
	})

	t.Run("token past the end", func(t *testing.T) {
		_, next, _, err := plugintesting.PaginateContractCommitments(all, 5, "")
		require.NoError(t, err)
		assert.Empty(t, next)
		_, token, _, err := plugintesting.PaginateContractCommitments(commitments(20), 10, "")
		require.NoError(t, err)
		page, next, total, err := plugintesting.PaginateContractCommitments(all, 10, token)
		require.NoError(t, err)
		assert.Empty(t, page)
		assert.Empty(t, next)
		assert.Equal(t, int32(5), total)
	})

	t.Run("empty list", func(t *testing.T) {
		page, next, total, err := plugintesting.PaginateContractCommitments(nil, 0, "")
		require.NoError(t, err)
		assert.Empty(t, page)
		assert.Empty(t, next)
		assert.Zero(t, total)
	})

	for _, token := range []string{"!not-a-token!", "bm90LWEtbnVtYmVy", "LTE="} {
		t.Run("malformed token "+token, func(t *testing.T) {
			_, _, _, err := plugintesting.PaginateContractCommitments(all, 2, token)
			assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitmentsRequest, "page_token")
		})
	}

	t.Run("negative page size", func(t *testing.T) {
		_, _, _, err := plugintesting.PaginateContractCommitments(all, -1, "")
		assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitmentsRequest, "page_size")
	})
}

func TestValidateGetContractCommitmentsResponse(t *testing.T) {
	window := &pbc.GetContractCommitmentsRequest{Start: ts(date(2025, 3, 1)), End: ts(date(2025, 4, 1))}
	valid := func() *pbc.GetContractCommitmentsResponse {
		return &pbc.GetContractCommitmentsResponse{Commitments: commitments(3), TotalCount: 3}
	}

	tests := []struct {
		name    string
		req     *pbc.GetContractCommitmentsRequest
		resp    func() *pbc.GetContractCommitmentsResponse
		wantMsg string
		also    error
	}{
		{name: "valid", req: &pbc.GetContractCommitmentsRequest{}, resp: valid},
		{name: "valid in window", req: window, resp: valid},
		{name: "valid empty", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse { return &pbc.GetContractCommitmentsResponse{} }},
		{name: "nil request treated as empty", resp: valid},
		{name: "valid page above pairwise limit", req: &pbc.GetContractCommitmentsRequest{PageSize: 200},
			resp: func() *pbc.GetContractCommitmentsResponse {
				return &pbc.GetContractCommitmentsResponse{Commitments: commitments(200), TotalCount: 200}
			}},
		{name: "nil response", req: &pbc.GetContractCommitmentsRequest{},
			resp:    func() *pbc.GetContractCommitmentsResponse { return nil },
			wantMsg: "response is nil"},
		{name: "exceeds default page size", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				return &pbc.GetContractCommitmentsResponse{Commitments: commitments(51), TotalCount: 51}
			}, wantMsg: "exceed page size 50"},
		{name: "exceeds requested page size", req: &pbc.GetContractCommitmentsRequest{PageSize: 2},
			resp: valid, wantMsg: "exceed page size 2"},
		{name: "nil record", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.Commitments[1] = nil
				return r
			}, wantMsg: "commitments[1]: record is nil"},
		{name: "invalid record", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.Commitments[2].BillingCurrency = ""
				return r
			}, wantMsg: "commitments[2]: billing_currency is required", also: plugintesting.ErrInvalidContractCommitment},
		{name: "duplicate ids", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.GetCommitments()[2].ContractCommitmentId = r.GetCommitments()[0].GetContractCommitmentId()
				return r
			}, wantMsg: "commitments[2]: duplicates contract_commitment_id \"cc-0000\" of commitments[0]"},
		{name: "duplicate ids above pairwise limit", req: &pbc.GetContractCommitmentsRequest{PageSize: 100},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := &pbc.GetContractCommitmentsResponse{Commitments: commitments(100), TotalCount: 100}
				r.Commitments[99].ContractCommitmentId = "cc-0007"
				return r
			}, wantMsg: "commitments[99]: duplicates contract_commitment_id \"cc-0007\" of commitments[7]"},
		{name: "outside window", req: window,
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.Commitments[1] = commitment("cc-old", date(2020, 1, 1), date(2021, 1, 1))
				return r
			}, wantMsg: "commitments[1]: contract_commitment_id \"cc-old\" does not overlap the requested window"},
		{name: "negative total", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.TotalCount = -1
				return r
			}, wantMsg: "total_count must not be negative"},
		{name: "total below page length", req: &pbc.GetContractCommitmentsRequest{},
			resp: func() *pbc.GetContractCommitmentsResponse {
				r := valid()
				r.TotalCount = 2
				return r
			}, wantMsg: "total_count 2 is less than the 3 commitments returned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugintesting.ValidateGetContractCommitmentsResponse(tt.req, tt.resp())
			if tt.wantMsg == "" {
				require.NoError(t, err)
				return
			}
			assertInvalidArgument(t, err, plugintesting.ErrInvalidContractCommitmentsResponse, tt.wantMsg)
			if tt.also != nil {
				assert.ErrorIs(t, err, tt.also, "error must also wrap %v", tt.also)
			}
		})
	}
}
