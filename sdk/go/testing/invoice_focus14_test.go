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
	"testing"
	"time"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestInvoiceDatasetValidatorsAllocationFree(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	period, err := pluginsdk.NewBillingPeriodBuilder().
		WithWindow(at, at.AddDate(0, 1, 0)).
		WithStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED).
		WithInvoiceIssuerName("Example Issuer").
		WithCreated(at).
		WithLastUpdated(at).
		Build()
	if err != nil {
		t.Fatalf("period: %v", err)
	}
	detail, err := pluginsdk.NewInvoiceDetailBuilder().
		WithBaseline(at).
		WithPaymentCurrency("EUR").
		WithPaymentCurrencyBilledCost(0).
		WithGrain(map[string]string{"ServiceName": "Compute", "x_BillingMode": "Pay-per-Use"}).
		WithExtendedColumns(map[string]string{"x_Tax": "1.25"}).
		Build()
	if err != nil {
		t.Fatalf("detail: %v", err)
	}

	allocs := testing.AllocsPerRun(100, func() {
		if validateErr := plugintesting.ValidateBillingPeriod(period); validateErr != nil {
			t.Fatalf("period: %v", validateErr)
		}
		if validateErr := plugintesting.ValidateInvoiceDetail(detail); validateErr != nil {
			t.Fatalf("detail: %v", validateErr)
		}
	})
	if allocs != 0 {
		t.Fatalf("validators allocated %v times, want 0", allocs)
	}
}

func TestValidateInvoiceDetail_Nil(t *testing.T) {
	err := plugintesting.ValidateInvoiceDetail(nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}
