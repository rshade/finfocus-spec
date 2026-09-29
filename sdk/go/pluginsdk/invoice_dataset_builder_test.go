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

package pluginsdk_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func periodAt(at time.Time) *pluginsdk.BillingPeriodBuilder {
	return pluginsdk.NewBillingPeriodBuilder().
		WithWindow(at, at.AddDate(0, 1, 0)).
		WithStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN).
		WithInvoiceIssuerName("Example Issuer").
		WithCreated(at).
		WithLastUpdated(at)
}

func TestBillingPeriodBuilder_Build(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	period, err := periodAt(at).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if period.GetBillingPeriodStatus() != pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN {
		t.Fatalf("status = %s", period.GetBillingPeriodStatus())
	}
}

func TestBillingPeriodBuilder_Rejects(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mutate  func(*pluginsdk.BillingPeriodBuilder)
		wantMsg string
	}{
		{name: "unset status", mutate: func(b *pluginsdk.BillingPeriodBuilder) {
			b.WithStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_UNSPECIFIED)
		}, wantMsg: "billing_period_status"},
		{name: "empty issuer", mutate: func(b *pluginsdk.BillingPeriodBuilder) {
			b.WithInvoiceIssuerName("")
		}, wantMsg: "invoice_issuer_name is required"},
		{name: "end not after start", mutate: func(b *pluginsdk.BillingPeriodBuilder) {
			b.WithWindow(at, at)
		}, wantMsg: "billing_period_end must be after"},
		{name: "updated before created", mutate: func(b *pluginsdk.BillingPeriodBuilder) {
			b.WithLastUpdated(at.Add(-time.Hour))
		}, wantMsg: "billing_period_last_updated must be >="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builder := periodAt(at)
			tt.mutate(builder)
			_, err := builder.Build()
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("Build() error = %v, want %q", err, tt.wantMsg)
			}
			if !errors.Is(err, plugintesting.ErrInvalidBillingPeriod) {
				t.Fatalf("Build() error = %v, want ErrInvalidBillingPeriod", err)
			}
		})
	}
}

func TestInvoiceDetailBuilder_BaselineAndZeroSettlement(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	line, err := pluginsdk.NewInvoiceDetailBuilder().
		WithBaseline(at).
		WithPaymentCurrency("EUR").
		WithPaymentCurrencyBilledCost(0).
		WithGrain(map[string]string{"ServiceName": "Compute", "x_BillingMode": "Pay-per-Use"}).
		WithExtendedColumns(map[string]string{"x_Tax": "1.25"}).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if line.PaymentCurrencyBilledCost == nil || line.GetPaymentCurrencyBilledCost() != 0 {
		t.Fatalf("settlement cost = %v, want present 0", line.GetPaymentCurrencyBilledCost())
	}
	if line.GetInvoiceDetailDescription() != "" {
		t.Fatalf("description = %q, want empty", line.GetInvoiceDetailDescription())
	}
}

func TestInvoiceDetailBuilder_Rejects(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mutate  func(*pluginsdk.InvoiceDetailBuilder)
		wantMsg string
	}{
		{name: "refund", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithChargeCategory(pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_REFUND)
		}, wantMsg: "charge_category"},
		{name: "unspecified category", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithChargeCategory(pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_UNSPECIFIED)
		}, wantMsg: "charge_category"},
		{name: "currency without cost", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithPaymentCurrency("EUR")
		}, wantMsg: "must be set together"},
		{name: "cost without currency", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithPaymentCurrencyBilledCost(1)
		}, wantMsg: "must be set together"},
		{name: "bad grain key", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithGrain(map[string]string{"service_name": "Compute"})
		}, wantMsg: "invoice_detail_grain"},
		{name: "lineage mismatch", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithPaymentCurrency("EUR").
				WithPaymentCurrencyBilledCost(3).
				WithPaymentCurrencyInvoiceDetailID("other-line")
		}, wantMsg: "payment_currency_invoice_detail_id"},
		{name: "bad extended key", mutate: func(b *pluginsdk.InvoiceDetailBuilder) {
			b.WithExtendedColumns(map[string]string{"Tax": "1"})
		}, wantMsg: "extended_columns"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			builder := pluginsdk.NewInvoiceDetailBuilder().WithBaseline(at)
			tt.mutate(builder)
			_, err := builder.Build()
			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("Build() error = %v, want %q", err, tt.wantMsg)
			}
			if !errors.Is(err, plugintesting.ErrInvalidInvoiceDetail) {
				t.Fatalf("Build() error = %v, want ErrInvalidInvoiceDetail", err)
			}
		})
	}
}

func TestInvoiceDetailBuilder_NegativeAndZeroBilledCost(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, cost := range []float64{0, -5.5} {
		line, err := pluginsdk.NewInvoiceDetailBuilder().WithBaseline(at).WithBilledCost(cost).Build()
		if err != nil {
			t.Fatalf("cost %v: %v", cost, err)
		}
		if line.GetBilledCost() != cost {
			t.Fatalf("billed cost = %v, want %v", line.GetBilledCost(), cost)
		}
	}
}

func TestIsValidInvoiceEnums(t *testing.T) {
	t.Parallel()
	if !pluginsdk.IsValidBillingPeriodStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED) {
		t.Fatal("CLOSED should be valid")
	}
	if pluginsdk.IsValidBillingPeriodStatus(pbc.FocusBillingPeriodStatus(99)) {
		t.Fatal("unknown billing period status should be invalid")
	}
	if !pluginsdk.IsValidInvoiceIssueStatus(pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_VOIDED) {
		t.Fatal("VOIDED should be valid")
	}
	if pluginsdk.IsValidInvoiceIssueStatus(pbc.FocusInvoiceIssueStatus(99)) {
		t.Fatal("unknown issue status should be invalid")
	}
}

func BenchmarkBillingPeriodBuilder_WithInvoiceIssuerName(b *testing.B) {
	builder := pluginsdk.NewBillingPeriodBuilder()
	b.ReportAllocs()
	for range b.N {
		builder.WithInvoiceIssuerName("Example Issuer")
	}
}

func BenchmarkInvoiceDetailBuilder_WithBilledCost(b *testing.B) {
	builder := pluginsdk.NewInvoiceDetailBuilder()
	b.ReportAllocs()
	for range b.N {
		builder.WithBilledCost(10)
	}
}

func BenchmarkInvoiceDetailBuilder_WithChargeCategory(b *testing.B) {
	builder := pluginsdk.NewInvoiceDetailBuilder()
	b.ReportAllocs()
	for range b.N {
		builder.WithChargeCategory(pbc.FocusChargeCategory_FOCUS_CHARGE_CATEGORY_USAGE)
	}
}

func BenchmarkIsValidInvoiceEnums(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_ = pluginsdk.IsValidBillingPeriodStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN)
		_ = pluginsdk.IsValidInvoiceIssueStatus(pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_ISSUED)
	}
}
