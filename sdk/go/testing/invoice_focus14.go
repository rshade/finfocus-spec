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
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// focusInvoiceGrainKeys are the FOCUS 1.4 InvoiceDetailGrain property names.
// Custom keys must use the x_ prefix instead.
//
//nolint:gochecknoglobals // Intentional zero-allocation membership set
var focusInvoiceGrainKeys = map[string]struct{}{
	"ContractId":   {},
	"RegionId":     {},
	"ResourceId":   {},
	"ResourceType": {},
	"ServiceName":  {},
	"SkuId":        {},
	"SkuMeter":     {},
	"SkuPriceId":   {},
	"SubAccountId": {},
}

var (
	// ErrInvalidBillingPeriod is wrapped by every ValidateBillingPeriod failure.
	ErrInvalidBillingPeriod = errors.New("invalid billing period")
	// ErrInvalidInvoiceDetail is wrapped by every ValidateInvoiceDetail failure.
	ErrInvalidInvoiceDetail = errors.New("invalid invoice detail")
)

func datasetError(sentinel error, format string, args ...any) error {
	return &invalidArgumentError{msg: fmt.Sprintf(format, args...), wrapped: sentinel}
}

func billingPeriodError(format string, args ...any) error {
	return datasetError(ErrInvalidBillingPeriod, format, args...)
}

func invoiceError(format string, args ...any) error {
	return datasetError(ErrInvalidInvoiceDetail, format, args...)
}

// ValidateBillingPeriod returns nil if p satisfies the FOCUS 1.4 Billing Period
// rules that pluginsdk.BillingPeriodBuilder.Build enforces. Failures wrap
// ErrInvalidBillingPeriod, name the column, and carry codes.InvalidArgument.
// It does not allocate on valid input. One-way status changes are not checked.
func ValidateBillingPeriod(p *pbc.BillingPeriod) error {
	if p == nil {
		return billingPeriodError("billing period is nil")
	}
	if err := requireTimestamp(ErrInvalidBillingPeriod, "billing_period_start", p.GetBillingPeriodStart()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidBillingPeriod, "billing_period_end", p.GetBillingPeriodEnd()); err != nil {
		return err
	}
	if err := requireAfter(ErrInvalidBillingPeriod, "billing_period_end", "billing_period_start",
		p.GetBillingPeriodEnd(), p.GetBillingPeriodStart()); err != nil {
		return err
	}
	if err := validateBillingPeriodStatus(p.GetBillingPeriodStatus()); err != nil {
		return err
	}
	if err := requireText(ErrInvalidBillingPeriod, "invoice_issuer_name", p.GetInvoiceIssuerName()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidBillingPeriod, "billing_period_created",
		p.GetBillingPeriodCreated()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidBillingPeriod, "billing_period_last_updated",
		p.GetBillingPeriodLastUpdated()); err != nil {
		return err
	}
	return requireNotBefore(ErrInvalidBillingPeriod, "billing_period_last_updated", "billing_period_created",
		p.GetBillingPeriodLastUpdated(), p.GetBillingPeriodCreated())
}

// ValidateInvoiceDetail returns nil if d satisfies the per-record FOCUS 1.4
// Invoice Detail rules that pluginsdk.InvoiceDetailBuilder.Build enforces.
// Failures wrap ErrInvalidInvoiceDetail. It does not allocate on valid input.
// Invoice sums, joins, and status transitions are not checked.
func ValidateInvoiceDetail(d *pbc.InvoiceDetail) error {
	if d == nil {
		return invoiceError("invoice detail is nil")
	}
	if err := validateInvoiceIdentity(d); err != nil {
		return err
	}
	if err := validateInvoiceAmounts(d); err != nil {
		return err
	}
	if err := validateInvoiceTimes(d); err != nil {
		return err
	}
	if err := requireText(ErrInvalidInvoiceDetail, "payment_terms", d.GetPaymentTerms()); err != nil {
		return err
	}
	if err := requireText(ErrInvalidInvoiceDetail, "reference_invoice_id", d.GetReferenceInvoiceId()); err != nil {
		return err
	}
	if err := validatePaymentCurrency(d); err != nil {
		return err
	}
	if err := validateInvoiceGrain(d.GetInvoiceDetailGrain()); err != nil {
		return err
	}
	return validateExtendedColumns(d.GetExtendedColumns())
}

