# Quickstart: Billing Period and Invoice Detail

## Billing period

```go
period, err := pluginsdk.NewBillingPeriodBuilder().
    WithWindow(start, start.AddDate(0, 1, 0)).
    WithStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_OPEN).
    WithInvoiceIssuerName("Example Issuer").
    WithCreated(start).
    WithLastUpdated(start).
    Build()
```

`Build` returns an error when the status is unset, the end is not after the start, or the last
update is before creation.

## Invoice line

```go
line, err := pluginsdk.NewInvoiceDetailBuilder().
    WithBaseline(start).
    Build()
```

`WithBaseline` fills the columns that do not allow nulls: a USD usage line, status Issued,
payment terms "Net 30", and a reference invoice id equal to the line's own invoice id.
Settlement currency is left unset. Set both `WithPaymentCurrency` and
`WithPaymentCurrencyBilledCost` when the issuer settles in another currency. A cost of 0 is
stored. Leaving the cost unset means the column is absent.

`WithChargeCategory(REFUND)` fails in `Build`.

## TypeScript

```typescript
const line = new InvoiceDetailBuilder().withBaseline(start).build();
builder.withBilledCost(99); // does not change line
```

`build()` returns a clone. The Go validator remains the rule set.
