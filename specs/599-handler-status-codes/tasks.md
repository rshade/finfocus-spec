---

description: "Tasks for keeping handler status codes in pluginsdk.Server"
---

# Tasks: Keep Handler Status Codes in the Plugin SDK Server

**Input**: `specs/599-handler-status-codes/` (spec, plan, research, contracts/server-errors.md)

**Tests**: Required and written first (Principle V). Test names are neutral (FR-008).

## Phase 1: Setup

- [X] T001 Confirm baseline on the stacked branch: `go test ./sdk/go/pluginsdk/` passes

## Phase 2: User Story 1 - A handler's deliberate status reaches the host (P1)

**Independent Test**: every wrapped RPC returns a handler's `InvalidArgument` unchanged.

- [X] T002 [US1] Create sdk/go/pluginsdk/handler_status_test.go (Apache header, `package pluginsdk`)
  with an `errorHandlerPlugin` that implements `DryRunHandler`, `SupportsProvider`,
  `RecommendationsProvider`, `BudgetsProvider`, `DismissProvider`, `BatchCostHandler`,
  `ResolveResourceTypesProvider`, and `PluginInfoProvider`, each returning a configured error, and a
  table `rpcCalls` that invokes each of the eight RPCs on a `*Server` with a valid request
- [X] T003 [US1] Add `TestHandlerStatusPassThrough`: for each RPC and for `InvalidArgument`,
  `NotFound`, `Unavailable`, a wrapped (`fmt.Errorf("%w")`) status, and a `GRPCStatus()` error type,
  the client gets the same code and message
- [X] T004 [US1] Add `TestHandlerStatusInternal`: for each RPC, a plain error, `context.Canceled`,
  and an `Unknown` status give `Internal` with the RPC's existing message, and the error text is
  absent
- [X] T005 [US1] Add `TestHandlerStatusConnect`: call all eight `ConnectHandler` methods directly
  with `connect.NewRequest`; a handler `InvalidArgument` comes back as a `*connect.Error` with
  `connect.CodeInvalidArgument` and the handler's message, and a plain error as
  `connect.CodeInternal` (FR-006, SC-001)
- [X] T006 [US1] Run the new tests; confirm T003 and T005 fail and T004 passes before T007
- [X] T007 [US1] Add `handlerStatus(err error, internalMsg string) (error, bool)` to
  sdk/go/pluginsdk/sdk.go per research R1, with a doc comment
- [X] T008 [US1] Use `handlerStatus` in all eight wrappers in sdk/go/pluginsdk/sdk.go, keeping each
  RPC's existing internal message and its error-level log for non-fallback errors (FR-005)

## Phase 3: User Story 2 - Embedding the generated stub is safe (P1)

**Independent Test**: a stub-embedding plugin and a plain plugin answer the seven optional RPCs alike.

- [X] T009 [US2] Add `TestHandlerStatusUnimplementedFallsBack` to
  sdk/go/pluginsdk/handler_status_test.go: a handler returning `Unimplemented` gets each RPC's
  not-a-provider answer from contracts/server-errors.md, including `GetPluginInfo` with and without
  `ServeConfig.PluginInfo` and `ResolveResourceTypes` with a `TypeRegistry`
- [X] T010 [US2] Add `TestStubEmbeddingMatchesPlainPlugin`: a plugin embedding
  `pbc.UnimplementedCostSourceServiceServer` and a plain plugin, same config, compared on the seven
  optional RPCs (code, and response via `proto.Equal`)
- [X] T011 [US2] Add `TestStubEmbeddingPassesBudgetsConformance`: `RunStandardConformance` on the
  stub-embedding plugin has no failing `GetBudgets` result
- [X] T012 [US2] Extract the not-a-provider branches into methods (`supportsDefault`,
  `recommendationsDefault`, `budgetsUnsupported`, `dismissUnsupported`, `resolveDefault`,
  `pluginInfoDefault`, `dryRunUnsupported`) and call them from both the `!ok` and fallback paths in
  sdk/go/pluginsdk/sdk.go; `BatchCost` falls back to `batchCostFallback`; log fallbacks at debug

## Phase 4: User Story 3 - Documentation (P2)

- [X] T013 [P] [US3] Add a "Handler Errors" section to sdk/go/pluginsdk/README.md with the three
  rules, the RPC list, that embedding the generated stub is safe, and a compiling example handler
  returning `status.Error(codes.InvalidArgument, ...)`
- [X] T014 [US3] Compile the README example in a temporary package and delete it

## Phase 5: Polish

- [X] T015 Add a "Handler Status Pattern (599-handler-status-codes)" section and Active Technologies
  / Recent Changes entries by hand in CLAUDE.md
- [X] T016 Run gates: gofmt/goimports, `make lint-go` (0 issues), `make test`,
  `go test -v -tags=integration ./sdk/go/testing/`, `make lint-markdown`, `make lint-yaml`,
  `make validate-npm`, `make generate && git diff --exit-code -- sdk/`, and a provider-name grep on
  the diff
- [X] T017 Confirm existing plain-error tests (dismiss, DryRun, Supports, ResolveResourceTypes,
  GetPluginInfo) pass unedited (SC-003)

## Dependencies

T001 → US1 tests (T002-T006) → T007-T008 → US2 tests (T009-T011) → T012 → US3 → Polish. T013 is
independent of code tasks.
