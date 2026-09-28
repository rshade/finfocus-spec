# Tasks: Supplemental Dataset Service (Contract Commitments)

**Input**: Design documents from `specs/544-supplemental-contract-commitments/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required (Constitution V, test-first). Test tasks precede the code they cover.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: US1 host retrieval, US2 discovery, US3 developer helpers, US4 conformance, US5 TypeScript

## Phase 1: Setup

- [X] T001 Record the approved decision verbatim in
  `.specify/assessments/focus-1-4-support/decision.md` (markdownlint fixes only).

**Test-first gate (Constitution V)**: Before T010, write T002 to T009. Run
`go vet ./sdk/go/testing/ ./sdk/go/pluginsdk/` and, in `sdk/typescript/packages/client`,
`npx vitest run test/supplemental-dataset.test.ts`. All MUST fail: Go on the undefined
`pbc.GetContractCommitmentsRequest`, `pbc.PluginCapability_PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS`,
and the new SDK symbols; vitest on the missing `supplemental_pb.js` and client module (the client
`tsconfig.json` excludes `test/`, so `tsc --noEmit` cannot see it). Record the failure output in the
PR body.

## Phase 2: Failing tests (test-first gate)

- [X] T002 [P] [US3] Write table tests for `ValidateContractCommitment` (rules C0–C8, including the
  builder's exact messages and NaN/Inf rejection), `ValidateGetContractCommitmentsRequest` (Q1
  "set together or not at all", Q2 invalid timestamp, Q3 "strictly after start", Q4 "not
  negative"), `ContractCommitmentMatchesWindow` (W1: commitment period first, contract period
  fallback, open bounds, half-open edges, no bounds), `PaginateContractCommitments` ("0 → 50; above
  1000 → 1000", malformed token, token past the end, exact total), and
  `ValidateGetContractCommitmentsResponse` (P1–P6, duplicate IDs below and above 64 records); every
  failure asserts its sentinel, `codes.InvalidArgument`, and no `rpc error:` prefix, in
  `sdk/go/testing/supplemental_test.go`.
- [X] T003 [P] [US3] Write benchmarks with `b.ReportAllocs()` for every exported function in T002 and
  `testing.AllocsPerRun` assertions that the commitment, request, window, and response (≤ 64
  records) validators allocate 0 on valid input, in `sdk/go/testing/supplemental_benchmark_test.go`.
- [X] T004 [P] [US4] Write tests for `NewMockContractCommitmentSource` (rejects nil, invalid, and
  duplicate commitments; copies input) and its `GetContractCommitments` (window filter, pages,
  totals, invalid requests), plus `RunContractCommitmentConformance` against the mock and
  `RunContractCommitmentScenariosForTest` against broken sources (ignores window, invalid record,
  duplicates across pages, ignores page size, accepts one-bound window, accepts inverted window,
  accepts negative page size, accepts garbage token, wrong total, unstable order), each failing its
  target scenario, in `sdk/go/testing/contract_commitment_conformance_test.go`.
- [X] T005 [P] [US1] Write Serve tests: a provider plugin serves `GetContractCommitments` over gRPC
  and Connect with identical records and identical error code and message for an invalid request;
  the Connect health check reports `finfocus.v1.SupplementalDatasetService` as serving; a plugin
  without the provider returns `Unimplemented` over both transports and the health check reports
  the service unknown, in `sdk/go/pluginsdk/supplemental_test.go`.
- [X] T006 [P] [US2] Extend capability tests: inference adds
  `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS` only for providers, explicit capabilities override,
  `IsValidCapability` accepts 16 and rejects 17, and legacy metadata maps 16 to
  `supports_contract_commitments`, in `sdk/go/pluginsdk/capabilities_test.go`,
  `sdk/go/pluginsdk/plugin_info_test.go`, and `sdk/go/pluginsdk/capability_compat_test.go`.
- [X] T007 [P] [US3] Write wrapper tests (each pluginsdk wrapper matches the testing function),
  a builder test that `Build()` rejects NaN cost, benchmarks for the wrappers, a page-size constant
  drift test, and a compiled `Example_contractCommitmentProvider` with `// Output:`, in
  `sdk/go/pluginsdk/supplemental_test.go`, `sdk/go/pluginsdk/contract_commitment_builder_test.go`,
  and `sdk/go/pluginsdk/example_test.go`.
- [X] T008 [P] [US5] Write vitest tests for `SupplementalDatasetClient`: one page, a two-page
  `contractCommitments` iteration in order with the token passed back, default page size 50, the
  empty-page guard, and `ConnectError` code propagation, in
  `sdk/typescript/packages/client/test/supplemental-dataset.test.ts`.
- [X] T009 Run the gate commands above and confirm the failures.

## Phase 3: Foundational (protocol)

- [X] T010 Create `proto/finfocus/v1/supplemental.proto` from
  `specs/544-supplemental-contract-commitments/contracts/supplemental.proto` word for word.
- [X] T011 Add `PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16;` with the comment
  "Plugin implements SupplementalDatasetService.GetContractCommitments." to `PluginCapability` in
  `proto/finfocus/v1/enums.proto`.
- [X] T012 Run `make generate`; revert unrelated regenerated files with `git checkout -- <file>`;
  run `buf lint` and `buf breaking --against '.git#branch=main'` (in an agent worktree, against a
  `git archive main proto buf.yaml` copy).

