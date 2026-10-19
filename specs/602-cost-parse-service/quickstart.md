# Quickstart: Validate Cost Parse Service

**Status**: Implemented and validated on 2026-10-18. Commands and recorded outcomes below cover the completed feature.

## Prerequisites

Use the repository's Go 1.27.1, Node 24.21.0+, npm 11.19.0+, Buf, golangci-lint and YAML/Markdown tooling.
Install the pinned toolchain with the project's normal mise workflow. Install JavaScript dependencies using the
repository lockfiles. Run commands from the repository root unless a command explicitly selects another directory.
Generation requires access to the existing Buf remote plugins. Do not change the pinned Connect Go generator.

```bash
npm ci
npm --prefix sdk/typescript ci
```

Review [the wire contract](contracts/parse.proto), [SDK/error contract](contracts/sdk-and-errors.md), and
[data model](data-model.md). A synthetic mock fixture proves protocol behavior; a vendor parser is needed only to
validate a real format. No cloud account, deployment, pricing backend or infrastructure mutation is needed for mock tests.

## Generate and check compatibility

After tests define the contract and canonical proto files are added:

```bash
make generate
make buf-lint
buf breaking --against '.git#branch=main'
git diff --check
```

Use the repository-managed Buf on PATH. Expected: both SDK bindings are regenerated, the service and capability are
additive, no existing field changes, and no Connect v2 imports or unrelated generated rewrites appear.

## Exercise Go parsing behavior

New Go acceptance tests use TestCostParse names so this focused selection runs
real tests. Check verbose output for nonzero tests; an empty matching set is not successful validation.

```bash
go test -v ./sdk/go/pluginsdk ./sdk/go/testing -run TestCostParse -count=1
go test ./sdk/go/registry -run TestSchemaDrift -count=1
go test -race ./sdk/go/pluginsdk ./sdk/go/testing
```

Expected scenarios:

1. Discover the mock's advertised format, modes, positive byte cap, and COST_PARSE capability. Parser-only explicit
   capability configuration omits inherited pricing capabilities; inference appends parsing for mixed plugins.
2. Submit the mock's valid named content and path fixtures through the harness. All returned descriptors validate
   and have response-unique source identities; no extra type-resolution call is needed.
3. Reject missing input, typed-nil file selection, empty root/set, unsafe/duplicate paths, malformed format and
   oversized options as InvalidArgument naming the field. Boundary-equal options and content sizes pass.
4. Send raw binary payloads containing both input fields in both orders, including an empty root member. Both gRPC
   and Connect binary paths return InvalidArgument and never invoke the provider; JSON conflicts also fail.
   Repeated instances of a single arm retain normal protobuf behavior.
5. Reject an unadvertised format/mode with FailedPrecondition and the correct ErrorInfo reason/domain. Reject excess
   raw input with ResourceExhausted. Preserve handler errors/details; missing or stub providers return Unimplemented.
6. Return a resource with an unresolved nested attribute and a source-linked resource_skipped warning. Verify the
   unresolved value is absent, warning code/message/severity are valid, and skipped sources need not be returned.
7. Validate malformed responses separately: nil entries/descriptors, duplicate source IDs, present/null unknown values,
   malformed paths, unspecified severity, empty warning code/message, and invalid expiration must fail validation.
8. Compare resources and warnings across 100 identical requests with unchanged dependencies, ignoring advisory expiry.
   Use injected listeners for Serve tests; check registration health without probing handler functionality at startup.
9. In a controlled temporary root, verify mock byte accounting rejects reads above its limit and a mock containment
   test rejects symlinks escaping the root. This proves the mock's policy, not generic enforcement for every plugin.

RunCostParseConformance on a real plugin takes caller-provided valid format/mode fixtures and advertised limits.
Checks target structure and errors, never vendor-specific resource contents. Record explicit skips when an unsupported
mode cannot be selected because both modes are accepted.

## Exercise TypeScript clients and REST

```bash
npm --prefix sdk/typescript run build
npm --prefix sdk/typescript run test
npm --prefix sdk/typescript run lint
```

