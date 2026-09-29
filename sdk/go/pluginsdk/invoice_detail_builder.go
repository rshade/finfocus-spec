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

package pluginsdk

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

const baselineInvoiceBilledCost = 10

// InvoiceDetailBuilder constructs a FOCUS 1.4 InvoiceDetail.
// Build fails when a mandatory column is missing or a conditional pair is half set.
// A zero settlement cost is present. Leaving it unset means the column is absent.
type InvoiceDetailBuilder struct {
	record *pbc.InvoiceDetail
}

// NewInvoiceDetailBuilder returns an empty invoice detail builder.
func NewInvoiceDetailBuilder() *InvoiceDetailBuilder {
	return &InvoiceDetailBuilder{record: &pbc.InvoiceDetail{}}
}

// WithIdentity sets the line id, invoice id, issuer, and billing account.
func (b *InvoiceDetailBuilder) WithIdentity(detailID, invoiceID, issuer, accountID string) *InvoiceDetailBuilder {
	b.record.InvoiceDetailId = detailID
	b.record.InvoiceId = invoiceID
	b.record.InvoiceIssuerName = issuer
	b.record.BillingAccountId = accountID
	return b
}

// WithBillingPeriod sets the inclusive start and exclusive end.
func (b *InvoiceDetailBuilder) WithBillingPeriod(start, end time.Time) *InvoiceDetailBuilder {
	b.record.BillingPeriodStart = timestamppb.New(start)
	b.record.BillingPeriodEnd = timestamppb.New(end)
	return b
}

// WithBilledCost sets the billed cost. Zero and negative values are stored.
func (b *InvoiceDetailBuilder) WithBilledCost(cost float64) *InvoiceDetailBuilder {
	b.record.BilledCost = cost
	return b
}

// WithBillingCurrency sets the ISO 4217 billing currency.
func (b *InvoiceDetailBuilder) WithBillingCurrency(code string) *InvoiceDetailBuilder {
	b.record.BillingCurrency = code
	return b
}

// WithChargeCategory sets the charge category. REFUND fails in Build.
func (b *InvoiceDetailBuilder) WithChargeCategory(category pbc.FocusChargeCategory) *InvoiceDetailBuilder {
	b.record.ChargeCategory = category
	return b
}

// WithIssueStatus sets the invoice issue status.
func (b *InvoiceDetailBuilder) WithIssueStatus(status pbc.FocusInvoiceIssueStatus) *InvoiceDetailBuilder {
	b.record.InvoiceIssueStatus = status
	return b
}

// WithIssueDate sets the official issue date. Omit the call to leave it null.
func (b *InvoiceDetailBuilder) WithIssueDate(issued time.Time) *InvoiceDetailBuilder {
	b.record.InvoiceIssueDate = timestamppb.New(issued)
	return b
}

// WithCreated sets when the line was instantiated.
func (b *InvoiceDetailBuilder) WithCreated(created time.Time) *InvoiceDetailBuilder {
	b.record.InvoiceDetailCreated = timestamppb.New(created)
	return b
}

// WithLastUpdated sets when the line was last updated.
func (b *InvoiceDetailBuilder) WithLastUpdated(updated time.Time) *InvoiceDetailBuilder {
	b.record.InvoiceDetailLastUpdated = timestamppb.New(updated)
	return b
}

// WithDescription sets the line description. Empty means null.
func (b *InvoiceDetailBuilder) WithDescription(description string) *InvoiceDetailBuilder {
	b.record.InvoiceDetailDescription = description
	return b
}

// WithGrain copies the granularity map. An empty map is stored as null.
func (b *InvoiceDetailBuilder) WithGrain(grain map[string]string) *InvoiceDetailBuilder {
	b.record.InvoiceDetailGrain = copyStringMap(grain)
	return b
}

// WithPaymentCurrency sets the settlement currency. Set the billed cost as well.
func (b *InvoiceDetailBuilder) WithPaymentCurrency(code string) *InvoiceDetailBuilder {
	b.record.PaymentCurrency = code
	return b
}

// WithPaymentCurrencyBilledCost sets a present settlement cost, including zero.
func (b *InvoiceDetailBuilder) WithPaymentCurrencyBilledCost(cost float64) *InvoiceDetailBuilder {
	b.record.PaymentCurrencyBilledCost = proto.Float64(cost)
	return b
}

// WithPaymentCurrencyInvoiceDetailID sets the lineage id for a different currency grain.
func (b *InvoiceDetailBuilder) WithPaymentCurrencyInvoiceDetailID(id string) *InvoiceDetailBuilder {
	b.record.PaymentCurrencyInvoiceDetailId = id
	return b
}

// WithPaymentDueDate sets the payment deadline. Omit the call to leave it null.
func (b *InvoiceDetailBuilder) WithPaymentDueDate(due time.Time) *InvoiceDetailBuilder {
	b.record.PaymentDueDate = timestamppb.New(due)
	return b
}

// WithPaymentTerms sets the payment terms, for example "Net 30".
func (b *InvoiceDetailBuilder) WithPaymentTerms(terms string) *InvoiceDetailBuilder {
	b.record.PaymentTerms = terms
	return b
}

// WithPurchaseOrderNumber sets the customer purchase order. Empty means null.
func (b *InvoiceDetailBuilder) WithPurchaseOrderNumber(number string) *InvoiceDetailBuilder {
	b.record.PurchaseOrderNumber = number
	return b
}

// WithReferenceInvoiceID sets the invoice this line refers to.
func (b *InvoiceDetailBuilder) WithReferenceInvoiceID(id string) *InvoiceDetailBuilder {
	b.record.ReferenceInvoiceId = id
	return b
}

// WithExtendedColumns copies custom monetary columns. An empty map is stored as null.
func (b *InvoiceDetailBuilder) WithExtendedColumns(columns map[string]string) *InvoiceDetailBuilder {
	b.record.ExtendedColumns = copyStringMap(columns)
	return b
}

// WithBaseline fills the columns that do not allow nulls: a USD usage line for
// one month, status Issued, terms "Net 30", and a reference id equal to the invoice id.
// Settlement currency stays absent.
func (b *InvoiceDetailBuilder) WithBaseline(at time.Time) *InvoiceDetailBuilder {
	return b.
		WithIdentity("detail-1", "invoice-1", "Example Issuer", "account-1").
		WithBillingPeriod(at, at.AddDate(0, 1, 0)).
		WithBilledCost(baselineInvoiceBilledCost).
		WithBillingCurrency("USD").
		WithChargeCategory(pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE).
		WithIssueStatus(pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_ISSUED).
		WithCreated(at).
		WithLastUpdated(at).
		WithPaymentTerms("Net 30").
		WithReferenceInvoiceID("invoice-1")
}

// Build validates and returns the invoice line.
// The error is the one ValidateInvoiceDetail returns.
func (b *InvoiceDetailBuilder) Build() (*pbc.InvoiceDetail, error) {
	if err := plugintesting.ValidateInvoiceDetail(b.record); err != nil {
		return nil, err
	}
	return b.record, nil
}
