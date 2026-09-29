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

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// Package-level slices make enum membership a zero-allocation scan.
// UNSPECIFIED is a known value. Builders that require a set column reject it separately.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var (
	allBillingPeriodStatuses = []pbc.FocusBillingPeriodStatus{
		pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_UNSPECIFIED,
		pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN,
		pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED,
	}
	allInvoiceIssueStatuses = []pbc.FocusInvoiceIssueStatus{
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_UNSPECIFIED,
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_OPEN,
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_ISSUED,
		pbc.FocusInvoiceIssueStatus_FOCUS_INVOICE_ISSUE_STATUS_VOIDED,
	}
)

// IsValidBillingPeriodStatus reports whether status is a defined billing period status,
// including UNSPECIFIED.
func IsValidBillingPeriodStatus(status pbc.FocusBillingPeriodStatus) bool {
	for _, candidate := range allBillingPeriodStatuses {
		if candidate == status {
			return true
		}
	}
	return false
}

// IsValidInvoiceIssueStatus reports whether status is a defined invoice issue status,
// including UNSPECIFIED.
func IsValidInvoiceIssueStatus(status pbc.FocusInvoiceIssueStatus) bool {
	for _, candidate := range allInvoiceIssueStatuses {
		if candidate == status {
			return true
		}
	}
	return false
}
