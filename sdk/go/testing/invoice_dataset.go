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
	"errors"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

var (
	// ErrInvalidBillingPeriodsRequest is wrapped by every
	// ValidateGetBillingPeriodsRequest and PaginateBillingPeriods failure.
	ErrInvalidBillingPeriodsRequest = errors.New("invalid billing periods request")
	// ErrInvalidBillingPeriodsResponse is wrapped by every
	// ValidateGetBillingPeriodsResponse failure.
	ErrInvalidBillingPeriodsResponse = errors.New("invalid billing periods response")
	// ErrInvalidInvoiceDetailsRequest is wrapped by every
	// ValidateGetInvoiceDetailsRequest and PaginateInvoiceDetails failure.
	ErrInvalidInvoiceDetailsRequest = errors.New("invalid invoice details request")
	// ErrInvalidInvoiceDetailsResponse is wrapped by every
	// ValidateGetInvoiceDetailsResponse failure.
	ErrInvalidInvoiceDetailsResponse = errors.New("invalid invoice details response")
)

// ValidateGetBillingPeriodsRequest returns nil if req is a well-formed
// GetBillingPeriodsRequest. The rules are the rules of
// ValidateGetContractCommitmentsRequest. Failures wrap
// ErrInvalidBillingPeriodsRequest, start with its text, and carry
// codes.InvalidArgument. It does not allocate on valid input.
func ValidateGetBillingPeriodsRequest(req *pbc.GetBillingPeriodsRequest) error {
	if req == nil {
		return validateDatasetRequest(ErrInvalidBillingPeriodsRequest, true, nil, nil, 0)
	}
	return validateDatasetRequest(
		ErrInvalidBillingPeriodsRequest,
		false,
		req.GetStart(),
		req.GetEnd(),
		req.GetPageSize(),
	)
}

// ValidateGetInvoiceDetailsRequest returns nil if req is a well-formed
// GetInvoiceDetailsRequest. The rules are the rules of
// ValidateGetBillingPeriodsRequest. Failures wrap
// ErrInvalidInvoiceDetailsRequest. It does not allocate on valid input.
func ValidateGetInvoiceDetailsRequest(req *pbc.GetInvoiceDetailsRequest) error {
	if req == nil {
		return validateDatasetRequest(ErrInvalidInvoiceDetailsRequest, true, nil, nil, 0)
	}
	return validateDatasetRequest(
		ErrInvalidInvoiceDetailsRequest,
		false,
		req.GetStart(),
		req.GetEnd(),
		req.GetPageSize(),
	)
}

// BillingPeriodMatchesWindow reports whether p's [billing_period_start,
// billing_period_end) overlaps the half-open window [start, end). An unset
// period bound is open-ended. When start or end is nil there is no window and
// every non-nil period matches. A nil period never matches. A period that ends
// exactly at the window start does not match, and neither does one that starts
// exactly at the window end. It does not allocate.
func BillingPeriodMatchesWindow(p *pbc.BillingPeriod, start, end *timestamppb.Timestamp) bool {
	if p == nil {
		return false
	}
	return periodOverlapsWindow(p.GetBillingPeriodStart(), p.GetBillingPeriodEnd(), start, end)
}

// InvoiceDetailMatchesWindow reports whether d's billing period overlaps the
// half-open window [start, end), using the same overlap rule as
// BillingPeriodMatchesWindow. A nil detail never matches. It does not allocate.
func InvoiceDetailMatchesWindow(d *pbc.InvoiceDetail, start, end *timestamppb.Timestamp) bool {
	if d == nil {
		return false
	}
	return periodOverlapsWindow(d.GetBillingPeriodStart(), d.GetBillingPeriodEnd(), start, end)
}

func periodOverlapsWindow(periodStart, periodEnd, start, end *timestamppb.Timestamp) bool {
	if start == nil || end == nil {
		return true
	}
	if periodStart != nil && !timestampBefore(periodStart, end) {
		return false
	}
	if periodEnd != nil && !timestampAfter(periodEnd, start) {
		return false
	}
	return true
}

// PaginateBillingPeriods returns one page of an already filtered, stably
// ordered list. The token, total, and error rules match
// PaginateContractCommitments, and failures wrap ErrInvalidBillingPeriodsRequest.
func PaginateBillingPeriods(
	periods []*pbc.BillingPeriod, pageSize int32, pageToken string,
) ([]*pbc.BillingPeriod, string, int32, error) {
	return paginateRecords(periods, pageSize, pageToken, ErrInvalidBillingPeriodsRequest)
}

// PaginateInvoiceDetails returns one page of an already filtered, stably
// ordered list. Failures wrap ErrInvalidInvoiceDetailsRequest.
func PaginateInvoiceDetails(
	details []*pbc.InvoiceDetail, pageSize int32, pageToken string,
) ([]*pbc.InvoiceDetail, string, int32, error) {
	return paginateRecords(details, pageSize, pageToken, ErrInvalidInvoiceDetailsRequest)
}

// ValidateGetBillingPeriodsResponse returns nil if resp is a valid answer to
// req. Each billing period passes ValidateBillingPeriod, overlaps the window,
// and the pair (invoice_issuer_name, billing_period_start) is unique.
// total_count is not negative and not less than the page length. A nil req is
// treated as an empty request. Failures wrap ErrInvalidBillingPeriodsResponse.
// It does not allocate on valid pages of up to 64 records.
func ValidateGetBillingPeriodsResponse(
	req *pbc.GetBillingPeriodsRequest, resp *pbc.GetBillingPeriodsResponse,
) error {
	var list []*pbc.BillingPeriod
	var total int32
	if resp != nil {
		list = resp.GetBillingPeriods()
		total = resp.GetTotalCount()
	}
	start, end := req.GetStart(), req.GetEnd()
	return validatePagedResponse(resp == nil, len(list), req.GetPageSize(), "billing periods",
		ErrInvalidBillingPeriodsResponse, total,
		func(i int) error { return validateResponseBillingPeriod(i, list[i], start, end) },
		func() error { return checkDuplicateBillingPeriods(list) },
	)
}

