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

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// BillingPeriodBuilder constructs a FOCUS 1.4 BillingPeriod.
// Build fails when a mandatory column is missing or inconsistent.
type BillingPeriodBuilder struct {
	record *pbc.BillingPeriod
}

// NewBillingPeriodBuilder returns an empty billing period builder.
func NewBillingPeriodBuilder() *BillingPeriodBuilder {
	return &BillingPeriodBuilder{record: &pbc.BillingPeriod{}}
}

// WithWindow sets the inclusive start and exclusive end.
func (b *BillingPeriodBuilder) WithWindow(start, end time.Time) *BillingPeriodBuilder {
	b.record.BillingPeriodStart = timestamppb.New(start)
	b.record.BillingPeriodEnd = timestamppb.New(end)
	return b
}

// WithStatus sets the billing period status.
func (b *BillingPeriodBuilder) WithStatus(status pbc.FocusBillingPeriodStatus) *BillingPeriodBuilder {
	b.record.BillingPeriodStatus = status
	return b
}

// WithInvoiceIssuerName sets the invoice issuer.
func (b *BillingPeriodBuilder) WithInvoiceIssuerName(name string) *BillingPeriodBuilder {
	b.record.InvoiceIssuerName = name
	return b
}

// WithCreated sets when the record was instantiated.
func (b *BillingPeriodBuilder) WithCreated(created time.Time) *BillingPeriodBuilder {
	b.record.BillingPeriodCreated = timestamppb.New(created)
	return b
}

// WithLastUpdated sets when the record was last updated.
func (b *BillingPeriodBuilder) WithLastUpdated(updated time.Time) *BillingPeriodBuilder {
	b.record.BillingPeriodLastUpdated = timestamppb.New(updated)
	return b
}

// Build validates and returns the billing period.
// The error is the one ValidateBillingPeriod returns.
func (b *BillingPeriodBuilder) Build() (*pbc.BillingPeriod, error) {
	if err := plugintesting.ValidateBillingPeriod(b.record); err != nil {
		return nil, err
	}
	return b.record, nil
}
