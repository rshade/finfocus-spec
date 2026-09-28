# Quickstart: Validating the Allocator Service

Runnable checks that prove 052 works end to end. The API is defined in
[contracts/go-sdk-api.md](./contracts/go-sdk-api.md) and
[contracts/typescript-api.md](./contracts/typescript-api.md), and the rules are in
[data-model.md](./data-model.md).

## Prerequisites

- Go per `go.mod`, Node.js ≥ 22, `npm install` done at the repository root and in
  `sdk/typescript/packages/client`
- Work on the feature branch: `git checkout -b 052-allocator-allocate` (research R14)

## 1. Contract compiles and is additive

```bash
make generate                      # produces allocation.pb.go, allocation_grpc.pb.go,
                                   # pbcconnect/allocation.connect.go, allocation_pb.ts
bin/buf lint
bin/buf breaking --against '.git#branch=main'
git status --short sdk/go/proto    # only allocation* files and enums.pb.go change
```

**Expected**: lint and breaking pass. If unrelated `*.connect.go` doc comments change, restore them
with `git checkout` (CLAUDE.md, unpinned remote plugins).

## 2. Capability discovery (US5)

```bash
go test ./sdk/go/pluginsdk/ -run 'Capabilit|LegacyCapability|IsValidCapability' -v
```

**Expected**:

- An allocator with inferred capabilities reports `PLUGIN_CAPABILITY_ALLOCATION` and
  `supports_allocation=true`.
- With `WithCapabilities(PLUGIN_CAPABILITY_ALLOCATION)`, exactly that capability is reported.
- `IsValidCapability(15)` is true and `IsValidCapability(16)` is false.
- The startup warning is logged for an allocator with no explicit capabilities.

## 3. Serving over both transports (US1, US3)

```bash
go test ./sdk/go/pluginsdk/ -run 'Allocator' -v
```

**Expected**:

- The reference allocator, served through `Serve`, answers the two-node fixture over gRPC and over
  Connect with identical rows, effective policy, digest, and warnings.
- A failing allocator's error arrives as InvalidArgument with the same message on both transports.
- In Connect mode the health check reports `finfocus.v1.AllocatorService` as SERVING.
- A plugin without `Allocate` has no `AllocatorService` registered, and a plugin with both
  `GetStats` and `Allocate` answers both.

## 4. Verification helpers (US2)

```bash
go test ./sdk/go/testing/ -run 'Conservation|ValidateAllocate|ResolveCurrency' -v
```

**Expected**: every rule in data-model.md (Q1–Q5, P1–P7, C1–C4) has a passing case and a failing
case (SC-005). The accepted inputs include:

- balanced rows, and a difference within 1e-6 relative
- a zero total with nothing priced
- `priced=false` resources excluded from the expected total
- cluster rows with zero CPU and memory portions

The rejected inputs include:

- overshoot and shortfall, with the error stating expected, actual, and difference
- NaN totals
- negative or NaN epsilon
- mixed currencies
- `USD`, empty, `EUR` (mixed); while `USD`, empty, `USD` resolves to `USD`, and all empty resolves
  to `USD`
- duplicate `(kind, id)`

## 5. Strict policy decoding

```bash
go test ./sdk/go/pluginsdk/ -run 'DecodePolicy' -v
```

**Expected**:

- `{"node_split":{"cpu":1}}` fails, naming `node_split.cpu`, with `status.Code(err) ==
  InvalidArgument`.
- Empty, whitespace, `null`, and `{}` leave the defaults untouched.
- Trailing data and malformed JSON fail.
- A nested override changes only that field.
- A supplied array fully replaces the default, with no carried-over elements.

## 6. Conformance suite (US4)

```bash
go test ./sdk/go/testing/ -run 'AllocatorConformance' -v
```

**Expected**:

- The reference allocator passes all 12 named subtests (SC-004).
- Each deliberately broken allocator fails at least its target scenario (SC-003):

  | Broken allocator | Fails |
  |------------------|-------|
  | over-allocation | conservation in `single_node` |
  | under-allocation | conservation in `single_node` |
  | dropped idle | `single_node` (idle presence, conservation) |
  | negative idle | `over_requested_node` |
  | ignores unknown fields | `policy_unknown_field` |
  | accepts unknown version | `policy_unknown_version` |
  | unstable digest | `fingerprint_stable` |

## 7. TypeScript client (US6)

```bash
cd sdk/typescript/packages/client
npx vitest run test/allocator.test.ts
npx tsc --noEmit
```

**Expected**: the round-trip test passes, and an InvalidArgument error surfaces as `ConnectError`
with that code. (`npm run build` fails at the DTS step on `main` as well; use `tsc --noEmit`.)

## 8. Nothing else changed (SC-006)

```bash
make test
golangci-lint run ./...
make lint-markdown
```

**Expected**: the full existing suite and lint pass unchanged, and the existing usage-source warning
text is unchanged.
