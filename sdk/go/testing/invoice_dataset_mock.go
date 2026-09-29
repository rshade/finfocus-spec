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

// MockInvoiceDatasetSource is the reference producer for
// SupplementalDatasetService.GetBillingPeriods and GetInvoiceDetails. It
// serves fixed lists, filtered by the request window and paged, and passes
// RunInvoiceDatasetConformance. It is safe for concurrent use.
//
// It is a separate type rather than a MockPlugin method so that MockPlugin's
// inferred capabilities and served services do not change.
type MockInvoiceDatasetSource struct {
	periods []*pbc.BillingPeriod
	details []*pbc.InvoiceDetail
}

// NewMockInvoiceDatasetSource returns a source serving deep copies of periods
// and details in the given order. It returns an error, and no source, if any
// record is nil, fails its dataset validator, repeats a billing-period identity
// (invoice_issuer_name, billing_period_start), or repeats an invoice_detail_id.
func NewMockInvoiceDatasetSource(
	periods []*pbc.BillingPeriod, details []*pbc.InvoiceDetail,
) (*MockInvoiceDatasetSource, error) {
	copiedPeriods, err := copyBillingPeriods(periods)
	if err != nil {
		return nil, err
	}
	copiedDetails, err := copyInvoiceDetails(details)
	if err != nil {
		return nil, err
	}
	return &MockInvoiceDatasetSource{periods: copiedPeriods, details: copiedDetails}, nil
}

func copyBillingPeriods(periods []*pbc.BillingPeriod) ([]*pbc.BillingPeriod, error) {
	seen := make(map[billingPeriodKey]int, len(periods))
	copies := make([]*pbc.BillingPeriod, 0, len(periods))
	for i, p := range periods {
		if err := ValidateBillingPeriod(p); err != nil {
			return nil, fmt.Errorf("billing_periods[%d]: %w", i, err)
		}
		key := billingPeriodIdentity(p)
		if first, dup := seen[key]; dup {
			return nil, fmt.Errorf(
				"billing_periods[%d]: duplicates invoice_issuer_name %q and billing_period_start of billing_periods[%d]",
				i,
				p.GetInvoiceIssuerName(),
				first,
			)
		}
		seen[key] = i
		copies = append(copies, proto.CloneOf(p))
	}
	return copies, nil
}

func copyInvoiceDetails(details []*pbc.InvoiceDetail) ([]*pbc.InvoiceDetail, error) {
	seen := make(map[string]int, len(details))
	copies := make([]*pbc.InvoiceDetail, 0, len(details))
	for i, d := range details {
		if err := ValidateInvoiceDetail(d); err != nil {
			return nil, fmt.Errorf("invoice_details[%d]: %w", i, err)
		}
		id := d.GetInvoiceDetailId()
		if first, dup := seen[id]; dup {
			return nil, fmt.Errorf("invoice_details[%d]: duplicates invoice_detail_id %q of invoice_details[%d]",
				i, id, first)
		}
		seen[id] = i
		copies = append(copies, proto.CloneOf(d))
	}
	return copies, nil
}

// GetBillingPeriods validates req, keeps the periods matching its window, and
// returns the requested page with the matching total. Returned records are
// copies. Invalid requests and malformed page tokens fail with
// codes.InvalidArgument.
func (m *MockInvoiceDatasetSource) GetBillingPeriods(
	_ context.Context, req *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error) {
	if err := ValidateGetBillingPeriodsRequest(req); err != nil {
		return nil, err
	}
	matching := make([]*pbc.BillingPeriod, 0, len(m.periods))
	for _, p := range m.periods {
		if BillingPeriodMatchesWindow(p, req.GetStart(), req.GetEnd()) {
			matching = append(matching, p)
		}
	}
	page, next, total, err := PaginateBillingPeriods(matching, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	out := make([]*pbc.BillingPeriod, len(page))
	for i, p := range page {
		out[i] = proto.CloneOf(p)
	}
	return &pbc.GetBillingPeriodsResponse{BillingPeriods: out, NextPageToken: next, TotalCount: total}, nil
}

// GetInvoiceDetails validates req, keeps the lines whose billing period matches
// the window, and returns the requested page with the matching total. Returned
// records are copies.
func (m *MockInvoiceDatasetSource) GetInvoiceDetails(
	_ context.Context, req *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error) {
	if err := ValidateGetInvoiceDetailsRequest(req); err != nil {
		return nil, err
	}
	matching := make([]*pbc.InvoiceDetail, 0, len(m.details))
	for _, d := range m.details {
		if InvoiceDetailMatchesWindow(d, req.GetStart(), req.GetEnd()) {
			matching = append(matching, d)
		}
	}
	page, next, total, err := PaginateInvoiceDetails(matching, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}
	out := make([]*pbc.InvoiceDetail, len(page))
	for i, d := range page {
		out[i] = proto.CloneOf(d)
	}
	return &pbc.GetInvoiceDetailsResponse{InvoiceDetails: out, NextPageToken: next, TotalCount: total}, nil
}
