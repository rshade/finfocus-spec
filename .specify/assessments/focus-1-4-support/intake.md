# Idea Intake: FOCUS 1.4 support

- **Slug**: focus-1-4-support
- **Created**: 2026-09-28
- **Source**: pasted text (maintainer request), plus facts the maintainer had already taken from
  the FOCUS v1.4 changelog (`https://focus.finops.org/docs/specification/v1-4/changelog/`)
- **Type**: new-capability

## Idea (as captured)

> Add FOCUS 1.4 support to finfocus-spec: Invoice Detail and Billing Period datasets, 17 new
> Contract Commitment columns, Invoice Detail ID and Commitment Program Eligibility Details on
> Cost and Usage.

Facts supplied with the request (from the v1.4 changelog, not yet verified at intake):

- New dataset **Invoice Detail** (about 22 columns, for example Invoice Detail ID, Invoice ID,
  Invoice Issue Date, Billed Cost, Billing Account ID, Billing Currency, Charge Category,
  Payment Currency, Payment Currency Billed Cost).
- **Billing Period** dataset columns listed as added: Billing Period Start, End, Status, Created,
  Last Updated, and Invoice Issuer Name.
- **Contract Commitment** gains 17 columns: 13 prefixed "Contract Commitment" (Applicability,
  Benefit Category, Created, Discount Percentage, Duration Type, Fulfillment Interval, Last
  Updated, Lifecycle Status, Model, Offer Category, Payment Interval, Payment Model, Payment
  Upfront Percentage), plus Invoice Issuer Name, Pricing Currency, Pricing Currency Contract
  Commitment Cost, and Service Provider Name.
- **Cost and Usage** gains 2 columns: Commitment Program Eligibility Details and Invoice Detail ID.
- Dataset attributes added: Correction Handling, Custom Column Handling, Dataset Completeness,
  Dataset Configuration, Delivery Handling, FOCUS Column Handling. Removed: Column Handling,
  Discount Handling, Invoice Handling.
- New features: Commitment Program Eligibility Details and Invoice Reconciliation.
- The changelog also mentions changed columns (about 14), which were not listed.

## Restated

Extend the finfocus-spec protobuf schema and SDKs, which implement FOCUS 1.3 today, so plugins can
emit FOCUS 1.4 data. That means two new datasets (Invoice Detail and Billing Period), new columns on
the existing Contract Commitment and Cost and Usage datasets, and any changes to existing columns.

## Origin & Context

- **Raised by**: repository maintainer (rshade)
- **Trigger**: the FinOps Foundation published FOCUS v1.4. The repo tracks FOCUS versions
  (FOCUS 1.3 landed in `specs/026-focus-1-3-migration/`).
- **Assumption (automated run)**: the slug `focus-1-4-support` was generated without asking,
  because this stage ran without a human in the loop.

## First-Glance Unknowns

- [NEEDS CLARIFICATION: exact data types, nullability, and requirement levels for each new column]
- [NEEDS CLARIFICATION: which roughly 14 existing columns changed in 1.4, and whether any change
  is breaking for existing proto fields (renames such as Invoice Issuer to Invoice Issuer Name)]
- [NEEDS CLARIFICATION: should Invoice Detail and Billing Period be new messages only, or also
  new RPCs or response fields for plugins to return them?]
- [NEEDS CLARIFICATION: do any FinFocus plugins or hosts need invoice-level data now, or is this
  conformance-driven?]
- [NEEDS CLARIFICATION: how dataset attributes (Correction Handling, Dataset Completeness, and
  others) should appear, if at all, in a gRPC protocol]
- [NEEDS CLARIFICATION: should this be one feature or split by dataset?]