Expected: CostParseClient is exported and honors injected/default transports. Parsing tests run against the generated
service shape, preserve byte arrays and timestamps, and preserve Connect error details. REST gateway tests configure
costParseClient and POST `/finfocus.v1.CostParseService/Parse`; absent client follows the existing 404 behavior.

A content-mode request in protobuf JSON uses a base64 content string:

```json
{
  "format": "fixture",
  "files": {
    "files": [
      { "path": "input.fixture", "content": "AP9B" }
    ]
  }
}
```

The bytes are 0x00, 0xff, 0x41. Use the mock's documented fixture data when checking semantic parse output; this example
specifically checks byte transport. Timestamps must round-trip as RFC 3339 strings with subsecond precision, and
ParseLimits byte values must retain int64 precision. REST errors retain existing category/status mapping; structured
ErrorInfo details are checked on Connect/gRPC clients, because the existing REST envelope does not expose them.

The existing REST 1 MiB body cap, base64 expansion, and server message ceilings are independent of advertised raw
input bytes. Use small fixtures here; no limit increase is assumed.

## Performance and full repository validation

Record baseline binaries from main before implementation and candidate binaries afterward. Store artifacts outside the
source tree. Run repeated benchmark comparisons with the same toolchain and environment; compare distributions,
not a single sub-nanosecond sample. Existing benchmark alerts remain informational.

```bash
go test ./sdk/go/testing ./sdk/go/pluginsdk -run '^$' -bench 'Parse|ValidateResourceDescriptor' -benchmem -count=10
make validate
```

Expected: new parsing validation reports bounded scaling and allocations, existing validators show no repeatable
slowdown, and full tests/lint/schemas/examples pass. If aggregate validation times out, record the incomplete aggregate
and run its constituent commands; do not claim it passed. Check README examples, generated symbols, godoc coverage,
links and synchronized Go/TypeScript documentation before any commit.

## Implementation evidence

Setup on 2026-10-18: Go go1.27.1 linux/amd64, Node v24.21.0, npm 11.19.0, Buf 1.32.1.
Candidate starting revision: `4bb2ee0cb6994193f605330bd2a94501e4facce5`.
Main baseline: `95937184813347845e876ab1b05dd199b6476dc3`. Connect generator remains pinned to v1.21.0.
Existing Git/npm ignore files cover Go, Node, build, environment and temporary artifacts; no additions needed.

Baseline sources and prebuilt binaries are under `/tmp/finfocus-cost-parse-baseline`. Reproduce from repository root:

```bash
mkdir -p /tmp/finfocus-cost-parse-baseline
git archive main | tar -x -C /tmp/finfocus-cost-parse-baseline
cd /tmp/finfocus-cost-parse-baseline
go test -c -o testing.test ./sdk/go/testing
go test -c -o pluginsdk.test ./sdk/go/pluginsdk
./testing.test -test.run '^$' -test.bench 'BenchmarkValidate' -test.benchmem -test.count 10
./pluginsdk.test -test.run '^$' -test.bench 'BenchmarkInferCapabilities' -test.benchmem -test.count 10
```

Both baseline binaries compiled successfully with the candidate toolchain. Baseline benchmark names include
BenchmarkValidateResourceDescriptor, BenchmarkValidateResourceDescriptor_WithTags, BenchmarkValidateTimeRange,
BenchmarkValidateTags, BenchmarkValidateGetActualCostRequest, BenchmarkValidateEstimateCostRequest,
BenchmarkValidateGetRecommendationsRequest, BenchmarkValidateSupportsRequest, BenchmarkRunStandardContractTests,
BenchmarkInferCapabilitiesRPC and BenchmarkInferCapabilitiesExplicit.

T003 initial RED: `go test ./sdk/go/testing -run TestCostParseContract -count=1` failed with undefined
ParseRequest, ParseResponse and COST_PARSE symbols before canonical definitions were added.

T004/T005: root `make generate` succeeded. `TestCostParseContract` passes with both input arms,
field-number assertions, source-linked responses and nanosecond timestamps. Only intended enum additions and
new parsing bindings were generated; the Connect v1 pin is unchanged.

