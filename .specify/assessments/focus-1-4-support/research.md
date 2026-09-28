# Idea Research: FOCUS 1.4 support

- **Slug**: focus-1-4-support
- **Created**: 2026-09-28
- **Evidence confidence (overall)**: high for what FOCUS 1.4 defines and how it maps onto this
  repo; low for demand (no plugin or host has asked for invoice-level data yet)

The primary source is the FOCUS specification repository at tag `v1.4`
(`FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec`), read file by file through `gh api`. Column
types, nullability, feature levels, and allowed values below are copied from each column's
"Content Constraints" table and "Requirements" section, not from a web page summary. Repo paths
are relative to the finfocus-spec root.

## Users & Demand

- The maintainer's stated goal is to track FOCUS releases. FOCUS 1.3 was adopted through issue
  #183 and `specs/026-focus-1-3-migration/`. — [source: `gh issue view 183`, `specs/026-*`]
  (confidence: high, cited)
- No open issue in `rshade/finfocus-spec` mentions FOCUS 1.4, invoices, or billing periods.
  A search of `rshade/finfocus` for "invoice" found nothing either. — [source:
  `gh issue list --search FOCUS` and `--search invoice`, 2026-09-28] (confidence: high, cited)
- No plugin in the ecosystem is known to return invoice or billing period data. The existing
  `ContractCommitment` message (FOCUS 1.3) is not returned by any RPC, so even 1.3's
  supplemental dataset has no observed consumer. — [source: `grep ContractCommitment
  proto/finfocus/v1/costsource.proto` finds only comments] (confidence: high, cited)
- The 1.3 migration already promised 1.4 work: `provider_name` and `publisher` are documented
  as "Will be removed in FOCUS 1.4". — [source: `proto/finfocus/v1/focus.proto` lines 38-39 and
  152-153, `specs/026-focus-1-3-migration/data-model.md` lines 77-78] (confidence: high, cited)
- Demand is **stated** (the spec-tracking goal), not **observed** (no user request).
  — ASSUMPTION based on the absence of issues (confidence: medium)

## Prior Art

### FOCUS 1.4 release facts

- FOCUS v1.4 was tagged on 2026-06-04. The changelog says "Announced June 2026".
  — [source: `gh api repos/.../FOCUS_Spec/releases` (tag `v1.4`, published
  2026-06-04T23:52:12Z); `CHANGELOG.md` at `v1.4`] (confidence: high, cited)
- Totals from the changelog: 2 new datasets, **47 new columns** (6 Billing Period, 22 Invoice
  Detail, 17 Contract Commitment, 2 Cost and Usage), 6 new attributes, 2 new supported features,
  and 17 new glossary entries. The spec classifies **Incompatible Changes: None**.
  — [source: FOCUS `CHANGELOG.md` v1.4] (confidence: high, cited)
- The website changelog header reads "Columns: +47 -2 ~14". The 2 removed columns are
  `ProviderName` and `PublisherName`. — [source: focus.finops.org v1.4 changelog page,
  WebFetch summary] (confidence: medium, cited; model summary of the page)

### The 14 changed columns

The website names these 14 changed columns: ContractCommitmentPeriodStart,
ContractCommitmentPeriodEnd, ContractPeriodStart, ContractPeriodEnd, AllocatedMethodDetails,
BilledCost, ChargeClass, ContractApplied, EffectiveCost, HostProviderName, InvoiceId,
PricingCurrency, PricingCurrencyEffectiveCost, and ServiceProviderName. — [source: WebFetch
summary] (confidence: medium). A diff of v1.3 and v1.4 column files (normalized for links and
formatting) confirms which changes matter for this repo:

