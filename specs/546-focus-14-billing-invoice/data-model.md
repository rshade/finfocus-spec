# Data Model: FOCUS 1.4 Billing Period and Invoice Detail

## BillingPeriod

| # | Field | Proto type | Null | Rule |
| --- | --- | --- | --- | --- |
| 1 | billing_period_start | Timestamp | No | Inclusive |
| 2 | billing_period_end | Timestamp | No | Exclusive, strictly after start |
| 3 | billing_period_status | FocusBillingPeriodStatus | No | OPEN or CLOSED |
| 4 | invoice_issuer_name | string | No | Non-empty |
| 5 | billing_period_created | Timestamp | No | Required |
| 6 | billing_period_last_updated | Timestamp | No | >= created |

`FocusBillingPeriodStatus`: UNSPECIFIED, OPEN, CLOSED. UNSPECIFIED is not valid on a record.

## InvoiceDetail

| # | Field | Proto type | Presence | Rule |
| --- | --- | --- | --- | --- |
| 1 | invoice_detail_id | string | Required | Non-empty |
| 2 | invoice_id | string | Required | Non-empty |
| 3 | invoice_issuer_name | string | Required | Non-empty |
| 4 | billing_account_id | string | Required | Non-empty |
| 5 | billing_period_start | Timestamp | Required | Inclusive |
| 6 | billing_period_end | Timestamp | Required | Exclusive, strictly after start |
| 7 | billed_cost | double | Required | Finite, any sign, zero allowed |
| 8 | billing_currency | string | Required | ISO 4217 |
| 9 | charge_category | FocusChargeCategory | Required | USAGE, PURCHASE, TAX, CREDIT, ADJUSTMENT |
| 10 | invoice_issue_status | FocusInvoiceIssueStatus | Required | OPEN, ISSUED, VOIDED |
| 11 | invoice_issue_date | Timestamp | Nullable | Unset means null |
| 12 | invoice_detail_created | Timestamp | Required | |
| 13 | invoice_detail_last_updated | Timestamp | Required | >= created |
| 14 | invoice_detail_description | string | Nullable | Empty means null |
| 15 | invoice_detail_grain | map<string, string> | Nullable | FOCUS key or `x_` prefix |
| 16 | payment_currency | string | Conditional | With field 17, or both empty |
| 17 | payment_currency_billed_cost | optional double | Conditional | Nil is absent. Zero is present |
| 18 | payment_currency_invoice_detail_id | string | Conditional | If set and field 17 is non-zero, equals field 1 |
| 19 | payment_due_date | Timestamp | Nullable | Unset means null |
| 20 | payment_terms | string | Required | Non-empty |
| 21 | purchase_order_number | string | Conditional | Empty allowed |
| 22 | reference_invoice_id | string | Required | Non-empty |
| 23 | extended_columns | map<string, string> | Custom | `x_` key, decimal value. Empty allowed |

`FocusInvoiceIssueStatus`: UNSPECIFIED, OPEN, ISSUED, VOIDED. UNSPECIFIED is not valid on a record.

`FocusChargeCategory` is the existing enum. REFUND and UNSPECIFIED fail invoice validation.
The enum itself is unchanged.

## Relationships

A billing period joins to invoice lines and cost rows on (billing period start, invoice issuer
name). An invoice line joins to cost rows on (invoice issuer name, invoice id) and optionally
invoice detail id. Those joins are not checked on one record.
