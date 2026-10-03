# Tasks: Structured Attributes on ResourceDescriptor

**Input**: Design documents from `specs/596-resource-descriptor-attributes/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md)

**Tests**: Required. Constitution Principle V (test-first) applies. Within each story, write the tests first
and run them to confirm they fail.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Confirm a clean baseline in the worktree: `make generate && git diff --exit-code -- sdk/`, then `go test
      ./sdk/go/pluginsdk/ ./sdk/go/testing/`, and record the `BenchmarkValidateResourceDescriptor_ZeroAllocs` allocs/op
      (`go test ./sdk/go/pluginsdk/ -run '^$' -bench ValidateResourceDescriptor -benchmem`)

---

## Phase 2: Foundational (proto first; blocks every story)

- [X] T002 Add `google.protobuf.Struct attributes = 12;` to `ResourceDescriptor` in `proto/finfocus/v1/costsource.proto`
      with the comment from `contracts/proto.md`: optional; unset or empty means the host sent none, so fall back to
      `tags`; hosts SHOULD keep sending `tags`; prefer `attributes` when both disagree; REQUIRED host redaction (`__`
      keys; credential-like keys whose names contain, case-insensitively, `password`, `secret`, `token`, `credential`,
      `privatekey`, `accesskey`, or `connectionstring`; IaC secret values such as the Pulumi signature
      `4dabf18193072939515e22adb298388d`); plugins MUST NOT log the field verbatim; "encoded size MUST NOT exceed 65536
      bytes (pluginsdk.MaxAttributesBytes)"; the batch must fit the transport limit (1 MB Connect, 4 MB gRPC default),
      so hosts split batches; integers above 2^53 go as strings; point to `pluginsdk.AttributeValue`
- [X] T003 Update the message-level comment above `ResourceDescriptor` and the `tags` field comment in
      `proto/finfocus/v1/costsource.proto` so they mention `attributes` as the structured alternative and the tag value
      limit of 2048
- [X] T004 Run `make generate`. Confirm that only `sdk/go/proto/finfocus/v1/costsource.pb.go` and the TypeScript
      `costsource_pb.ts` change (restore any unrelated reformatted `*.connect.go` with `git checkout`). Run `make
      buf-lint` and `buf breaking --against 'https://github.com/rshade/finfocus-spec.git#branch=main'` (archive
      `origin/main` if the git URL cannot be read)

**Checkpoint**: The field exists in the Go and TypeScript bindings.

---

## Phase 3: User Story 1 - A plugin reads nested inputs on any cost RPC (P1) 🎯 MVP

**Goal**: A ten-segment nested input reaches the plugin intact.

**Independent Test**: A harness test sends `GetProjectedCost` with nested attributes, and the plugin
observes the exact value.

- [X] T005 [US1] Write `TestResourceDescriptorAttributesDelivered` in `sdk/go/testing/attributes_delivery_test.go`
      (Apache header, package `testing_test`). Use a capturing plugin that embeds `*plugintesting.MockPlugin` and
      records the request descriptor. Make it a table over every RPC that carries a `ResourceDescriptor` (`Supports`,
      `GetProjectedCost`, `GetPricingSpec`, `BatchCost` `resources`, and `GetRecommendations` `target_resources`) so
      that SC-001 is covered. In each, send a descriptor whose `attributes` hold
      `spec.jobTemplate.spec.template.spec.containers[0].resources.requests.cpu = "250m"` (ten segments, CronJob depth)
      plus `tags`, over `TestHarness`. Assert with `proto.Equal` that the received `attributes` equal the sent ones and
      that `tags` are unchanged. Do not use `t.Parallel()` on subtests that share the harness

**Checkpoint**: US1 passes with the generated field alone.

---

## Phase 4: User Story 2 - Size bound in both validation layers (P1)

**Goal**: `proto.Size(attributes) <= 65536` is enforced identically in `pluginsdk` and `testing`.

**Independent Test**: At-limit and over-limit tables in both packages.

- [X] T006 [P] [US2] Add a shared test helper `attributesOfSize(t, n int) *structpb.Struct` in each test package
      (`sdk/go/pluginsdk/attributes_test.go` and `sdk/go/testing/contract_test.go`). It builds a one-key struct whose
      string value is padded, and adjusts the padding until `proto.Size` equals exactly `n`, because varint length
      prefixes can skip one size at a boundary. It asserts that result
- [X] T007 [P] [US2] Add cases to `TestValidateResourceDescriptor` in `sdk/go/pluginsdk/batch_test.go`: nil attributes
      pass; empty struct passes; `proto.Size == MaxAttributesBytes` (65536) passes; 65537 fails with
      `codes.InvalidArgument` and a message that contains `attributes`, `65537`, and `65536`; and `MaxAttributesBytes ==
      65536` is pinned as a literal
- [X] T008 [P] [US2] Add cases to the `ValidateResourceDescriptor` tests in `sdk/go/testing/contract_test.go`: at limit
      passes; over limit returns an error that satisfies `errors.Is(err, plugintesting.ErrAttributesTooLarge)` with a
      `ContractError` whose `Field` is `attributes`, and whose message contains the size (`65537`) and the limit
      (`65536`); `plugintesting.MaxAttributesBytes == 65536`
- [X] T009 [US2] Implement in `sdk/go/pluginsdk/batch.go`: add the constant `MaxAttributesBytes = 64 << 10` to the
      DoS-guard block, with a comment giving the reason (a bound that sits well under the 1 MB Connect and 4 MB gRPC
      message limits; the wire size is what counts against them). In `ValidateResourceDescriptor`, after the tag checks,
      add `if attrs := resource.GetAttributes(); attrs != nil { if n := proto.Size(attrs); n > MaxAttributesBytes {
      return status.Errorf(codes.InvalidArgument, "attributes size %d bytes exceeds maximum %d", n, MaxAttributesBytes)
      } }`. Add the rule to the function's doc list
- [X] T010 [US2] Implement in `sdk/go/testing/contract.go`: add the constant `MaxAttributesBytes = 64 << 10` with the
      reason, add the sentinel `ErrAttributesTooLarge = fmt.Errorf("attributes exceed maximum encoded size of %d bytes",
      MaxAttributesBytes)` beside `ErrTagValueTooLong`, so the message names the limit, and add the nil-guarded size
      check in `ValidateResourceDescriptor` that returns `NewContractError("attributes", n, ErrAttributesTooLarge)`
- [X] T011 [US2] Add contract-suite cases `ResourceDescriptor_AttributesAtLimitAccepted` and
      `ResourceDescriptor_AttributesOverLimitRejected` to `registerResourceDescriptorTests` in
      `sdk/go/testing/contract.go`, and update any test that counts registered contract tests
- [X] T012 [US2] Run `go test ./sdk/go/pluginsdk/ -run '^$' -bench ValidateResourceDescriptor -benchmem` and confirm 0
      allocs/op without attributes, matching the T001 baseline (FR-017, SC-004)

**Checkpoint**: Both layers agree at 65536 and 65537.

---

## Phase 5: User Story 3 - Tag value limit 2048 (P2)

**Goal**: Both layers accept 2048 bytes and reject 2049. Counts and key limits do not change.

**Independent Test**: Boundary tables in both packages.

- [X] T013 [P] [US3] In `sdk/go/pluginsdk/batch_test.go`, pin `MaxTagValueLength == 2048`, `MaxTagKeyLength == 128`, and
      `MaxTagsPerResource == 256` as literals, and add 2048-byte (pass) and 2049-byte (fail) tag value cases to
      `TestValidateResourceDescriptor` and `TestValidateBatchCostRequestResourceDescriptorValidation`
- [X] T014 [P] [US3] In `sdk/go/testing/contract_test.go`, pin `MaxTagValueLength == 2048`, `MaxTagCount == 50`, and
      `MaxTagKeyLength == 128`, and add 2048 (pass) and 2049 (fail) cases for `ValidateTags` and
      `ValidateResourceDescriptor`
- [X] T015 [US3] Change `MaxTagValueLength` from 256 to 2048 in `sdk/go/pluginsdk/batch.go` and
      `sdk/go/testing/contract.go`. Update the `ValidateResourceDescriptor` doc list ("tag values must not exceed
      MaxTagValueLength (2048 bytes)") and the DoS-guard block comment so its comparison with `testing/contract.go` is
      accurate: "Tag key and value lengths match the contract; counts and other fields are more generous" (FR-008).
      State that lengths are bytes

**Checkpoint**: The two layers agree on the new tag limit.

---

## Phase 6: User Story 4 - Dotted-path accessor (P2)

**Goal**: `pluginsdk.AttributeValue(attrs *structpb.Struct, path string) (*structpb.Value, bool)`.

**Independent Test**: Table tests for every row of the data-model accessor table.

- [X] T016 [US4] Write `TestAttributeValue` in `sdk/go/pluginsdk/attributes_test.go` as a table test over a fixture
      struct. The cases are: a top-level hit; a nested hit; a list index hit (`containers.0.name`); the ten-segment
      CronJob path; a missing key; an out-of-range index; a negative index (`-1`); a non-numeric index on a list; a
      segment through a scalar; an explicit null (found, `NullValue`); a nil struct; an empty path; and empty segments
      (`a..b`, `.a`, `a.`). Add `BenchmarkAttributeValue` and assert 0 allocs with `testing.AllocsPerRun`
- [X] T017 [US4] Implement `AttributeValue` in the new `sdk/go/pluginsdk/attributes.go` (Apache header). Godoc covers
      the path grammar, that a key containing `.` cannot be addressed, and that a missing segment is reported as `false`
      and never as an error. Walk iteratively, without recursion. Scan segments with `strings.IndexByte` and parse
      indexes without `strconv` allocations. Include no provider-specific paths

**Checkpoint**: The accessor passes and makes 0 allocs/op.

---

## Phase 7: User Story 5 - Conformance exercises attributes (P2)

**Goal**: A Basic-level conformance test sends nested attributes, and the mock, which ignores them, passes.

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestConformance|TestRPCCorrectness'`.