| Column | What changed in 1.4 | Impact here |
| :--- | :--- | :--- |
| BilledCost | Covered/covering charges; MUST be 0 for non-invoicing entities; sum per InvoiceId within the Rounding Variance Tolerance | Validation doc only; no proto change |
| EffectiveCost | Explicit equality to BilledCost per ChargeCategory; amortization of covering charges | Possible new conformance rules |
| ContractApplied | Moves to a JSON Object Schema (`ContractAppliedObject`), a migration-compatible change | **Existing `WithContractApplied(commitmentID)` stores a plain ID, which was already non-conformant in 1.3** |
| AllocatedMethodDetails | Now `AllocatedMethodDetailsObject` (JSON Object Schema); Column type "Dimension / Metric" | Stored as a string; docs and validation only |
| InvoiceId | Feature level Recommended → **Conditional** (MUST when the issuer supports payable invoices) | Docs and dry-run hints |
| PricingCurrency | Allows nulls True → **False** | proto3 string cannot express null anyway |
| PricingCurrencyEffectiveCost | Allows nulls True → **False**; MUST be the PricingCurrency equivalent of EffectiveCost | Possible conformance rule |
| ChargeClass | "Correction" now tied to a closed billing period, not a previously invoiced one | Docs |
| ServiceProviderName, HostProviderName | Presence moved to dataset level; HostProviderName MUST match ServiceProviderName otherwise | Docs |
| ContractCommitmentPeriod\*, ContractPeriod\* | Presence relocated to dataset-level rules | None |

The diff also shows content changes to columns the website does not list as changed: Tags and
AllocatedTags (tag-scheme prefix rules), SkuPriceDetails (JSON Object type), CommitmentDiscountId,
CommitmentDiscountStatus, ResourceId, and SkuPriceId (new rules for Used and Unused commitments).
In the Contract Commitment dataset, ContractCommitmentId and ContractId changed from Allows nulls
True to False. — [source: local diff of `specification/datasets/*/columns/*.md`, v1.3 against
v1.4] (confidence: high, cited)

### New datasets: column definitions

**Billing Period** (new supporting dataset, 6 columns). The dataset MUST be present when the
invoice issuer supports payable invoices. It joins to Cost and Usage and to Invoice Detail on
(BillingPeriodStart, InvoiceIssuerName).

| Column | Type | Feature level | Nulls | Key rules / allowed values |
| :--- | :--- | :--- | :--- | :--- |
| BillingPeriodCreated | Date/Time | Mandatory | No | When the record was instantiated |
| BillingPeriodEnd | Date/Time | Mandatory | No | Exclusive end bound |
| BillingPeriodLastUpdated | Date/Time | Mandatory | No | Must be >= BillingPeriodCreated |
| BillingPeriodStart | Date/Time | Mandatory | No | Inclusive start bound |
| BillingPeriodStatus | String (enum) | Mandatory | No | `Open`, `Closed`; MUST NOT go from Closed back to Open unless the customer requests or approves it |
| InvoiceIssuerName | String | Mandatory | No | The entity that issues invoices |

**Invoice Detail** (new transactional dataset, 22 columns). The dataset MUST be present when
the invoice issuer supports payable invoices. It MUST represent every invoice line item with a
non-zero BilledCost. It joins to Cost and Usage on (InvoiceIssuerName, InvoiceId) and optionally
on InvoiceDetailId.