T006/T007 initial RED reported missing ParseLimits/ValidateParseRequest/ValidateParseResponse APIs.
After T008–T010, `go test ./sdk/go/testing -run TestCostParse -count=1` passed.
The neutral descriptor uses the existing `custom` provider; an initial `fixture` provider was rejected by the
existing descriptor rules and corrected in the test fixture. No descriptor validator was changed.

US1 T011–T014 RED: missing metadata/provider/mock symbols and schema rejection of cost_parse were observed.
T015–T022 focused validation passed:

```bash
go test ./sdk/go/pluginsdk ./sdk/go/testing ./sdk/go/registry \
  -run 'TestCostParse|TestSchemaDrift|TestIsValidCapability|TestAllFunctions|TestMetrics_LatencyAccuracy' -count=1
```

Both Serve transports accept content and root fixtures, advertise parsing and limits, preserve missing/stub
Unimplemented behavior, and reject invalid successful provider responses as Internal. Connect health reflects
interface registration only, including generated stubs, and never invokes Parse. gRPC Serve retains its existing
service-registration behavior. MockCostParser emits custom:compute:Instance descriptors with native fixture source IDs;
the existing pricing mock prices them directly with zero type-resolution calls. Content and path fixtures produce
identical resources/warnings across 100 calls. Constructor and response descriptors are deep-cloned.

The initial broader package run found outdated capability boundary/count assertions, now updated for 19/33.
It also failed existing TestMetrics_LatencyAccuracy (118 ms against a 100 ± 10 ms expectation); a focused rerun passed.
The initial mock compatibility test used the gRPC Name signature as pluginsdk.Plugin and omitted the existing
DISMISS_RECOMMENDATIONS capability; its adapter and expected baseline were corrected without changing MockPlugin.

US2 T023–T031: error tests passed against the existing US1 dispatch. Raw codec tests first failed all eight
binary conflict scenarios (both orders, both transports, empty root included). Provider-policy tests first failed
on missing WithMockParseLimits. After parser-local codec and bounded-root implementation:

```bash
go test ./sdk/go/pluginsdk ./sdk/go/testing -run TestCostParse -count=1
go test -race ./sdk/go/pluginsdk ./sdk/go/testing -run TestCostParse -count=1
```

Both commands passed. Conflicts return InvalidArgument naming input and zero provider calls. Repeated same-arm
fields, unknown fields, malformed protobuf and other RPC codec behavior retain normal semantics. JSON conflicts
already rejected correctly. Unsupported ErrorInfo reasons/domain and recognized provider status/details survive
both transports. Unknown/plain errors and malformed successful responses become sanitized Internal.
The mock rejects unknown fixture options, names malformed files/physical lines, reads every .fixture in lexical
walk order with bounded raw-byte totals, and rejects escaping file symlinks using os.OpenRoot containment.
Exact budgets pass; one-byte-over budgets exhaust resources in both modes, including multiple files.

US3 T032–T037: partial transport cases passed using existing dispatch/validators. Mock partial tests first failed
unsupported unknown/skip/notice lines and duplicate-source error categories. After implementation, focused
TestCostParse tests passed in pluginsdk/testing. Source identities, nested unknown casing and known zero/empty
values survive both transports. All-skipped and empty responses validate; malformed provider responses become Internal.
Caller descriptors, requests and later fixture parses remain unchanged.
Response benchmark smoke run passed valid, rejected, nested_unknown, warning_heavy and descriptor_near_limit
cases using `go test ./sdk/go/testing -run '^$' -bench BenchmarkCostParseResponse -benchmem -benchtime=10ms -count=1`.
The valid response path reported zero allocations in this run; final repeated measurements follow in T054.

US4 T038–T049: conformance/freshness tests first failed missing APIs; client test suites first failed missing
cost-parse module/exports; 18 gateway tests first failed on the missing route (absent-client case already passed).
After implementation, all eight Go conformance scenarios run over bufconn; explicit skip reasons cover supported
both modes and missing content fixture. Every supplied fixture is cloned/validated and exercised for 100 calls
with per-call 10-second deadlines. Pure input omits expiry; controlled dependency expiry is cloned and may advance
without changing resources/warnings. Valid expired timestamps remain advisory.