- [X] T018 [US5] Add `TestRPCCorrectnessGetProjectedCostWithAttributes` in `sdk/go/testing/rpc_correctness_test.go`.
      Look up `RPCCorrectness_GetProjectedCostWithAttributes` in `RPCCorrectnessTests()`, run it against
      `NewMockPlugin()`, and assert success. Assert that the test is `ConformanceLevelBasic`
- [X] T019 [US5] Implement `testGetProjectedCostWithAttributesRPC` and its `ConformanceSuiteTest` entry (`MinLevel:
      ConformanceLevelBasic`, `Category: CategoryRPCCorrectness`) in `sdk/go/testing/rpc_correctness.go`. It builds the
      ten-segment nested `attributes` with `structpb.NewStruct` and adds `tags`, sends `GetProjectedCost`, and validates
      the response with `ValidateProjectedCostResponse`. Its details string states that attributes were accepted
- [X] T020 [US5] Update every test that asserts the number of RPC correctness or conformance tests (search
      `sdk/go/testing/*_test.go` for counts), then run `go test ./sdk/go/testing/` and `go test -v -run TestConformance
      ./sdk/go/testing/`
- [X] T021 [P] [US5] Add a TypeScript round-trip test in
      `sdk/typescript/packages/client/test/resource-attributes.test.ts`. Use `create(ResourceDescriptorSchema,
      {attributes: {spec: {replicas: 3, containers: [{resources: {requests: {cpu: "250m"}}}]}}})` with `toBinary` and
      `fromBinary`, and with `toJson` and `fromJson`, and assert deep equality. Run `cd sdk/typescript && npm ci && npm
      run build && npm test`

