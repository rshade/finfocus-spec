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
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func billingPeriod(issuer string, start, end time.Time) *pbc.BillingPeriod {
	created := ts(start)
	return &pbc.BillingPeriod{
		BillingPeriodStart:       ts(start),
		BillingPeriodEnd:         ts(end),
		BillingPeriodStatus:      pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED,
		InvoiceIssuerName:        issuer,
		BillingPeriodCreated:     created,
		BillingPeriodLastUpdated: created,
	}
}

func invoiceLine(id string, start, end time.Time) *pbc.InvoiceDetail {
	created := ts(start)
	return &pbc.InvoiceDetail{
		InvoiceDetailId:          id,
		InvoiceId:                "invoice-1",
		InvoiceIssuerName:        "Example Issuer",
		BillingAccountId:         "account-1",
		BillingPeriodStart:       ts(start),
		BillingPeriodEnd:         ts(end),
		BilledCost:               10,
		BillingCurrency:          "USD",
		ChargeCategory:           pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		InvoiceIssueStatus:       pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_ISSUED,
		InvoiceDetailCreated:     created,
		InvoiceDetailLastUpdated: created,
		PaymentTerms:             "Net 30",
		ReferenceInvoiceId:       "invoice-1",
	}
}

func periodWindow() (time.Time, time.Time) {
	return date(2025, 6, 1), date(2025, 7, 1)
}

func TestBillingPeriodMatchesWindow(t *testing.T) {
	start, end := periodWindow()
	windowStart, windowEnd := ts(start), ts(end)
	tests := []struct {
		name  string
		from  time.Time
		until time.Time
		want  bool
	}{
		{name: "ends at window start", from: date(2025, 5, 1), until: start, want: false},
		{name: "starts at window end", from: end, until: date(2025, 8, 1), want: false},
		{name: "overlaps the start", from: date(2025, 5, 15), until: date(2025, 6, 15), want: true},
		{name: "inside", from: date(2025, 6, 10), until: date(2025, 6, 20), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := billingPeriod("Issuer", tt.from, tt.until)
			assert.Equal(t, tt.want, plugintesting.BillingPeriodMatchesWindow(p, windowStart, windowEnd))
			line := invoiceLine("detail-1", tt.from, tt.until)
			assert.Equal(t, tt.want, plugintesting.InvoiceDetailMatchesWindow(line, windowStart, windowEnd))
		})
	}
	open := billingPeriod("Issuer", start, end)
	assert.True(t, plugintesting.BillingPeriodMatchesWindow(open, nil, nil))
	assert.False(t, plugintesting.BillingPeriodMatchesWindow(nil, windowStart, windowEnd))
	assert.False(t, plugintesting.InvoiceDetailMatchesWindow(nil, windowStart, windowEnd))

	openEnd := proto.CloneOf(open)
	openEnd.BillingPeriodEnd = nil
	assert.True(t, plugintesting.BillingPeriodMatchesWindow(openEnd, windowStart, windowEnd))
	startsAtEnd := proto.CloneOf(open)
	startsAtEnd.BillingPeriodStart = windowEnd
	startsAtEnd.BillingPeriodEnd = nil
	assert.False(t, plugintesting.BillingPeriodMatchesWindow(startsAtEnd, windowStart, windowEnd))
}

func TestValidateInvoiceDatasetRequests(t *testing.T) {
	start := ts(date(2025, 6, 1))
	onlyStart := &pbc.GetBillingPeriodsRequest{Start: start}
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsRequest(onlyStart),
		plugintesting.ErrInvalidBillingPeriodsRequest, "both be set")
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsRequest(nil),
		plugintesting.ErrInvalidBillingPeriodsRequest, "request is nil")
	negative := &pbc.GetInvoiceDetailsRequest{PageSize: -1}
	assertInvalidArgument(t, plugintesting.ValidateGetInvoiceDetailsRequest(negative),
		plugintesting.ErrInvalidInvoiceDetailsRequest, "page_size")
	require.NoError(t, plugintesting.ValidateGetBillingPeriodsRequest(&pbc.GetBillingPeriodsRequest{PageSize: 1001}))
	require.NoError(t, plugintesting.ValidateGetInvoiceDetailsRequest(&pbc.GetInvoiceDetailsRequest{}))
}

