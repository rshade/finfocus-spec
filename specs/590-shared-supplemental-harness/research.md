# Research: Shared Conformance Harness and Duplicate-Key Check

## R1. Shape of the duplicate-key helper

- **Decision**: `findDuplicate[T any, K comparable](list []T, key func(T) K) (int, int, bool)`
  in `sdk/go/testing/supplemental.go`, next to `pairwiseDuplicateLimit`. Callers pass a
  method expression (`(*pbc.ContractCommitment).GetContractCommitmentId`) or a named
  function (`billingPeriodIdentity`), and build their own error from `(i, first)`.
- **Rationale**: Returning indices rather than an error keeps each dataset's error text,
  sentinel, and `invalidArgumentError` wrapping exactly where it is today (FR-003, FR-008).
  A method expression or a top-level function is a static func value, so passing it does not
  allocate (FR-004). In generic code `key` is an indirect call that cannot be inlined, so the
  pairwise path computes each key once into a `[pairwiseDuplicateLimit]K` stack array and
  compares the stored keys. Calling `key(list[j])` in the inner loop measured +21% on
  `page_50` in T019; the array version measured about 4% faster than `main`, still 0 allocs.
  The results are unnamed because `nonamedreturns` is enabled.
- **Same result on both paths**: The pairwise scan returns the lowest `i` that has an earlier
  equal key, matched against the lowest such `j`. The map path stores each key's first index
  and returns at the first repeat, which is the same `i`, and the stored index is the earliest
  `j`. So FR-002 holds by construction, and a table test pins it at sizes 64 and 65.
- **Alternatives considered**:
  - A helper that takes an `errFn func(i, first int) error`. This was rejected because it is a
    second func parameter at every call site, and a closure over `list` would allocate.
  - An `eq func(a, b T) bool` for the pairwise path. This was rejected because the map path
    needs a comparable key anyway, so one `key` function serves both paths.

## R2. Shape of the shared harness

- **Decision**: an unexported generic `bufconnHarness[C any]` in a new
  `sdk/go/testing/bufconn_harness.go`. It holds the listener, server, connection, client,
  and a `newClient func(grpc.ClientConnInterface) C`. It is built by
  `newBufconnHarness(register func(*grpc.Server), newClient)`, and has the methods
  `Start(testing.TB)`, `Stop()`, and an unexported `dial()`. Each exported
  harness embeds it by value:
  `type ContractCommitmentHarness struct{ bufconnHarness[pbc.SupplementalDatasetServiceClient] }`.
- **Rationale**: Embedding promotes `Start`, `Stop`, and `Client` with identical signatures,
  so the exported types, constructors, and method sets stay source-compatible (FR-006), with
  no deprecations. The generated `pbc.New*Client` constructors already have the
  `func(grpc.ClientConnInterface) C` shape, and the generated `pbc.Register*Server` functions
  fit `register`. `TestHarness.createClientConnection` (spec validation) calls `h.dial()`,
  so the one deprecated-dial suppression lives in `dial` (FR-007).
- **Godoc (revised in T015)**: `go/doc` does list methods promoted from the unexported
  embedded field, but it prints the generic signature unsubstituted:
  `func (h *ContractCommitmentHarness) Client() C`. `Start` and `Stop` have no type
  parameter in their signatures and read correctly. So each exported harness declares its own
  three-line `Client()` that returns its concrete client type, with its original doc comment.
  This shadows nothing, because `bufconnHarness` no longer has a `Client` method; the lifecycle
  test reads `h.client`. `go doc` now shows, for example,
  `Client() pbc.SupplementalDatasetServiceClient`.
- **Alternatives considered**:
  - One exported `SupplementalHarness` with aliases for the two old names. This was rejected
    because aliasing would make the two names the same type and would need a combined
    constructor. It also adds exported surface that the issue does not require.
  - Type aliases to the generic instantiation (`type X = bufconnHarness[...]`). This was
    rejected because it exposes an unexported type through an exported name, and the two
    supplemental harnesses would become the same type.
  - Converting only the two supplemental harnesses. This was rejected because it leaves four
    copies. The issue's own proposal includes `TestHarness`.

## R3. Interaction with in-flight claims (#579, #580)

- **Decision**: Touch only the harness block (type, constructor, `Start`, `Stop`, `Client`)
  of `allocator_conformance.go` and `scorer_conformance.go`.
- **Rationale**: Those issues change the allocator and scorer request and response shapes,
  which affects scenarios and adapters, not the bufconn lifecycle. Git merges by hunk, so
  any conflict stays local to that block.

## R4. Performance check

- **Decision**: A/B `BenchmarkValidateGetContractCommitmentsResponse` (page_50 and page_1000)
  against a test binary prebuilt from `main`, with `-count=6`, and compare with `benchstat`
  if it is available.
- **Rationale**: Calling the key through a func value instead of an inlined getter could
  cost some nanoseconds on the 64-record pairwise path. The allocation-free tests guard the
  hard requirement (0 allocs). The benchmark only shows whether there is a regression worth
  reporting.
