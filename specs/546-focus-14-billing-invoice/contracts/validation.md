# Validation Contract

`ValidateBillingPeriod` and `ValidateInvoiceDetail` are the rule set. Builder `Build` methods
return their errors unchanged. Failures wrap `ErrInvalidBillingPeriod` or `ErrInvalidInvoiceDetail`,
carry `InvalidArgument`, and do not prefix the message with `rpc error`.

A valid record allocates nothing in either function. Error paths may allocate.

## Billing period failures

- nil record
- missing start, end, created, or last updated
- end not strictly after start
- status other than OPEN or CLOSED
- empty invoice issuer
- last updated before created

## Invoice detail failures

- nil record
- empty invoice detail id, invoice id, issuer, billing account id, payment terms, or reference
  invoice id
- missing billing period bounds, or an end that is not strictly after the start
- non-finite billed cost
- empty or non-ISO billing currency
- charge category other than USAGE, PURCHASE, TAX, CREDIT, or ADJUSTMENT
- issue status other than OPEN, ISSUED, or VOIDED
- missing created or last updated, or last updated before created
- an invalid optional timestamp when issue date or due date is set
- settlement currency and settlement cost not both set or both omitted
- non-ISO settlement currency, or a non-finite settlement cost
- payment-currency invoice detail id set, settlement cost non-zero, and the id different from
  invoice detail id
- a grain key that is not a FOCUS property and does not start with `x_`
- an extended column key without the `x_` prefix, or a value that is not a decimal

Purchase order number, description, empty grain, and empty extended columns are accepted.