| Column | Type | Feature level | Nulls | Key rules / allowed values |
| :--- | :--- | :--- | :--- | :--- |
| BilledCost | Decimal | Mandatory | No | In BillingCurrency; sum per (InvoiceDetailId, InvoiceId, InvoiceIssuerName) = payable amount when Issued; must match the Cost and Usage sum within `MAX(100 × Subunit, SQRT(Rows) × 0.5 × Subunit)` unless Tax or Open |
| BillingAccountId | String | Mandatory | No | Unique within the invoice issuer |
| BillingCurrency | String | Mandatory | No | ISO 4217 national currency, as on the invoice |
| BillingPeriodEnd | Date/Time | Mandatory | No | Must match Cost and Usage for the same InvoiceId |
| BillingPeriodStart | Date/Time | Mandatory | No | Must match Cost and Usage for the same InvoiceId |
| ChargeCategory | String (enum) | Mandatory | No | `Usage`, `Purchase`, `Tax`, `Credit`, `Adjustment`; MAY be `Usage` when a line aggregates non-Tax categories |
| InvoiceDetailCreated | Date/Time | Mandatory | No | <= BillingPeriod.LastUpdated when the period is Closed |
| InvoiceDetailDescription | String | Mandatory | Yes | SHOULD NOT be null |
| InvoiceDetailGrain | JSON (Key-Value) | Mandatory | Yes | Properties that define line granularity; FOCUS keys ContractId, RegionId, ResourceId, ResourceType, ServiceName, SkuId, SkuMeter, SkuPriceId, SubAccountId; custom keys use the `x_` prefix |
| InvoiceDetailId | String | Mandatory | No | Unique within an InvoiceId |
| InvoiceDetailLastUpdated | Date/Time | Mandatory | No | Must be >= InvoiceDetailCreated |
| InvoiceId | String | Mandatory | No | MAY exist before the invoice is issued |
| InvoiceIssueDate | Date/Time | Mandatory | Yes | Official issue date |
| InvoiceIssueStatus | String (enum) | Mandatory | No | `Open`, `Issued`, `Voided`; MUST NOT go from Issued back to Open unless the customer approves |
| InvoiceIssuerName | String | Mandatory | No | The entity that issues invoices |
| PaymentCurrency | String | Conditional | No | Present when billing and payment currencies can differ |
| PaymentCurrencyBilledCost | Decimal | Conditional | No | Same condition; MAY be non-zero while BilledCost is 0 on aggregate rows |
| PaymentCurrencyInvoiceDetailId | String | Conditional | No | Present when the two currencies are aggregated at different levels |
| PaymentDueDate | Date/Time | Mandatory | Yes | Payment deadline |
| PaymentTerms | String | Mandatory | No | For example "Net 30" |
| PurchaseOrderNumber | String | Conditional | Yes | Present when customers can enter PO numbers |
| ReferenceInvoiceId | String | Mandatory | No | The original InvoiceId when adjusting another invoice, otherwise the row's own InvoiceId |

The InvoiceDetail dataset MUST also include custom columns for any monetary metric on an invoice
that has no FOCUS column.

### Contract Commitment: 17 new columns

Presence rules are now written at dataset level. Every column below is required
("ContractCommitment MUST include") except where a condition is shown.

| Column | Type | Feature level | Nulls | Key rules / allowed values |
| :--- | :--- | :--- | :--- | :--- |
| ContractCommitmentApplicability | JSON Object | Mandatory | No | `ContractCommitmentApplicabilityObject`: IsGlobalScope, IsComplexScope, Applicability{Cost,Usage} 0-1, InclusionOperator/ExclusionOperator (`And`/`Or`), Inclusions/Exclusions rules with operators In, NotIn, StartsWith, NotStartsWith, Contains, NotContains, EndsWith, Exists, DoesNotExist, and the `*` wildcard |
| ContractCommitmentBenefitCategory | String (enum) | Mandatory | No | `Discount`, `Entitlement`, `Availability`, `Other` |
| ContractCommitmentCreated | Date/Time | Mandatory | No | When the record was instantiated |
| ContractCommitmentDiscountPercentage | Decimal | Mandatory | Yes | 0.0-1.0; NOT null when BenefitCategory is Discount; MUST be null when Availability; one tier per row |
| ContractCommitmentDurationType | String (format) | Mandatory | No | "`[positive int] [unit]`", units Minute(s), Hour(s), Day(s), Week(s), Month(s), Quarter(s), Year(s) |
| ContractCommitmentFulfillmentInterval | String (enum) | Mandatory | No | `Hourly`, `Daily`, `Weekly`, `Monthly`, `Quarterly`, `Semi-Annual`, `Annual`, `Full Period`, `Transactional`, `Custom`; not `Full Period` when Model is Continuous |
| ContractCommitmentLastUpdated | Date/Time | Mandatory | No | Must be >= Created |
| ContractCommitmentLifecycleStatus | String (enum) | Mandatory | No | `Proposed`, `Pending`, `Active`, `Exhausted`, `Expired`, `Canceled`, `Superseded`; a record replaced by a new ID MUST become Superseded |
| ContractCommitmentModel | String (enum) | Mandatory | No | `Continuous`, `Discontinuous`; Discontinuous when the interval is Full Period |
| ContractCommitmentOfferCategory | String (enum) | Mandatory | No | `Public`, `Negotiated` |
| ContractCommitmentPaymentInterval | String (enum) | Mandatory | No | `One-Time`, `Monthly`, `Quarterly`, `Semi-Annual`, `Annual`, `Custom`; One-Time when the model is All Upfront |
| ContractCommitmentPaymentModel | String (enum) | Mandatory | No | `No Upfront`, `Partial Upfront`, `All Upfront` |
| ContractCommitmentPaymentUpfrontPercentage | Decimal | Conditional (the provider offers Partial Upfront) | No | 1.0 when All Upfront, 0.0 when No Upfront, strictly between 0 and 1 when Partial |
| InvoiceIssuerName | String | Mandatory | No | The entity that issues invoices |
| PricingCurrency | String | Conditional (pricing and billing currencies can differ) | No | ISO 4217 |
| PricingCurrencyContractCommitmentCost | Decimal | Conditional (same condition) | Yes | NOT null when Category is Spend and PricingCurrency is set; MAY be null for Usage |
| ServiceProviderName | String | Mandatory | No | Service provider |