func TestValidateGetBillingPeriodsResponse(t *testing.T) {
	start, end := periodWindow()
	good := billingPeriod("Issuer", date(2025, 6, 1), date(2025, 7, 1))
	req := &pbc.GetBillingPeriodsRequest{Start: ts(start), End: ts(end), PageSize: 2}
	ok := &pbc.GetBillingPeriodsResponse{BillingPeriods: []*pbc.BillingPeriod{good}, TotalCount: 1}
	require.NoError(t, plugintesting.ValidateGetBillingPeriodsResponse(req, ok))

	dup := &pbc.GetBillingPeriodsResponse{
		BillingPeriods: []*pbc.BillingPeriod{good, proto.CloneOf(good)}, TotalCount: 2,
	}
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsResponse(req, dup),
		plugintesting.ErrInvalidBillingPeriodsResponse, "duplicates")

	outside := billingPeriod("Issuer", date(2024, 1, 1), date(2024, 2, 1))
	miss := &pbc.GetBillingPeriodsResponse{BillingPeriods: []*pbc.BillingPeriod{outside}, TotalCount: 1}
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsResponse(req, miss),
		plugintesting.ErrInvalidBillingPeriodsResponse, "does not overlap")

	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsResponse(req, nil),
		plugintesting.ErrInvalidBillingPeriodsResponse, "response is nil")

	otherEnd := proto.CloneOf(good)
	otherEnd.BillingPeriodEnd = ts(date(2025, 8, 1))
	sameStart := &pbc.GetBillingPeriodsResponse{
		BillingPeriods: []*pbc.BillingPeriod{good, otherEnd}, TotalCount: 2,
	}
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsResponse(req, sameStart),
		plugintesting.ErrInvalidBillingPeriodsResponse, "duplicates")

	shifted := proto.CloneOf(good)
	shifted.BillingPeriodStart = ts(date(2025, 6, 1).Add(time.Nanosecond))
	distinct := &pbc.GetBillingPeriodsResponse{
		BillingPeriods: []*pbc.BillingPeriod{good, shifted}, TotalCount: 2,
	}
	require.NoError(t, plugintesting.ValidateGetBillingPeriodsResponse(req, distinct))

	short := &pbc.GetBillingPeriodsResponse{BillingPeriods: []*pbc.BillingPeriod{good}, TotalCount: 0}
	assertInvalidArgument(t, plugintesting.ValidateGetBillingPeriodsResponse(req, short),
		plugintesting.ErrInvalidBillingPeriodsResponse, "total_count")
}

func TestValidateGetInvoiceDetailsResponse(t *testing.T) {
	start, end := periodWindow()
	good := invoiceLine("detail-1", date(2025, 6, 1), date(2025, 7, 1))
	zero := 0.0
	good.PaymentCurrency = "EUR"
	good.PaymentCurrencyBilledCost = &zero
	req := &pbc.GetInvoiceDetailsRequest{Start: ts(start), End: ts(end)}
	require.NoError(t, plugintesting.ValidateGetInvoiceDetailsResponse(req, &pbc.GetInvoiceDetailsResponse{
		InvoiceDetails: []*pbc.InvoiceDetail{good}, TotalCount: 1,
	}))

	half := proto.CloneOf(good)
	half.PaymentCurrencyBilledCost = nil
	assertInvalidArgument(t, plugintesting.ValidateGetInvoiceDetailsResponse(req, &pbc.GetInvoiceDetailsResponse{
		InvoiceDetails: []*pbc.InvoiceDetail{half}, TotalCount: 1,
	}), plugintesting.ErrInvalidInvoiceDetailsResponse, "payment_currency")

	refund := proto.CloneOf(good)
	refund.ChargeCategory = pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_REFUND
	refund.PaymentCurrency = ""
	refund.PaymentCurrencyBilledCost = nil
	assertInvalidArgument(t, plugintesting.ValidateGetInvoiceDetailsResponse(req, &pbc.GetInvoiceDetailsResponse{
		InvoiceDetails: []*pbc.InvoiceDetail{refund}, TotalCount: 1,
	}), plugintesting.ErrInvalidInvoiceDetailsResponse, "REFUND")
}

