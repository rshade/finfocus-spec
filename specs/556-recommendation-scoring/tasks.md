# Tasks: Recommendation Scoring

Write each test task first and confirm it fails before its implementation task.

## Phase 1: Contract

- [x] T001 Add `proto/finfocus/v1/scoring.proto` and `PLUGIN_CAPABILITY_RECOMMENDATION_SCORING = 18`
- [x] T002 `make generate` for Go, Connect, and TypeScript; `buf lint`; `buf breaking`

## Phase 2: Validators (US3)

- [x] T003 Table tests in `sdk/go/testing/scoring_test.go`
- [x] T004 `sdk/go/testing/scoring.go`

## Phase 3: Mock and conformance (US4)

- [x] T005 Tests in `scorer_mock_test.go` and `scorer_conformance_test.go`, including broken scorers
- [x] T006 `scorer_mock.go`, `scorer_conformance.go`, and the export in `export_test.go`

## Phase 4: SDK serving (US1, US2)

- [x] T007 `pluginsdk/scorer_serve_test.go`: transport parity, errors, health, discovery, legacy name
- [x] T008 `RecommendationScorerProvider`, `optionalServices`, adapters, health names
- [x] T009 `maxValidCapability`, `optionalCapabilities`, `legacyCapabilityNames`, bounds test
- [x] T010 Startup warning for scorer-only plugins

## Phase 5: TypeScript and docs (US5)

- [x] T011 `RecommendationScorerClient` and its test
- [x] T012 `docs/recommendation-scoring.md`, package READMEs, `docs/README.md`
- [x] T013 CLAUDE.md pattern notes and this feature folder

## Phase 6: Verification

- [x] T014 `go build ./...`, `go test -race ./...`, `make lint`, markdownlint