### Cost and Usage: 2 new columns

| Column | Type | Feature level | Nulls | Key rules |
| :--- | :--- | :--- | :--- | :--- |
| CommitmentProgramEligibilityDetails | JSON Object | Conditional (the provider has one or more commitment programs) | Yes | `{"CommitmentPrograms":[{"ProgramType":"…"}]}`; NOT null when the charge is eligible, even if no commitment applied; one ProgramType MUST match CommitmentDiscountType when that is set; no term or payment data; custom keys use `x_` |
| InvoiceDetailId | String | Conditional (the issuer supports payable invoices) | Yes | Null when there is no invoice or provisional invoice; unique within an InvoiceId |

### Attributes

Six attributes were added: CorrectionHandling, CustomColumnHandling, DatasetCompleteness,
DatasetConfiguration, DeliveryHandling, and FocusColumnHandling. Three were removed:
ColumnHandling, DiscountHandling, and InvoiceHandling. These rules govern how datasets are
delivered, such as corrections, completeness, and custom columns, rather than per-row fields.
CurrencyFormat now requires ISO 4217 for **every** currency value, and the allowance for virtual
currency values is gone. — [source: FOCUS `CHANGELOG.md` v1.4, "Changed attributes"]
(confidence: high, cited)

### Internal prior art: FOCUS 1.3 migration (`specs/026-focus-1-3-migration/`)

- **Field numbering**: 1.2 used 1-58 and 1.3 used 59-66. The proto comment says "FOCUS 1.4
  could use 67+". — [source: `focus.proto` lines 297-306, `026/research.md` §4] (high, cited)
- **Supplemental dataset as a standalone message**: `ContractCommitment` (fields 1-12) with one
  enum (`FocusContractCommitmentCategory`), a fluent builder with `Build()` validation, and no
  RPC. — [source: `focus.proto` lines 366-453, `contract_commitment_builder.go`] (high, cited)
- **Deprecation**: the `[deprecated = true]` option plus a zerolog warning once per process
  (`sync.Once`) when both the old and new fields are set. — [source: `focus_builder.go` lines
  14-50 and 545-570] (high, cited)
- **JSON-bearing columns stored as strings**: `allocated_method_details` and `sku_price_details`
  are `string`. Tags and allocated tags are `map<string,string>`. — [source: `focus.proto`]
  (high, cited)
- **Validation**: `validateFocus13Rules` holds cross-field rules, currently only
  AllocatedMethodId requires AllocatedResourceId. — [source: `focus_conformance.go` lines
  340-360] (high, cited)

## Market & Context

- Cloud providers publish FOCUS exports, and FOCUS 1.4 is the current release. Hosts that parse
  FOCUS 1.4 exports expect InvoiceDetailId and CommitmentProgramEligibilityDetails on cost rows.
  — ASSUMPTION (confidence: medium). Which providers have shipped 1.4 exports is not verified.