func TestMockInvoiceDatasetSourcePagesAndFilters(t *testing.T) {
	periods := []*pbc.BillingPeriod{
		billingPeriod("A", date(2025, 5, 1), date(2025, 6, 1)),
		billingPeriod("A", date(2025, 6, 1), date(2025, 7, 1)),
		billingPeriod("B", date(2025, 6, 15), date(2025, 8, 1)),
	}
	lines := []*pbc.InvoiceDetail{
		invoiceLine("keep", date(2025, 6, 1), date(2025, 7, 1)),
		invoiceLine("drop", date(2024, 1, 1), date(2024, 2, 1)),
	}
	source, err := plugintesting.NewMockInvoiceDatasetSource(periods, lines)
	require.NoError(t, err)

	start, end := periodWindow()
	periodsResp, err := source.GetBillingPeriods(context.Background(), &pbc.GetBillingPeriodsRequest{
		Start: ts(start), End: ts(end), PageSize: 1,
	})
	require.NoError(t, err)
	require.Len(t, periodsResp.GetBillingPeriods(), 1)
	assert.Equal(t, "A", periodsResp.GetBillingPeriods()[0].GetInvoiceIssuerName())
	assert.Equal(t, int32(2), periodsResp.GetTotalCount())
	assert.NotEmpty(t, periodsResp.GetNextPageToken())

	next, err := source.GetBillingPeriods(context.Background(), &pbc.GetBillingPeriodsRequest{
		Start: ts(start), End: ts(end), PageSize: 1, PageToken: periodsResp.GetNextPageToken(),
	})
	require.NoError(t, err)
	require.Len(t, next.GetBillingPeriods(), 1)
	assert.Equal(t, "B", next.GetBillingPeriods()[0].GetInvoiceIssuerName())
	assert.Empty(t, next.GetNextPageToken())

	details, err := source.GetInvoiceDetails(context.Background(), &pbc.GetInvoiceDetailsRequest{
		Start: ts(start), End: ts(end),
	})
	require.NoError(t, err)
	require.Len(t, details.GetInvoiceDetails(), 1)
	assert.Equal(t, "keep", details.GetInvoiceDetails()[0].GetInvoiceDetailId())
	assert.Equal(t, int32(1), details.GetTotalCount())

	_, err = source.GetInvoiceDetails(context.Background(), &pbc.GetInvoiceDetailsRequest{PageToken: "!not-a-token!"})
	assertInvalidArgument(t, err, plugintesting.ErrInvalidInvoiceDetailsRequest, "malformed")

	past := base64.StdEncoding.EncodeToString([]byte("5"))
	pastResp, err := source.GetBillingPeriods(context.Background(), &pbc.GetBillingPeriodsRequest{PageToken: past})
	require.NoError(t, err)
	assert.Empty(t, pastResp.GetBillingPeriods())
	assert.Empty(t, pastResp.GetNextPageToken())
	assert.Equal(t, int32(3), pastResp.GetTotalCount())

	negative := base64.StdEncoding.EncodeToString([]byte("-1"))
	_, err = source.GetBillingPeriods(context.Background(), &pbc.GetBillingPeriodsRequest{PageToken: negative})
	assertInvalidArgument(t, err, plugintesting.ErrInvalidBillingPeriodsRequest, "malformed")

	dup := []*pbc.BillingPeriod{billingPeriod("A", date(2025, 6, 1), date(2025, 7, 1)), periods[1]}
	_, err = plugintesting.NewMockInvoiceDatasetSource(dup, nil)
	require.Error(t, err)
}