func validateInvoiceIdentity(d *pbc.InvoiceDetail) error {
	if err := requireText(ErrInvalidInvoiceDetail, "invoice_detail_id", d.GetInvoiceDetailId()); err != nil {
		return err
	}
	if err := requireText(ErrInvalidInvoiceDetail, "invoice_id", d.GetInvoiceId()); err != nil {
		return err
	}
	if err := requireText(ErrInvalidInvoiceDetail, "invoice_issuer_name", d.GetInvoiceIssuerName()); err != nil {
		return err
	}
	return requireText(ErrInvalidInvoiceDetail, "billing_account_id", d.GetBillingAccountId())
}

func validateInvoiceAmounts(d *pbc.InvoiceDetail) error {
	if !isFinite(d.GetBilledCost()) {
		return invoiceError("billed_cost must be finite")
	}
	if err := requireText(ErrInvalidInvoiceDetail, "billing_currency", d.GetBillingCurrency()); err != nil {
		return err
	}
	if !currency.IsValid(d.GetBillingCurrency()) {
		return invoiceError("billing_currency must be a valid ISO 4217 currency code, got %q", d.GetBillingCurrency())
	}
	if err := validateInvoiceChargeCategory(d.GetChargeCategory()); err != nil {
		return err
	}
	return validateInvoiceIssueStatus(d.GetInvoiceIssueStatus())
}

func validateInvoiceTimes(d *pbc.InvoiceDetail) error {
	if err := requireTimestamp(ErrInvalidInvoiceDetail, "billing_period_start", d.GetBillingPeriodStart()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidInvoiceDetail, "billing_period_end", d.GetBillingPeriodEnd()); err != nil {
		return err
	}
	if err := requireAfter(ErrInvalidInvoiceDetail, "billing_period_end", "billing_period_start",
		d.GetBillingPeriodEnd(), d.GetBillingPeriodStart()); err != nil {
		return err
	}
	if err := optionalTimestamp(ErrInvalidInvoiceDetail, "invoice_issue_date", d.GetInvoiceIssueDate()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidInvoiceDetail, "invoice_detail_created",
		d.GetInvoiceDetailCreated()); err != nil {
		return err
	}
	if err := requireTimestamp(ErrInvalidInvoiceDetail, "invoice_detail_last_updated",
		d.GetInvoiceDetailLastUpdated()); err != nil {
		return err
	}
	if err := requireNotBefore(ErrInvalidInvoiceDetail, "invoice_detail_last_updated", "invoice_detail_created",
		d.GetInvoiceDetailLastUpdated(), d.GetInvoiceDetailCreated()); err != nil {
		return err
	}
	return optionalTimestamp(ErrInvalidInvoiceDetail, "payment_due_date", d.GetPaymentDueDate())
}

func validatePaymentCurrency(d *pbc.InvoiceDetail) error {
	code := d.GetPaymentCurrency()
	cost := d.PaymentCurrencyBilledCost
	codeSet := code != ""
	costSet := cost != nil
	if codeSet != costSet {
		return invoiceError("payment_currency and payment_currency_billed_cost must be set together")
	}
	if codeSet && !currency.IsValid(code) {
		return invoiceError("payment_currency must be a valid ISO 4217 currency code, got %q", code)
	}
	if costSet && !isFinite(*cost) {
		return invoiceError("payment_currency_billed_cost must be finite")
	}
	return validatePaymentLineage(d, cost)
}

func validatePaymentLineage(d *pbc.InvoiceDetail, cost *float64) error {
	lineage := d.GetPaymentCurrencyInvoiceDetailId()
	if lineage == "" || cost == nil || *cost == 0 || lineage == d.GetInvoiceDetailId() {
		return nil
	}
	return invoiceError(
		"payment_currency_invoice_detail_id must match invoice_detail_id when payment_currency_billed_cost is non-zero")
}

func validateInvoiceGrain(grain map[string]string) error {
	for key := range grain {
		if _, ok := focusInvoiceGrainKeys[key]; ok {
			continue
		}
		if strings.HasPrefix(key, "x_") && len(key) > len("x_") {
			continue
		}
		return invoiceError("invoice_detail_grain key %q must be a FOCUS property or start with x_", key)
	}
	return nil
}