- Doing nothing costs little now. Plugins can put 1.4 columns in
  `FocusCostRecord.extended_columns` (the "backpack" map, field 23) without a schema change.
  — [source: `focus.proto` lines 292-294] (confidence: high, cited)
- The repo's docs and code call this a "FOCUS 1.3" SDK. Falling one release behind is a
  positioning cost more than a functional one. — ASSUMPTION (confidence: medium)

## Data & Constraints

### Repo mapping

| 1.4 item | Current repo state | Needed |
| :--- | :--- | :--- |
| Cost and Usage `InvoiceDetailId` | Missing | `string invoice_detail_id = 67;` in `FocusCostRecord` |
| Cost and Usage `CommitmentProgramEligibilityDetails` | Missing | `string commitment_program_eligibility_details = 68;` (JSON string, matching `allocated_method_details`) |
| Cost and Usage `InvoiceIssuerName` | `string invoice_issuer = 40;` | No wire change. Column name vs field name is cosmetic. Update comments, docs, and possibly add a `WithInvoiceIssuerName` alias. Renaming the proto field would break the generated Go and TS APIs |
| Removed `ProviderName` / `PublisherName` | `provider_name = 1`, `publisher = 55`, both `[deprecated = true]` | Keep them on the wire; removal would break MINOR semver. **`validateMandatoryFields` still requires `provider_name`** (`focus_conformance.go` lines 170-181), which contradicts 1.4. That validation needs to accept `service_provider_name` or be version-aware |
| Contract Commitment +17 | `ContractCommitment` fields 1-12 | Fields 13-29, plus up to 7 new enums |
| Contract Commitment `ContractCommitmentDescription` (1.3 column) | **Missing from proto (pre-existing gap)** | Candidate for field 30 |
| Billing Period dataset | No message | New `BillingPeriod` message plus a `FocusBillingPeriodStatus` enum |
| Invoice Detail dataset | No message | New `InvoiceDetail` message plus a `FocusInvoiceIssueStatus` enum |
| Invoice Detail `ChargeCategory` | `FocusChargeCategory` has a `REFUND = 5` value, which is not a FOCUS 1.2+ value | Reuse the enum; the validator must reject REFUND and UNSPECIFIED on invoice rows |
| `FocusFieldNames()` in `dry_run.go` | 66 names, "FOCUS 1.2/1.3" | Add the 2 new cost-row names. Whether dry-run should cover the other datasets is open |
| `docs/focus-columns.md` | "FOCUS 1.2/1.3 Column Reference" | Add a 1.4 section |
| `sdk/go/jsonld/vocabulary.go` | Has 1.3 terms and `invoiceIssuer` | Add 1.4 terms |
| TypeScript SDK | Generated `focus_pb.ts`; the hand-written `FocusRecordBuilder` has only 4 methods | Regenerate. Builder parity is optional |
| `sdk/go/testing` (`focus13_conformance_test.go`, `mock_plugin.go`) | 1.3 conformance tests | Add a `focus14_conformance_test.go` |

### Proposed field-number plan (all additive)

`FocusCostRecord`: 67 `invoice_detail_id` (string), 68 `commitment_program_eligibility_details`
(string, JSON). Reserve 69-80 for later 1.4.x or 1.5 cost-row columns in a comment.

`ContractCommitment`, with fields 13-29 in changelog order:

| # | Field | Proto type |
| :--- | :--- | :--- |
| 13 | contract_commitment_applicability | string (JSON) |
| 14 | contract_commitment_benefit_category | enum FocusContractCommitmentBenefitCategory |
| 15 | contract_commitment_created | google.protobuf.Timestamp |
| 16 | contract_commitment_discount_percentage | optional double (null vs 0 matters) |
| 17 | contract_commitment_duration_type | string ("3 Years") |
| 18 | contract_commitment_fulfillment_interval | enum FocusContractCommitmentFulfillmentInterval |
| 19 | contract_commitment_last_updated | google.protobuf.Timestamp |
| 20 | contract_commitment_lifecycle_status | enum FocusContractCommitmentLifecycleStatus |
| 21 | contract_commitment_model | enum FocusContractCommitmentModel |
| 22 | contract_commitment_offer_category | enum FocusContractCommitmentOfferCategory |
| 23 | contract_commitment_payment_interval | enum FocusContractCommitmentPaymentInterval |
| 24 | contract_commitment_payment_model | enum FocusContractCommitmentPaymentModel |
| 25 | contract_commitment_payment_upfront_percentage | optional double |
| 26 | invoice_issuer_name | string |
| 27 | pricing_currency | string |
| 28 | pricing_currency_contract_commitment_cost | optional double |
| 29 | service_provider_name | string |
| 30 | contract_commitment_description | string (1.3 gap) |

The `optional` keyword is new to `focus.proto`, which uses plain `double` everywhere. It is
proposed only where the spec requires a null that differs from 0.0, for example
DiscountPercentage "MUST be null when Availability".

`BillingPeriod` (new): 1 billing_period_start, 2 billing_period_end, 3 billing_period_status
(enum: UNSPECIFIED, OPEN, CLOSED), 4 invoice_issuer_name, 5 billing_period_created,
6 billing_period_last_updated.

`InvoiceDetail` (new): 1 invoice_detail_id, 2 invoice_id, 3 invoice_issuer_name,
4 billing_account_id, 5 billing_period_start, 6 billing_period_end, 7 billed_cost (double),
8 billing_currency, 9 charge_category (FocusChargeCategory), 10 invoice_issue_status (enum:
UNSPECIFIED, OPEN, ISSUED, VOIDED), 11 invoice_issue_date (Timestamp, nullable by being unset),
12 invoice_detail_created, 13 invoice_detail_last_updated, 14 invoice_detail_description,
15 invoice_detail_grain (map<string,string>, following the Tags decision in 026 §5),
16 payment_currency, 17 payment_currency_billed_cost (optional double),
18 payment_currency_invoice_detail_id, 19 payment_due_date, 20 payment_terms,
21 purchase_order_number, 22 reference_invoice_id, 23 extended_columns (map<string,string>, for
the custom monetary columns the dataset requires).

**Breaking?** No. Every change adds fields, messages, or enums, which is a MINOR bump under
`buf breaking` rules. The only behavior change is relaxing the `provider_name` requirement in
`validateMandatoryFields`. That validation gets looser, so it is compatible. Deleting fields 1
and 55 should be deferred to a v2 package and marked `reserved`.

### Constraints

- proto3 scalars cannot express null. The spec draws null-vs-zero distinctions that need
  `optional` or wrapper types. — [source: `focus.proto` conventions; FOCUS column rules above]
  (high, cited)
- The builder pattern requires `Build()` to fail fast and allocate nothing on simple setters.
  1.3 setters bench below 1 ns/op with 0 allocs/op. — [source: CLAUDE.md FOCUS 1.3 performance
  notes] (high, cited)
- `make generate` also regenerates TS bindings, and it can reformat unrelated `*.connect.go`
  files. — [source: CLAUDE.md, 051 and 052 notes] (high, cited)
- Many 1.4 rules span rows or datasets: invoice sums, the rounding tolerance, BillingPeriod
  joins, Superseded lifecycle rules, and one-way status transitions. A per-record validator
  cannot check them. — [source: column requirements above] (high, cited)

## Evidence Against the Idea

- **No consumer.** No RPC returns the 1.3 `ContractCommitment` message, and none would return
  `BillingPeriod` or `InvoiceDetail` either. Without a delivery RPC, the 30 new message fields
  are schema with no path to the wire. — [source: `costsource.proto` grep] (high, cited)
- **FinFocus is a cost-estimation and actual-cost tool, not an AP or invoice system.** Invoice
  reconciliation (payment terms, PO numbers, due dates) sits far from the plugin use cases
  (projected cost, recommendations, allocation). — ASSUMPTION (medium)
