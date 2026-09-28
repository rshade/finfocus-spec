# Research: Supplemental Dataset Service (Contract Commitments)

All decisions below were made without a human, from the decision record
(`.specify/assessments/focus-1-4-support/decision.md`), the FOCUS research in the same folder, issue
544, and the 044, 051, and 052 precedents.

## R1. Service and file naming

- **Decision**: `proto/finfocus/v1/supplemental.proto`, `service SupplementalDatasetService`, RPC
  `GetContractCommitments`, messages `GetContractCommitmentsRequest` and
  `GetContractCommitmentsResponse`. Go provider `ContractCommitmentProvider`; capability
  `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16`; legacy key `supports_contract_commitments`.
- **Rationale**: FOCUS calls these "supplemental datasets", and issue 544 already proposes the
  service name. Stage B adds invoice RPCs to the same service, so the service is named for the
  dataset family, while the provider and capability are named per dataset so capabilities stay
  independent (decision handoff).
- **Alternatives**: `ContractCommitmentService` (would force a second service in stage B);
  one capability for the whole service (a commitment-only plugin would then advertise invoices).

## R2. Correction and Delivery Handling

- **Decision**: Not on the response. The operation has snapshot semantics: each call returns the
  source's current view for the window, which is FOCUS Replacement / Overwrite.
- **Rationale**: In FOCUS 1.4 these describe a dataset export, not a row, and the 1.3 Contract
  Commitment dataset has no such column. With no producer or consumer, adding an enum now would be
  guesswork; a field can be added later without a breaking change.
- **Alternatives**: A `correction_handling` enum on the response (unused until a Delta/Ledger
  producer exists).

## R3. Request filter and window semantics

- **Decision**: Optional `[start, end)` window only. Both unset returns all; exactly one set, an
  invalid timestamp, or `end <= start` is invalid-argument. A commitment matches when its period
  overlaps the window. The period is the commitment period when either commitment-period bound is
  set, otherwise the contract period; unset bounds are open-ended; a commitment with no bounds
  always matches.
- **Rationale**: Commitments span many billing periods, so overlap (not containment) is the useful
  query. Falling back to the contract period keeps records that only carry contract dates
  filterable. Mirrors `GetStats` window validation (051).
- **Alternatives**: Status or ID filters (no FOCUS 1.3 status column; no consumer); requiring a
  window (hosts wanting a full sync would have to invent one).

## R4. Allocation-free response validation

- **Decision**: Duplicate-ID detection compares pairs for pages of up to 64 records (zero
  allocations) and uses a map above that.
- **Rationale**: A map always allocates; pairwise comparison is O(n²) and becomes slow near the
  1000-record maximum. 64 covers the default page size of 50 with room to spare, so the common case
  is allocation-free. Benchmarks report both regimes.
- **Alternatives**: Always a map (allocates every call); always pairwise (about 500 000 string
  compares for a full page).

## R5. Pagination

- **Decision**: No "return all" mode. `page_size` 0 means `DefaultPageSize` (50), above
  `MaxPageSize` (1000) clamps to 1000, negative is invalid-argument. Tokens are base64 offsets,
  the same format as `pluginsdk.EncodePageToken`. `total_count` is exact.
- **Rationale**: The legacy "return all" branch in `GetActualCost` exists for pre-044 hosts, which
  cannot exist for a new RPC. Clamping (not rejecting) oversize pages matches 044. An exact total
  lets the conformance suite check a full walk.
- **Alternatives**: Reject `page_size > 1000` (stricter than every other paged RPC); allow
  `total_count = 0` when expensive (sources hold bounded lists; clarified in the spec).

## R6. Where the rules live

- **Decision**: `sdk/go/testing/supplemental.go` owns `ValidateContractCommitment`,
  `ValidateGetContractCommitmentsRequest`, `ValidateGetContractCommitmentsResponse`,
  `ContractCommitmentMatchesWindow`, and `PaginateContractCommitments`. `pluginsdk` wraps each one,
  and `ContractCommitmentBuilder.validate` calls `ValidateContractCommitment`.
- **Rationale**: `sdk/go/testing` cannot import `pluginsdk` (CLAUDE.md, 051/052), and the
  conformance suite and mock need the rules. One implementation means the builder, plugins, hosts,
  and conformance cannot disagree.
- **Error messages**: `ValidateContractCommitment` keeps the builder's existing messages
  (for example `contract_commitment_id is required`) with no sentinel prefix, so `Build()` output
  is unchanged. The request and response validators prefix their sentinel, as 052 does.
- **Alternatives**: Keep the rules private to the builder and duplicate them (drift risk).

## R7. Non-finite amounts

- **Decision**: The shared validator also rejects NaN and infinite `contract_commitment_cost` and
  `contract_commitment_quantity`. This applies to `Build()` too.
- **Rationale**: `cost < 0` is false for NaN, so the builder accepted NaN. Non-finite amounts are
  never valid FOCUS data and break JSON encoding. The change only rejects inputs no valid producer
  sends.

## R8. Error type

- **Decision**: Reuse the 052 `invalidArgumentError` (plain `Error()`, `Unwrap()` to a sentinel,
  `GRPCStatus()` returning InvalidArgument). New sentinels: `ErrInvalidContractCommitment`,
  `ErrInvalidContractCommitmentsRequest`, `ErrInvalidContractCommitmentsResponse`.
- **Rationale**: CLAUDE.md requires this pattern so both transports report the same code without
  the `rpc error:` prefix.

## R9. Reference producer placement and startup warning

- **Decision**: A separate `MockContractCommitmentSource` in `sdk/go/testing`, built with
  `NewMockContractCommitmentSource(commitments)`, which returns an error for an invalid, nil, or
  duplicate commitment and stores deep copies. No startup warning in `Serve` for commitment
  providers that rely on inferred capabilities.
- **Rationale**: Adding the method to `MockPlugin` would change what every existing `MockPlugin`
  user serves and infers. Commitment sources are billing plugins that are normally also cost
  sources, so the inferred pricing capabilities are usually correct; the docs tell
  commitment-only plugins to set capabilities explicitly.
- **Alternatives**: Extend `warnInferredOnlyCapabilities` (noisy for the expected producers).

## R10. Request validation in the adapter

- **Decision**: The gRPC and Connect adapters pass requests straight to the provider, as in 051
  and 052. Providers call `pluginsdk.ValidateGetContractCommitmentsRequest`.
- **Rationale**: The conformance harness serves an implementation directly, so validating only in
  `Serve` would let a plugin pass in production and fail conformance (or the reverse). One place
  keeps the behavior identical.

## R11. Stage B notes

- `GetBillingPeriods` and `GetInvoiceDetails` join the same service. The adapter will then implement
  all three methods and return `Unimplemented` for a provider the plugin lacks; `optionalServices`
  gains an `invoices` field, and the service is registered when either provider exists.
- The 17 Contract Commitment columns from issue 542 travel over this RPC unchanged; issue 542
  should extend `ValidateContractCommitment` for any new rules (enums, percentage ranges).