func TestMockInvoiceDatasetConformance(t *testing.T) {
	source, err := plugintesting.NewMockInvoiceDatasetSource(
		[]*pbc.BillingPeriod{
			billingPeriod("A", date(2025, 1, 1), date(2025, 2, 1)),
			billingPeriod("B", date(2024, 1, 1), date(2024, 2, 1)),
		},
		[]*pbc.InvoiceDetail{
			invoiceLine("line-1", date(2025, 1, 1), date(2025, 2, 1)),
			invoiceLine("line-2", date(1890, 1, 1), date(1890, 2, 1)),
		},
	)
	require.NoError(t, err)
	plugintesting.RunInvoiceDatasetConformance(t, source)
}

func TestInvoiceDatasetRPCValidatorsAllocationFree(t *testing.T) {
	p := billingPeriod("Issuer", date(2025, 6, 1), date(2025, 7, 1))
	line := invoiceLine("detail-1", date(2025, 6, 1), date(2025, 7, 1))
	req := &pbc.GetBillingPeriodsRequest{Start: ts(date(2025, 6, 1)), End: ts(date(2025, 7, 1)), PageSize: 50}
	detailReq := &pbc.GetInvoiceDetailsRequest{Start: req.GetStart(), End: req.GetEnd(), PageSize: 50}
	periodResp := &pbc.GetBillingPeriodsResponse{BillingPeriods: []*pbc.BillingPeriod{p}, TotalCount: 1}
	detailResp := &pbc.GetInvoiceDetailsResponse{InvoiceDetails: []*pbc.InvoiceDetail{line}, TotalCount: 1}
	periods64 := make([]*pbc.BillingPeriod, 64)
	lines64 := make([]*pbc.InvoiceDetail, 64)
	for i := range 64 {
		periods64[i] = billingPeriod(fmt.Sprintf("Issuer-%d", i), date(2025, 6, 1), date(2025, 7, 1))
		lines64[i] = invoiceLine(fmt.Sprintf("detail-%d", i), date(2025, 6, 1), date(2025, 7, 1))
	}
	req64 := &pbc.GetBillingPeriodsRequest{Start: req.GetStart(), End: req.GetEnd(), PageSize: 64}
	detailReq64 := &pbc.GetInvoiceDetailsRequest{Start: req.GetStart(), End: req.GetEnd(), PageSize: 64}
	periodResp64 := &pbc.GetBillingPeriodsResponse{BillingPeriods: periods64, TotalCount: 64}
	detailResp64 := &pbc.GetInvoiceDetailsResponse{InvoiceDetails: lines64, TotalCount: 64}
	checks := map[string]func(){
		"billing request": func() { _ = plugintesting.ValidateGetBillingPeriodsRequest(req) },
		"invoice request": func() { _ = plugintesting.ValidateGetInvoiceDetailsRequest(detailReq) },
		"billing window": func() {
			_ = plugintesting.BillingPeriodMatchesWindow(p, req.GetStart(), req.GetEnd())
		},
		"invoice window": func() {
			_ = plugintesting.InvoiceDetailMatchesWindow(line, req.GetStart(), req.GetEnd())
		},
		"billing response":    func() { _ = plugintesting.ValidateGetBillingPeriodsResponse(req, periodResp) },
		"invoice response":    func() { _ = plugintesting.ValidateGetInvoiceDetailsResponse(detailReq, detailResp) },
		"billing response 64": func() { _ = plugintesting.ValidateGetBillingPeriodsResponse(req64, periodResp64) },
		"invoice response 64": func() {
			_ = plugintesting.ValidateGetInvoiceDetailsResponse(detailReq64, detailResp64)
		},
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