## Phase 4: User Story 3 - Developer helpers (P1)

- [X] T013 [US3] Implement sentinels, `ValidateContractCommitment` (C0–C8, builder messages, no
  prefix), `ValidateGetContractCommitmentsRequest`, `ContractCommitmentMatchesWindow`,
  `PaginateContractCommitments`, and `ValidateGetContractCommitmentsResponse` (pairwise duplicate
  check up to 64 records, map above) with the 052 `invalidArgumentError`, in
  `sdk/go/testing/supplemental.go`.
- [X] T014 [US3] Make `ContractCommitmentBuilder.validate` call
  `plugintesting.ValidateContractCommitment` in `sdk/go/pluginsdk/contract_commitment_builder.go`.
- [X] T015 [US3] Add delegating wrappers with godoc in `sdk/go/pluginsdk/supplemental.go`.

**Checkpoint**: T002, T003, T007 (except serving) pass.

## Phase 5: User Story 1 - Host retrieves commitments (P1)

- [X] T016 [US1] Add `ContractCommitmentProvider` next to `AllocatorProvider`, an
  `optionalServices.commitments` field set in `newOptionalServices`, registration in `serveGRPC`
  and `serveConnect`, and the health entry `pbcconnect.SupplementalDatasetServiceName`, in
  `sdk/go/pluginsdk/sdk.go`.
- [X] T017 [US1] Add `contractCommitmentGRPCServer` and `contractCommitmentConnectHandler`
  (errors through `toConnectError`) in `sdk/go/pluginsdk/supplemental.go`.

**Checkpoint**: T005 passes.

## Phase 6: User Story 2 - Discovery (P1)

- [X] T018 [US2] Infer the capability in `inferCapabilities`, set `optionalCapabilities = 9` and
  `maxValidCapability = PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS // 16`, and update comments, in
  `sdk/go/pluginsdk/plugin_info.go`.
- [X] T019 [US2] Add `supports_contract_commitments` to `legacyCapabilityNames` and update the
  "1-14" comment in `sdk/go/pluginsdk/capability_compat.go`.

**Checkpoint**: T006 passes, including `TestLegacyCapabilityMapCompleteness`.

## Phase 7: User Story 4 - Conformance (P2)

- [X] T020 [US4] Implement `MockContractCommitmentSource` and `NewMockContractCommitmentSource` in
  `sdk/go/testing/supplemental_mock.go`.
- [X] T021 [US4] Implement `ContractCommitmentServer`, `ContractCommitmentHarness`, the seven
  scenarios, and `RunContractCommitmentConformance` in
  `sdk/go/testing/contract_commitment_conformance.go`; expose
  `RunContractCommitmentScenariosForTest` in `sdk/go/testing/export_test.go`.

**Checkpoint**: T004 passes.

## Phase 8: User Story 5 - TypeScript (P3)

- [X] T022 [US5] Implement `SupplementalDatasetClient` in
  `sdk/typescript/packages/client/src/clients/supplemental-dataset.ts` and export it and
  `supplemental_pb.js` from `sdk/typescript/packages/client/src/index.ts`.

**Checkpoint**: T008 passes; `npx tsc --noEmit` is clean.

## Phase 9: Polish

- [X] T023 [P] Write `docs/supplemental-datasets.md` (service, window rule, pagination, snapshot
  semantics, errors, capability, AWS/Azure/GCP commitment examples) and link it from
  `docs/README.md`.
- [X] T024 [P] Add a "Contract Commitments (Supplemental Datasets)" section and TOC entry to
  `sdk/go/pluginsdk/README.md`, a conformance section to `sdk/go/testing/README.md`, a client section
  to `sdk/typescript/README.md`, and short entries in `README.md` and `PLUGIN_DEVELOPER_GUIDE.md`.
- [X] T025 [P] Add a "Supplemental Dataset SDK Pattern (544)" note, Active Technologies, and Recent
  Changes entries to `CLAUDE.md` by hand.
- [X] T026 Run `goimports -w` on changed Go files, `golangci-lint run ./...`, `make test`, the TS
  checks, and `npx markdownlint-cli2` on changed Markdown; run every `quickstart.md` step.
- [X] T027 Write the gitignored `PR_MESSAGE.md` and validate it with `npx commitlint < PR_MESSAGE.md`.

## Dependencies & Execution Order

- **Test-first gate**: T002 to T008 are written after Setup and before T010, and must fail (T009).
- **Foundational (T010 to T012)**: Sequential; blocks all implementation.
- **US3 (T013 to T015)**: Depends on Foundational. US1, US2, and US4 use its helpers.
- **US1 (T016, T017)** and **US2 (T018, T019)**: Depend on Foundational; the `pluginsdk` test
  package compiles only once T015, T017, and T018 exist, so run their checkpoints together.
- **US4 (T020, T021)**: Depends on US3.
- **US5 (T022)**: Depends only on Foundational.
- **Polish**: Depends on all stories.

## Parallel Opportunities

- T002 to T008 touch different files and run in parallel.
- After T012: US3 (Go testing package) and US5 (TypeScript) run in parallel.
- T023 to T025 run in parallel.

## Implementation Strategy

MVP is US3 + US1 + US2 (a served, discoverable, validated RPC). US4 turns it into tested protocol,
and US5 completes SDK parity. All ship in one PR.