- **The escape hatch exists.** `extended_columns` already carries arbitrary 1.4 cost-row
  columns. — [source: `focus.proto` field 23] (high, cited)
- **1.3 debt is still open.** `ContractApplied` is stored as a plain commitment ID, not the
  JSON object that 1.3 and 1.4 require. `ContractCommitmentDescription` is missing. Conformance
  still requires the deprecated `provider_name`. Supporting 1.4 on top of these gaps inherits
  them. — [source: `focus_builder.go` lines 482-492, `focus.proto`, `focus_conformance.go`]
  (high, cited)
- **Size.** 47 columns, 9 new enums, 2 messages, builders, conformance, and docs make this
  roughly 3-4 times the 1.3 migration, which added 8 cost columns and 12 commitment fields.
  — [source: counts above vs `026/data-model.md`] (medium, cited)

## Gaps & Open Questions

- [NEEDS CLARIFICATION: Should `BillingPeriod` and `InvoiceDetail` be message-only, like
  `ContractCommitment`, or get a delivery RPC or response field? The same question applies,
  unanswered, to `ContractCommitment` in 1.3.]
- [NEEDS CLARIFICATION: Should conformance become version-aware (1.2, 1.3, or 1.4 profile) or
  jump to 1.4? This decides whether `provider_name` stays mandatory for anyone.]
- [NEEDS CLARIFICATION: Should `ContractApplied` get a typed helper that emits the 1.4
  `ContractAppliedObject` JSON? The `ContractAppliedObject` schema was not extracted in this
  pass; see `specification/datasets/cost_and_usage/columns/contractapplied.md` at `v1.4`.]
- [NEEDS CLARIFICATION: Should JSON columns (Applicability, CommitmentProgramEligibilityDetails,
  InvoiceDetailGrain) stay as strings or `map<string,string>`, or become typed proto messages?
  Typed messages give validation but diverge from how 1.3 handled JSON columns.]
- [NEEDS CLARIFICATION: Is the 1.3 `ContractCommitment` builder's hard requirement on
  `billing_currency` correct? FOCUS lists BillingCurrency there as Allows nulls = True in both
  1.3 and 1.4.]
- [NEEDS CLARIFICATION: Which cloud providers ship FOCUS 1.4 exports today (AWS, Azure, GCP,
  OCI)? Not researched.]
- Gap: the "14 changed columns" list comes from a model summary of the website. The local diff
  agrees on the substantive ones but also finds changes the site does not list; see "The 14
  changed columns" above.
- Gap: the new dataset attributes (CorrectionHandling, DatasetCompleteness, and others) were
  not mapped to protocol concepts. They look like data-generator documentation and delivery
  rules, not fields.

## Sources

- `https://github.com/FinOps-Open-Cost-and-Usage-Spec/FOCUS_Spec` at tags `v1.4` and `v1.3`:
  `CHANGELOG.md`, `specification/datasets/**/dataset.md`, and
  `specification/datasets/**/columns/*.md`, fetched through `gh api` (host: api.github.com;
  policy: allowlisted, GitHub)
- `https://focus.finops.org/docs/specification/v1-4/changelog/` (host: focus.finops.org;
  policy: confirmed-by-user, since the requester named this host explicitly; one page, no
  crawling)
- Local repo: `proto/finfocus/v1/focus.proto`, `proto/finfocus/v1/enums.proto`,
  `proto/finfocus/v1/costsource.proto`, `sdk/go/pluginsdk/focus_builder.go`,
  `sdk/go/pluginsdk/contract_commitment_builder.go`, `sdk/go/pluginsdk/focus_conformance.go`,
  `sdk/go/pluginsdk/dry_run.go`, `sdk/go/jsonld/vocabulary.go`, `docs/focus-columns.md`,
  `sdk/typescript/packages/client/src/`, and `specs/026-focus-1-3-migration/`
- GitHub issues: `rshade/finfocus-spec` #183 (FOCUS 1.3 migration); issue searches for "FOCUS"
  and "invoice" (host: github.com; policy: allowlisted)