func validateExtendedColumns(columns map[string]string) error {
	for key, value := range columns {
		if !strings.HasPrefix(key, "x_") || len(key) <= len("x_") {
			return invoiceError("extended_columns key %q must start with x_", key)
		}
		if !isDecimalString(value) {
			return invoiceError("extended_columns %q must be a decimal", key)
		}
	}
	return nil
}

func validateBillingPeriodStatus(status pbc.FocusBillingPeriodStatus) error {
	switch status {
	case pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN,
		pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED:
		return nil
	case pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_UNSPECIFIED:
		return billingPeriodError("billing_period_status must be OPEN or CLOSED, got %s", status)
	}
	return billingPeriodError("billing_period_status must be OPEN or CLOSED, got %s", status)
}

func validateInvoiceIssueStatus(status pbc.FocusInvoiceIssueStatus) error {
	switch status {
	case pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_OPEN,
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_ISSUED,
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_VOIDED:
		return nil
	case pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_UNSPECIFIED:
		return invoiceError("invoice_issue_status must be OPEN, ISSUED, or VOIDED, got %s", status)
	}
	return invoiceError("invoice_issue_status must be OPEN, ISSUED, or VOIDED, got %s", status)
}

func validateInvoiceChargeCategory(category pbc.FocusChargeCategory) error {
	switch category {
	case pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE,
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_PURCHASE,
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_TAX,
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_CREDIT,
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_ADJUSTMENT:
		return nil
	case pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_UNSPECIFIED,
		pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_REFUND:
		return rejectInvoiceChargeCategory(category)
	}
	return rejectInvoiceChargeCategory(category)
}

func rejectInvoiceChargeCategory(category pbc.FocusChargeCategory) error {
	return invoiceError("charge_category must be USAGE, PURCHASE, TAX, CREDIT, or ADJUSTMENT, got %s", category)
}

func requireText(sentinel error, name, value string) error {
	if value == "" {
		return datasetError(sentinel, "%s is required", name)
	}
	return nil
}

func requireTimestamp(sentinel error, name string, ts *timestamppb.Timestamp) error {
	if ts == nil {
		return datasetError(sentinel, "%s is required", name)
	}
	if !timestampOK(ts) {
		return datasetError(sentinel, "%s is not a valid timestamp", name)
	}
	return nil
}

func optionalTimestamp(sentinel error, name string, ts *timestamppb.Timestamp) error {
	if ts == nil || timestampOK(ts) {
		return nil
	}
	return datasetError(sentinel, "%s is not a valid timestamp", name)
}

func requireAfter(sentinel error, laterName, earlierName string, later, earlier *timestamppb.Timestamp) error {
	if !timestampAfter(later, earlier) {
		return datasetError(sentinel, "%s must be after %s", laterName, earlierName)
	}
	return nil
}

func requireNotBefore(sentinel error, laterName, earlierName string, later, earlier *timestamppb.Timestamp) error {
	if timestampBefore(later, earlier) {
		return datasetError(sentinel, "%s must be >= %s", laterName, earlierName)
	}
	return nil
}

func timestampOK(ts *timestamppb.Timestamp) bool {
	const (
		minSeconds = -62135596800
		maxSeconds = 253402300799
	)
	seconds := ts.GetSeconds()
	nanos := ts.GetNanos()
	return seconds >= minSeconds && seconds <= maxSeconds && nanos >= 0 && nanos < 1_000_000_000
}

func timestampBefore(a, b *timestamppb.Timestamp) bool {
	if a.GetSeconds() != b.GetSeconds() {
		return a.GetSeconds() < b.GetSeconds()
	}
	return a.GetNanos() < b.GetNanos()
}

func timestampAfter(a, b *timestamppb.Timestamp) bool {
	if a.GetSeconds() != b.GetSeconds() {
		return a.GetSeconds() > b.GetSeconds()
	}
	return a.GetNanos() > b.GetNanos()
}

func isDecimalString(value string) bool {
	if value == "" {
		return false
	}
	i := 0
	if value[0] == '-' {
		i++
	}
	if i >= len(value) {
		return false
	}
	sawDigit := false
	sawDot := false
	for ; i < len(value); i++ {
		switch c := value[i]; {
		case c >= '0' && c <= '9':
			sawDigit = true
		case c == '.' && !sawDot:
			sawDot = true
		default:
			return false
		}
	}
	return sawDigit
}