**Checkpoint**: The mock passes every level, and the TypeScript bindings round-trip.

---

## Phase 8: User Story 6 - Documentation (P2)

**Goal**: Every non-historical document states the new field, the rules, and the limits (FR-013 to FR-015,
SC-005).

**Independent Test**: The quickstart §6 search finds nothing stale.

- [X] T022 [P] [US6] Update `docs/PROPERTY_MAPPING.md`. Add a "Structured Attributes" section covering: what
      `attributes` carries; the redaction rule and the credential-like key list; prefer `attributes` and fall back to
      `tags`; the 64 KiB bound and the batch split rule with both transport limits; the 2^53 guidance; and an
      `AttributeValue` example. Rewrite the "How the Host Hands Properties" bullets at lines 82-86 so they no longer say
      that the structured form exists only on `EstimateCost`, and give tag value length 2048. Note that the reference
      host does not populate `attributes` until rshade/finfocus#1525
- [X] T023 [P] [US6] Update `PLUGIN_DEVELOPER_GUIDE.md`. Change the sample validator limits (`len(value) > 256` becomes
      2048), and add an attributes size check to the sample. Change the GetProjectedCost section to "prefer
      `attributes`, fall back to `tags`", with an `AttributeValue` snippet. Add "do not log `attributes` verbatim"
      beside the credentials guidance in "Handling Credentials Safely"