func validateResponseBillingPeriod(i int, p *pbc.BillingPeriod, start, end *timestamppb.Timestamp) error {
	if p == nil {
		return newInvalidArgument(ErrInvalidBillingPeriodsResponse, "billing_periods[%d]: record is nil", i)
	}
	if err := ValidateBillingPeriod(p); err != nil {
		return invalidRecord(ErrInvalidBillingPeriodsResponse, "billing_periods", i, err)
	}
	if !BillingPeriodMatchesWindow(p, start, end) {
		return newInvalidArgument(ErrInvalidBillingPeriodsResponse,
			"billing_periods[%d]: invoice_issuer_name %q does not overlap the requested window",
			i, p.GetInvoiceIssuerName())
	}
	return nil
}

func checkDuplicateBillingPeriods(list []*pbc.BillingPeriod) error {
	if len(list) <= pairwiseDuplicateLimit {
		for i := 1; i < len(list); i++ {
			for j := range i {
				if sameBillingPeriodIdentity(list[i], list[j]) {
					return duplicateBillingPeriod(i, j, list[i])
				}
			}
		}
		return nil
	}
	seen := make(map[billingPeriodKey]int, len(list))
	for i, p := range list {
		key := billingPeriodIdentity(p)
		if first, dup := seen[key]; dup {
			return duplicateBillingPeriod(i, first, p)
		}
		seen[key] = i
	}
	return nil
}

type billingPeriodKey struct {
	issuer  string
	seconds int64
	nanos   int32
}

func billingPeriodIdentity(p *pbc.BillingPeriod) billingPeriodKey {
	start := p.GetBillingPeriodStart()
	return billingPeriodKey{issuer: p.GetInvoiceIssuerName(), seconds: start.GetSeconds(), nanos: start.GetNanos()}
}

func sameBillingPeriodIdentity(a, b *pbc.BillingPeriod) bool {
	return billingPeriodIdentity(a) == billingPeriodIdentity(b)
}

func duplicateBillingPeriod(i, first int, p *pbc.BillingPeriod) error {
	return newInvalidArgument(ErrInvalidBillingPeriodsResponse,
		"billing_periods[%d]: duplicates invoice_issuer_name %q and billing_period_start of billing_periods[%d]",
		i, p.GetInvoiceIssuerName(), first)
}

// ValidateGetInvoiceDetailsResponse returns nil if resp is a valid answer to
// req. Each line passes ValidateInvoiceDetail, its billing period overlaps the
// window, and invoice_detail_id is unique. total_count is not negative and not
// less than the page length. Failures wrap ErrInvalidInvoiceDetailsResponse.
// It does not allocate on valid pages of up to 64 records.
func ValidateGetInvoiceDetailsResponse(
	req *pbc.GetInvoiceDetailsRequest, resp *pbc.GetInvoiceDetailsResponse,
) error {
	var list []*pbc.InvoiceDetail
	var total int32
	if resp != nil {
		list = resp.GetInvoiceDetails()
		total = resp.GetTotalCount()
	}
	start, end := req.GetStart(), req.GetEnd()
	return validatePagedResponse(resp == nil, len(list), req.GetPageSize(), "invoice details",
		ErrInvalidInvoiceDetailsResponse, total,
		func(i int) error { return validateResponseInvoiceDetail(i, list[i], start, end) },
		func() error { return checkDuplicateInvoiceDetails(list) },
	)
}

func validateResponseInvoiceDetail(i int, d *pbc.InvoiceDetail, start, end *timestamppb.Timestamp) error {
	if d == nil {
		return newInvalidArgument(ErrInvalidInvoiceDetailsResponse, "invoice_details[%d]: record is nil", i)
	}
	if err := ValidateInvoiceDetail(d); err != nil {
		return invalidRecord(ErrInvalidInvoiceDetailsResponse, "invoice_details", i, err)
	}
	if !InvoiceDetailMatchesWindow(d, start, end) {
		return newInvalidArgument(ErrInvalidInvoiceDetailsResponse,
			"invoice_details[%d]: invoice_detail_id %q does not overlap the requested window",
			i, d.GetInvoiceDetailId())
	}
	return nil
}

func checkDuplicateInvoiceDetails(list []*pbc.InvoiceDetail) error {
	if len(list) <= pairwiseDuplicateLimit {
		for i := 1; i < len(list); i++ {
			id := list[i].GetInvoiceDetailId()
			for j := range i {
				if list[j].GetInvoiceDetailId() == id {
					return duplicateInvoiceDetail(i, j, id)
				}
			}
		}
		return nil
	}
	seen := make(map[string]int, len(list))
	for i, d := range list {
		id := d.GetInvoiceDetailId()
		if first, dup := seen[id]; dup {
			return duplicateInvoiceDetail(i, first, id)
		}
		seen[id] = i
	}
	return nil
}

func duplicateInvoiceDetail(i, first int, id string) error {
	return newInvalidArgument(ErrInvalidInvoiceDetailsResponse,
		"invoice_details[%d]: duplicates invoice_detail_id %q of invoice_details[%d]", i, id, first)
}
