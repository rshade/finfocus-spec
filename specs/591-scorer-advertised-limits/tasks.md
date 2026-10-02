# Tasks: Scorer Advertised Limits

- [x] T001 [FR-001, FR-003, FR-004] Tests: `sdk/go/testing/scorer_limits_test.go` (format/parse round trip, malformed values,
  `IsBatchTooLarge` true only for the oversize error, still `InvalidArgument`).
- [x] T002 [FR-001, FR-003, FR-004] Implement `sdk/go/testing/scorer_limits.go` and the `scoring.go` oversize error.
- [x] T003 [FR-002, FR-003, FR-004] Tests then code: `pluginsdk` `WithScorerLimits`, `ParseScorerLimits`, `IsBatchTooLarge`,
  `Validate` rejection, and `GetPluginInfo` over gRPC returning the keys.
- [x] T004 [FR-005] Tests then code: Connect detail preservation in `toConnectError`.
- [x] T005 [FR-006] Mock `AdvertisedScorerMetadata`; conformance `advertised_limits` and tightened
  `oversize_batch`; broken-scorer tests for each failure.
- [x] T006 [FR-007, FR-008] `scoring.proto` comments; `make generate`; verify `sdk/` diff is comments only.
- [x] T007 [FR-007] TypeScript `parseScorerLimits` / `isBatchTooLarge` with vitest tests; build.
- [x] T008 [FR-007] Docs: `docs/recommendation-scoring.md`, `sdk/go/testing/README.md`, root CLAUDE.md pattern note.
- [x] T009 [FR-008] Gates: buf-lint, buf breaking, make test, lint-go, lint-markdown, TS build/test.
- [x] T010 [Constitution VIII] Benchmarks for `ParseScorerLimits` and `IsBatchTooLarge` in
  `sdk/go/testing/scorer_limits_test.go`.