TypeScript full workspace build/test/lint passed: client 206 tests, middleware 43, framework plugins 8.
New client coverage includes both arms, AP9B bytes, nanosecond RFC 3339 timestamps, ErrorInfo and exact bigint
metadata through MaxInt64. The ES2018 source target uses the existing protoInt64 parser rather than direct
BigInt global declarations. All 19 new gateway cases pass, including exact/over 1 MiB body boundaries with
only three raw content bytes, sanitization, status mapping and JSON oneof conflicts.

Independent review found three issues and regression tests reproduced each before fixes: provider-returned
metadata bypassed the parse advertisement gate; a FIFO fixture could block during open; conformance rejected
providers with empty raw capabilities requesting inference. Final discovery validation now covers effective
capabilities, contained mock opens are nonblocking before regular-file checks, and conformance accepts the
SDK's documented empty-capability inference behavior. Focused TestCostParse tests pass after all fixes.

The full race suites passed before review fixes; the final full rerun is recorded below. An initial make validate
passed all Go tests, pinned Go lint and Buf/Markdown lint, then stopped because yamllint was unavailable.
The direct PATH golangci-lint is 2.15.0, while Makefile uses the mise-pinned 2.13.2. The latter reports zero
issues. The newer direct binary reports four complexity findings byte-identical in baseline main and pre-feature
HEAD; no unrelated function changes were made to silence those tool-version differences.

T050–T052 documentation: root/Go/plugin/testing/TypeScript READMEs and docs/cost-parse.md are synchronized.
The planned sdk/go/README.md did not exist and was created. The conformance Go example was extracted,
compiled and run successfully; provider configuration Go example compiles/validates and the TypeScript
example passes tsc under ES2018 declarations. Markdown lint passed all 783 discovered files.

## Final performance evidence

Prebuilt binaries use the same Go 1.27.1 toolchain on QEMU Virtual CPU version 2.5+ (linux/amd64).
Every row below has ten repetitions. Raw outputs remain under /tmp/finfocus-cost-parse-baseline.
Validation and existing benchmarks used 100 ms samples; wire codec benchmarks used 10 ms samples.
The codec measurements include the full RPC transport, tracing and serialization costs, rather than isolating
the scanner; a no-op logger prevents rejected-request logs from corrupting benchmark rows.