- [X] T024 [P] [US6] Update `sdk/go/pluginsdk/README.md`. Document `AttributeValue` with an example,
      `MaxAttributesBytes`, the new `MaxTagValueLength`, and the `ValidateResourceDescriptor` attributes rule
- [X] T025 [P] [US6] Update `sdk/go/testing/README.md`. Document `MaxAttributesBytes`, `ErrAttributesTooLarge`, the 2048
      tag value limit, and the new conformance test `RPCCorrectness_GetProjectedCostWithAttributes`
- [X] T026 [US6] Audit the remaining docs from research R9 (`README.md`, `ROADMAP.md`, `docs/ADVANCED_PATTERNS.md`,
      `docs/plugin-registry-specification.md`, `docs/usage-source.md`, `docs/allocator.md`, `sdk/go/pricing/README.md`,
      `sdk/go/registry/README.md`, `sdk/typescript/README.md`, and `sdk/go/CLAUDE.md`). Update every passage that lists
      `ResourceDescriptor` fields, describes how inputs reach a plugin, or states a tag value limit
- [X] T027 [US6] Add a "Resource Attributes Pattern (596-resource-descriptor-attributes)" section, an Active
      Technologies entry, and a Recent Changes entry to `CLAUDE.md` by hand, not with `update-agent-context.sh`. Mirror
      them in `AGENTS.md` and `GEMINI.md` if those are separate files and not symlinks
- [X] T028 [US6] Run the quickstart §6 grep and `make lint-markdown`, and fix any finding

---

## Phase 9: Polish

- [X] T029 Run the full gate set: `make generate && git diff --exit-code -- sdk/go/proto`, `make buf-lint`, `buf
      breaking`, `make test`, `go test -v -tags=integration ./sdk/go/testing/`, `make lint-go`, `make lint-markdown`,
      `make lint-yaml`, and `make validate-npm`
- [X] T030 Mark the tasks complete and update `checklists/requirements.md` if the spec changed

## Dependencies

- Phase 2 (T002–T004) blocks every story.
- US1 (T005) needs only Phase 2.
- US2 (T006–T012) and US3 (T013–T015) both edit `batch.go`, `contract.go`, and their tests, so run them in
  sequence (US2, then US3).
- US4 (T016–T017) is independent once Phase 2 is done. US2's T006 helper lives in the same test file, so
  coordinate.
- US5 (T018–T021) needs Phase 2. T021 (TypeScript) can run in parallel with the Go tasks.
- US6 (T022–T028) follows US2 to US5, so the docs describe the final API.

## Parallel Examples

- After T004: T005 (US1), T016 (US4 tests), and T021 (TypeScript) touch different files.
- In US2: T007 and T008 run in parallel.
- In US6: T022 to T025 run in parallel.

## Implementation Strategy

The MVP is Phase 2 plus US1: the field alone unblocks rshade/finfocus#1525. US2 must ship in the same PR,
because an unbounded field is not acceptable. Everything else lands in the same PR, as the constitution's
same-PR documentation rule requires.

## Phase 10: Convergence

- [X] T031 Record in research.md R6 why the TypeScript `ResourceDescriptorBuilder` gained `withAttributes`
  per plan: research R6 (unrequested)
