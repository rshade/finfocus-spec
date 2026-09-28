# Contract: `proto/finfocus/v1/focus.proto` changes

All changes are additive or comment-only. `buf breaking --against '.git#branch=main'` must pass.

## New fields in `FocusCostRecord`

Insert after `string contract_applied = 66;`:

```proto
  // ==========================================================================
  // FOCUS 1.4 Additions (Field Numbers 67-68, 69-80 reserved)
  // ==========================================================================
  //
  // Field Numbering Strategy (continued):
  // - Fields 67-68: FOCUS 1.4 Cost and Usage columns
  // - Fields 69-80: Reserved for future Cost and Usage columns (FOCUS 1.4.x or 1.5).
  //   Do not use them for anything else.
  //
  // FOCUS 1.4 removes the ProviderName and PublisherName columns. Fields 1 and 55
  // stay on the wire, deprecated, until a v2 package can reserve them.

  // InvoiceDetailId: Identifier of the invoice line item this row contributes to.
  // Unique within an InvoiceId, so invoice_id MUST be set when this is set.
  // Empty means null: there is no invoice or only a provisional invoice.
  // FOCUS 1.4 Section: Invoice Detail ID (CONDITIONAL: required when the invoice
  // issuer supports payable invoices).
  string invoice_detail_id = 67;

  // CommitmentProgramEligibilityDetails: JSON object listing the commitment
  // programs this charge is eligible for, whether or not one was applied, e.g.
  // {"CommitmentPrograms":[{"ProgramType":"Savings Plan"}]}.
  // Stored as a JSON string, like allocated_method_details. When set, it MUST be
  // a well-formed JSON object. One ProgramType SHOULD match commitment_discount_type
  // when that is set; custom keys use the "x_" prefix. Empty means null.
  // FOCUS 1.4 Section: Commitment Program Eligibility Details (CONDITIONAL:
  // required when the provider has one or more commitment programs).
  string commitment_program_eligibility_details = 68;
```

## Comment-only changes

| Field | New comment text (replaces the existing FOCUS status lines) |
| :--- | :--- |
| message `FocusCostRecord` | "normalized to the FinOps FOCUS specification (1.2, 1.3 and 1.4)" and a line listing the 1.4 additions |
| `provider_name = 1` | "DEPRECATED in FOCUS 1.3: Use service_provider_name instead. Removed from FOCUS 1.4; kept on the wire for backward compatibility. Validation accepts service_provider_name in its place." |
| `publisher = 55` | "DEPRECATED in FOCUS 1.3: Use host_provider_name instead. Removed from FOCUS 1.4; kept on the wire for backward compatibility." |
| `invoice_id = 19` | Add "FOCUS 1.4: CONDITIONAL (was RECOMMENDED): required when the invoice issuer supports payable invoices." |
| `invoice_issuer = 40` | "InvoiceIssuerName (FOCUS 1.4 column name; formerly InvoiceIssuer): The entity that issues the invoice. The proto field keeps its original name." |
| `pricing_currency = 51` | Add "FOCUS 1.4: not nullable when the column applies." |
| `pricing_currency_effective_cost = 53` | Add "FOCUS 1.4: not nullable when the column applies; MUST be the PricingCurrency equivalent of effective_cost." |
| `allocated_method_details = 62` | Add "FOCUS 1.4 defines this column as a JSON object (AllocatedMethodDetailsObject); the SDK does not enforce the format." |
| 1.3 numbering comment | "Future FOCUS versions should continue this pattern (FOCUS 1.4 uses 67-68)." |
