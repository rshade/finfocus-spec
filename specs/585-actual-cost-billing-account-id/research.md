# Research: Caller-Supplied Billing Account ID on Actual Cost Requests

## R1: Field shape and number

- **Decision**: `string billing_account_id = 9;` on `GetActualCostRequest`, with no `optional`
  keyword and no `reserved 10;` statement.
- **Rationale**: Field 9 is the next free tag (fields 1-8 are taken). Proto3 `string` has no
  presence, so empty and absent are the same value on the wire. That matches the contract: empty
  means "not supplied." An `optional` field would add a presence bit that the contract does not
  use, and it would generate a pointer field in Go. Field 10 is held by a comment, not by
  `reserved`. A `reserved` statement forbids the tag, so the later billing account name field
  would have to remove it first.
- **Alternatives considered**: `optional string` (presence nobody reads); a wrapper message such as
  `BillingContext` (larger change for one value; the issue asks for a plain string); the `tags`
  map (ruled out by the issue: tags are filters).

## R2: Wire compatibility

- **Decision**: An additive change. `buf breaking` against `main` must report nothing.
- **Rationale**: Old clients never send tag 9, so new plugins read `""`. Old plugins skip the
  unknown tag 9. Both directions keep working, which covers spec User Story 2 scenarios 2 and 3.
- **Alternatives considered**: None. This is the standard proto3 additive pattern the repository
  already uses (for example `page_size` = 7 and `page_token` = 8 in spec 044).

## R3: Reference producer

- **Decision**: `MockPlugin.GetActualCost` attaches a `FocusCostRecord` to each result when
  `billing_account_id` is non-empty, and attaches none when it is empty. The record is built in
  `sdk/go/testing` as a struct literal.
- **Rationale**: Today the mock never attaches a FOCUS record, so the acceptance test needs a
  producer. Keying the behavior on the request field leaves every existing test untouched, since no
  existing caller sets the field. `sdk/go/testing` cannot import `pluginsdk` (import cycle), so the
  record is a literal, not a `FocusRecordBuilder` chain. The record is validated by
  `pluginsdk.ValidateFocusRecord` in an external test package, which can import both.
- **Alternatives considered**: A new `MockPlugin` option such as `AttachFocusRecords bool` (adds
  configuration that the request field already expresses); always attaching a record (would need an
  invented account id when the field is empty, which the spec forbids).

## R4: Batch path

- **Decision**: `MockPlugin.batchActualCost` does not set the field. `BatchCostRequest` is unchanged.
- **Rationale**: `BatchCostRequest` has no billing account field, and the issue scopes the change to
  `GetActualCostRequest`. The batch path keeps producing results without FOCUS records, as today.
- **Alternatives considered**: Adding the field to `BatchCostRequest` or `ResourceDescriptor` (out
  of scope; a later issue can add it if a host needs it).

## R5: Conformance rule

- **Decision**: New exported validator `ValidateActualCostBillingAccount(req, resp) error` in
  `sdk/go/testing/contract.go`, and a new RPC correctness test
  `RPCCorrectness_GetActualCostBillingAccount` at the Standard level. The test sends a non-empty id
  and fails when any attached FOCUS record carries a different id. Results without a FOCUS record
  pass. NotFound and Unavailable are accepted, like `RPCCorrectness_GetActualCostRPC`.
- **Rationale**: FR-004 is the only rule that can be checked mechanically. FR-003 ("do not invent")
  cannot: a test cannot tell an invented id from a real one when the request was empty. The level
  matches the existing GetActualCost correctness test. The validator returns a `ContractError` that
  wraps a new sentinel `ErrBillingAccountIDMismatch`, matching the rest of `contract.go`. It names
  the field as `results[i].focus_record.billing_account_id`.
- **Alternatives considered**: Folding the check into `RPCCorrectness_GetActualCostRPC` (hides a
  distinct rule inside a generic test, and SC-004 wants the rule reported on its own); a
  `pluginsdk` delegate (no current caller; can be added later without breaking anything).

## R6: TypeScript SDK

- **Decision**: No hand-written TypeScript source changes. The regenerated `GetActualCostRequest`
  gains `billingAccountId: string`. Add a vitest case proving the actual cost iterator sends the id
  on every page.
- **Rationale**: `CostSourceClient.getActualCost` forwards the request whole, and the iterator
  `clone()`s it, so the field rides through. A test pins that behavior (FR-012).
- **Alternatives considered**: A builder or helper for the field (one string field does not need
  one).

## R7: Agent context files

- **Decision**: Skip `update-agent-context.sh` and add the CLAUDE.md "Active Technologies" and
  "Recent Changes" entries by hand.
- **Rationale**: The script drops the first line of wrapped "Recent Changes" entries (a known
  problem in this repository).
