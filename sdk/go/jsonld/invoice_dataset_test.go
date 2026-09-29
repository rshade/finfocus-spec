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

package jsonld_test

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/jsonld"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSerializeInvoiceDetail_OptionalSettlementCost(t *testing.T) {
	present := &pbc.InvoiceDetail{
		InvoiceDetailId:           "detail-1",
		InvoiceId:                 "invoice-1",
		BilledCost:                0,
		PaymentCurrency:           "EUR",
		PaymentCurrencyBilledCost: proto.Float64(0),
	}
	raw, err := jsonld.NewSerializer().SerializeInvoiceDetail(present)
	if err != nil {
		t.Fatalf("SerializeInvoiceDetail() error = %v", err)
	}
	doc := decodeJSONLD(t, raw)
	if doc["@type"] != jsonld.InvoiceDetailType {
		t.Fatalf("@type = %v", doc["@type"])
	}
	if doc["billedCost"] != float64(0) {
		t.Fatalf("billedCost = %v, want 0", doc["billedCost"])
	}
	if doc["paymentCurrencyBilledCost"] != float64(0) {
		t.Fatalf("paymentCurrencyBilledCost = %v, want 0", doc["paymentCurrencyBilledCost"])
	}

	absent := &pbc.InvoiceDetail{InvoiceDetailId: "detail-1", InvoiceId: "invoice-1", BilledCost: 12}
	raw, err = jsonld.NewSerializer().SerializeInvoiceDetail(absent)
	if err != nil {
		t.Fatalf("SerializeInvoiceDetail() error = %v", err)
	}
	doc = decodeJSONLD(t, raw)
	if _, ok := doc["paymentCurrencyBilledCost"]; ok {
		t.Fatalf("absent settlement cost was written: %v", doc["paymentCurrencyBilledCost"])
	}
}

func TestSerializeBillingPeriod_Type(t *testing.T) {
	raw, err := jsonld.NewSerializer().SerializeBillingPeriod(&pbc.BillingPeriod{InvoiceIssuerName: "Example Issuer"})
	if err != nil {
		t.Fatalf("SerializeBillingPeriod() error = %v", err)
	}
	doc := decodeJSONLD(t, raw)
	if doc["@type"] != jsonld.BillingPeriodType {
		t.Fatalf("@type = %v", doc["@type"])
	}
	if doc["invoiceIssuerName"] != "Example Issuer" {
		t.Fatalf("issuer = %v", doc["invoiceIssuerName"])
	}
}

func decodeJSONLD(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return doc
}
