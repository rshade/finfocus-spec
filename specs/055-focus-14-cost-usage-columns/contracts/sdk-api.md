# Contract: SDK API additions

## Go: `sdk/go/pluginsdk`

### Builder setters (`focus_builder.go`)

```go
// WithInvoiceDetailID sets the FOCUS 1.4 InvoiceDetailId: the invoice line item this
// cost row contributes to. It is unique within an InvoiceId, so also set invoice_id
// (WithInvoice or WithFinancials); Build fails otherwise. The setter does not validate
// or allocate.
func (b *FocusRecordBuilder) WithInvoiceDetailID(invoiceDetailID string) *FocusRecordBuilder

// WithCommitmentProgramEligibilityDetails sets the FOCUS 1.4
// CommitmentProgramEligibilityDetails column, a JSON object such as
// {"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}. The setter stores the string
// as given and does not validate or allocate; Build fails with
// ErrInvalidCommitmentProgramEligibilityDetails unless it is a well-formed JSON object.
func (b *FocusRecordBuilder) WithCommitmentProgramEligibilityDetails(detailsJSON string) *FocusRecordBuilder
```

`WithInvoice`'s godoc notes that `invoiceIssuer` carries the FOCUS 1.4 `InvoiceIssuerName` column.

### Sentinel errors (`focus_conformance.go`)

```go
// ErrInvalidCommitmentProgramEligibilityDetails indicates that
// commitment_program_eligibility_details is set but is not a well-formed JSON object.
ErrInvalidCommitmentProgramEligibilityDetails = errors.New(
    "commitment_program_eligibility_details must be a well-formed JSON object",
)

// ErrInvoiceIDMissingForInvoiceDetail indicates invoice_detail_id is set without invoice_id.
ErrInvoiceIDMissingForInvoiceDetail = errors.New("invoice_id required when invoice_detail_id is set")
```

Both are wrapped with `NewValidationErrorWithCause`, so `errors.Is` and `errors.As(*ValidationError)`
both work.

### Changed behavior

- `ValidateFocusRecord` / `ValidateFocusRecordWithOptions` / `Build()`: a record with only
  `service_provider_name` passes the provider rule. With both provider names empty, the error is
  `ValidationError{FieldName: "provider_name", Constraint: "required", Expected:
  "non-empty service_provider_name (or deprecated provider_name)"}`.
- The same functions apply rules V2 and V3 from data-model.md.
- `FocusFieldNames()` returns 68 names, including the two new ones.

### Example (`example_test.go`)

`ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails` builds a FOCUS 1.4 record that
sets only the service provider, an invoice detail ID and eligibility details, prints the two new
values and `Build`'s error, then shows that a truncated JSON value fails with
`errors.Is(err, pluginsdk.ErrInvalidCommitmentProgramEligibilityDetails) == true`.

## Go: `sdk/go/jsonld`

```go
InvoiceDetailID                     = "focus:invoiceDetailId"
CommitmentProgramEligibilityDetails = "focus:commitmentProgramEligibilityDetails"
```

The serializer emits `invoiceDetailId` and `commitmentProgramEligibilityDetails` when non-empty.

## TypeScript: `@rshade/finfocus-client`

Generated `FocusCostRecord` gains `invoiceDetailId: string` and
`commitmentProgramEligibilityDetails: string`.

`FocusRecordBuilder` gains:

```ts
withInvoiceDetailId(id: string): this
// throws ValidationError("Invoice detail ID cannot be empty", "invoiceDetailId") for empty/blank

withCommitmentProgramEligibilityDetails(detailsJson: string): this
// throws ValidationError(..., "commitmentProgramEligibilityDetails") unless JSON.parse
// returns a non-null, non-array object
```