| Output file | Benchmark | Samples | Median ns/op | Range ns/op | B/op | allocs/op |
| --- | --- | --- | --- | --- | --- | --- |
| descriptor-baseline-final.txt | BenchmarkValidateResourceDescriptor | 10 | 2.57 | 2.29–2.91 | 0 | 0 |
| descriptor-baseline-final.txt | BenchmarkValidateResourceDescriptor_WithTags | 10 | 35.99 | 35.31–37.61 | 0 | 0 |
| descriptor-candidate.txt | BenchmarkValidateResourceDescriptor | 10 | 2.42 | 1.82–2.88 | 0 | 0 |
| descriptor-candidate.txt | BenchmarkValidateResourceDescriptor_WithTags | 10 | 36.77 | 35.18–38.15 | 0 | 0 |
| capabilities-baseline-final.txt | BenchmarkInferCapabilitiesDirect | 10 | 48.49 | 45.20–53.36 | 64 | 1 |
| capabilities-baseline-final.txt | BenchmarkInferCapabilitiesMinimal | 10 | 45.65 | 43.03–50.17 | 64 | 1 |
| capabilities-baseline-final.txt | BenchmarkInferCapabilitiesRPC | 10 | 812.60 | 758.70–936.10 | 872 | 6 |
| capabilities-baseline-final.txt | BenchmarkInferCapabilitiesExplicit | 10 | 715.10 | 657.70–797.40 | 864 | 6 |
| capabilities-candidate.txt | BenchmarkInferCapabilitiesDirect | 10 | 49.37 | 45.63–55.55 | 64 | 1 |
| capabilities-candidate.txt | BenchmarkInferCapabilitiesMinimal | 10 | 46.73 | 42.19–49.43 | 64 | 1 |
| capabilities-candidate.txt | BenchmarkInferCapabilitiesRPC | 10 | 821.60 | 741.70–925.80 | 872 | 6 |
| capabilities-candidate.txt | BenchmarkInferCapabilitiesExplicit | 10 | 747.30 | 684.10–788.50 | 864 | 6 |
| parse-validation-candidate.txt | BenchmarkCostParseRequest/valid | 10 | 102.40 | 82.76–131.80 | 16 | 1 |
| parse-validation-candidate.txt | BenchmarkCostParseRequest/rejected | 10 | 190.45 | 161.90–215.90 | 176 | 3 |
| parse-validation-candidate.txt | BenchmarkCostParseRequest/near_limit | 10 | 107.40 | 88.52–122.70 | 16 | 1 |
| parse-validation-candidate.txt | BenchmarkCostParseMetadata/valid | 10 | 164.80 | 131.10–185.30 | 117 | 4 |
| parse-validation-candidate.txt | BenchmarkCostParseMetadata/rejected | 10 | 342.10 | 294.90–385.20 | 229 | 7 |
| parse-validation-candidate.txt | BenchmarkCostParseMetadata/max_int64 | 10 | 197.55 | 177.80–223.50 | 136 | 4 |
| parse-validation-candidate.txt | BenchmarkCostParseResponse/valid | 10 | 26.24 | 20.68–29.95 | 0 | 0 |
| parse-validation-candidate.txt | BenchmarkCostParseResponse/rejected | 10 | 148.85 | 136.30–179.70 | 80 | 2 |
| parse-validation-candidate.txt | BenchmarkCostParseResponse/nested_unknown | 10 | 103.80 | 82.81–124.60 | 32 | 1 |
| parse-validation-candidate.txt | BenchmarkCostParseResponse/warning_heavy | 10 | 167.60 | 127.40–174.10 | 0 | 0 |
| parse-validation-candidate.txt | BenchmarkCostParseResponse/descriptor_near_limit | 10 | 197.10 | 181.10–244.90 | 16 | 1 |
| parse-codec-candidate.txt | BenchmarkCostParseBinaryGuard/normal | 10 | 174393.00 | 139850.00–253394.00 | 10382.5 | 164 |
| parse-codec-candidate.txt | BenchmarkCostParseBinaryGuard/rejected | 10 | 191827.50 | 86429.00–228955.00 | 10227 | 151 |
| parse-codec-candidate.txt | BenchmarkCostParseBinaryGuard/near_limit | 10 | 518254.00 | 334536.00–770329.00 | 480574 | 180 |
| parse-codec-candidate.txt | BenchmarkCostParseBinaryGuard/non_parse | 10 | 202437.50 | 146384.00–253651.00 | 13142.5 | 250 |

Existing descriptor validation sources/call sites are unchanged. Baseline/candidate distributions overlap,
and existing allocation counts are unchanged. Observed median differences in this run are about -6% for
plain descriptors, +2% with tags, +1% inferred capability RPC and +4.5% explicit capability RPC; these samples
do not establish a repeatable parsing-attributable regression. CI benchmark alerts remain informational at
the existing threshold. Use the retained binaries for further interleaved measurements on a quieter machine.

Reproduce the new samples with:

```bash
cd /tmp/finfocus-cost-parse-baseline
./testing-candidate.test -test.run '^$' -test.bench BenchmarkCostParse -test.benchmem -test.benchtime=100ms -test.count=10
./pluginsdk-candidate.test -test.run '^$' -test.bench BenchmarkCostParseBinaryGuard -test.benchmem -test.benchtime=10ms -test.count=10
```

## Final validation and requirement reconciliation

All implementation tasks T001–T057 are complete. Final commands and actual results:

| Check | Result |
| --- | --- |
| PATH=/tmp/finfocus-validation-tools/bin:$PATH make validate | PASS: all Go packages, pinned Go lint, Buf lint, Markdown/YAML lint, schemas and examples |
| go test -race ./sdk/go/pluginsdk ./sdk/go/testing | PASS after the three review fixes |
| npm --prefix sdk/typescript run build | PASS all three workspaces |
| npm --prefix sdk/typescript test | PASS: 206 client + 43 middleware + 8 framework = 257 tests |
| npm --prefix sdk/typescript run lint | PASS all three workspaces |
| make generate; make buf-lint; buf breaking --against '.git#branch=main' | PASS; additive generation and no Connect v2 |
| Whitespace checks | PASS except the generator-standard terminal blank line in new parse_pb.ts; see publication note below |
| Extracted Go conformance and configuration examples | PASS: compiled and executed |
| Extracted TypeScript parsing example | PASS under ES2018 type checking |
| Exported declaration godoc audit | 1665/1738 = 95.80% across Go SDK; new parse sources 43/43 = 100% |
| Local feature-document links and new source Apache headers | PASS: 5 documents, 26 sources |

The missing YAML tool was installed only under /tmp using uv; no project dependencies or lockfiles changed.
The successful aggregate used the temporary tool directory in PATH as shown above. The aggregate was not
reported as passing until its process exited successfully. No aggregate timeout occurred.
Raw validation logs are /tmp/finfocus-parse-make-validate-final.log, /tmp/finfocus-parse-race-final.log,
/tmp/finfocus-parse-ts-build-final.log, /tmp/finfocus-parse-ts-tests-final.log and
/tmp/finfocus-parse-ts-lint-final.log. Benchmark distributions and reproduction commands appear above.

| Requirements | Evidence |
| --- | --- |
| FR-001, FR-010, FR-021; SC-001 | Additive Parse protocol; direct mock pricing without ResolveResourceTypes; existing full suites and Buf breaking pass |
| FR-002, FR-016; SC-007 | Metadata/config/provider gates; capability inference and explicit parser-only override; missing/stub Unimplemented; schema drift tests |
| FR-003–FR-009; SC-002 | Request/limits/error/codec/mock-policy tests on both transports; exact byte/options boundaries, strict raw XOR, field/file/line errors and ErrorInfo |
| FR-011–FR-013; SC-003, SC-004 | Shared response, partial transport and partial fixture tests; unique sources, absent nested unknowns, known zero/empty values and separate skipped warnings |
| FR-014, FR-015; SC-005 | 100-call fixture/conformance equality, controlled advancing expiry, absent/expired valid advisory timestamps and both-mode boundary tests |
| FR-017, FR-018 | Real gRPC/Connect serving, parser registration health with no invocation probe, shared validators and non-parser codec compatibility |
| FR-019 | Separate neutral mock and configurable bufconn conformance; all eight named scenarios, explicit skips, bounded probes and ten-second call deadlines |
| FR-020; SC-006 | Generated Go/TS parity, binary AP9B/nanosecond round trips, ErrorInfo and REST mapping tests, exact bigint metadata |
| FR-022; SC-008 | Cost-parse guide and compiling README examples cover option ownership, containment, explicit/unique/ambiguous selection and content/dependency-aware caching |

All constitution gates in plan.md were re-evaluated against these results. Checklist files remain unchanged: the
implementation skill treats them as read-only requirements-quality gates; all original 16 entries remain checked.
The T057 checklist-note instruction was therefore fulfilled by recording reconciliation here rather than editing
reviewer-owned checklist content. Final extension configuration parsed successfully with no after_implement hooks.
Implementation validation completed before publication. Existing source-design artifacts were preserved.

## Publication verification

Precommit staging exposed the generator-standard terminal blank line in the new TypeScript parse_pb.ts.
The existing scoring_pb.ts and costsource_pb.ts also end with two newline bytes. Generated output was
preserved without manual edits. Normal staged whitespace checks pass for all other files; the new generated
file passes with only blank-at-eof checking disabled. Earlier unstaged git diff checks did not include new
untracked files. The complete staged check and scoped exception were verified before committing.
