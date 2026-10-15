# Tasks: Plugin Settings on Plugin Info

**Input**: Design documents from `specs/601-plugin-info-settings/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Required. Constitution Principle V (test-first) applies. Tests that name generated
types cannot compile before Phase 2. After `make generate`, the new tests fail because
`WithSettings`, `MockPlugin.Settings`, and `ValidatePluginSettings` do not exist yet. Write
those tests before the code they cover. Do not mark `checklists/security.md`; those items
stay reviewer-owned.

**Organization**: Tasks are grouped by user story so each story can be checked on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: The user story the task belongs to (US1-US5)

## Phase 1: Setup

- [ ] T001 Confirm the worktree baseline on `601-plugin-info-settings`: `make generate && git diff
  --exit-code -- sdk/` is clean, and `go test ./sdk/go/pluginsdk/ ./sdk/go/testing/` passes before
  any change. Re-check `gh issue list --label processing:roadmap` and stop if another issue holds
  the label besides #635

## Phase 2: Foundational (proto first, blocks every story)

- [x] T002 Add `PluginSettingType` and `PluginSetting` to proto/finfocus/v1/costsource.proto
  immediately before `GetPluginInfoResponse`, and `repeated PluginSetting settings = 7` on that
  response. Enum values, in order: `PLUGIN_SETTING_TYPE_UNSPECIFIED = 0`, `STRING = 1`, `INT = 2`,
  `BOOL = 3`, `ENUM = 4`, `URL = 5`, `DURATION = 6`. Message fields: `key = 1`, `type = 2`,
  `required = 3`, `secret = 4`, `default_value = 5`, `description = 6`, `env = 7`,
  `repeated string enum_values = 8`. Comments state: `env` is the delivery contract (the host
  injects the value into the plugin process environment under that name); `secret` is advisory
  and hosts must not display, log, or echo the value; empty `default_value` means no default;
  an empty `settings` list means the plugin advertises no settings and entries are not copied
  into `metadata`. Do not add a `PluginCapability`. Do not edit `enums.proto`
- [x] T003 Run `make generate` to regenerate sdk/go/proto/finfocus/v1/ and
  sdk/typescript/packages/client/src/generated/. Restore any unrelated reformatted `*.connect.go`
  files with `git checkout`
- [x] T004 Run `make buf-lint` and `buf breaking` against main. In this worktree
  `buf breaking --against '.git#branch=main'` can fail to read the git object; archive
  `origin/main` into a scratch directory and run `buf breaking --against` that directory. Both
  must report nothing

**Checkpoint**: Field 7 exists in Go (`GetSettings()`) and TypeScript (`settings`) bindings.

## Phase 3: User Story 1 - A host learns a plugin's settings (Priority: P1) MVP

**Goal**: A configured plugin returns its settings in declaration order, and the mock can carry
the same list.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'TestPluginSettings_StaticOrder'` and
`go test ./sdk/go/testing/ -run TestMockPluginSettings`.

### Tests (write first, confirm they fail)

- [x] T005 [P] [US1] Create sdk/go/pluginsdk/plugin_settings_test.go (`package pluginsdk`, Apache
  2.0 header) with `TestPluginSettings_StaticOrder`. Build info with `NewPluginInfo` and
  `WithSettings` for `api_key` (`PLUGIN_SETTING_TYPE_STRING`, required, secret, env
  `TYPESAFE_API_KEY`) then `timeout` (`PLUGIN_SETTING_TYPE_DURATION`, default `30s`, env
  `JEV_TIMEOUT`). Serve it through `NewServer` and `GetPluginInfo`. Assert both entries, in
  that order, with those attributes. Assert `metadata` does not contain `api_key`,
  `TYPESAFE_API_KEY`, or `30s`. Mutate the caller's slice after `NewPluginInfo` and assert the
  returned list is unchanged
- [x] T006 [P] [US1] Create sdk/go/testing/mock_plugin_settings_test.go (`package testing_test`,
  Apache 2.0 header) with `TestMockPluginSettings` over a `TestHarness` (no `t.Parallel()` on
  subtests that share a harness). When `MockPlugin.Settings` holds the same two entries, every
  `GetPluginInfo` response copies them in order. When `Settings` is nil, `GetSettings()` is
  empty

### Implementation

- [x] T007 [US1] In sdk/go/pluginsdk/plugin_info.go add `PluginInfo.Settings []*pbc.PluginSetting`
  and `WithSettings(settings ...*pbc.PluginSetting)`. Clone each entry with `proto.CloneOf`,
  keep nils, and preserve order
- [x] T008 [US1] In sdk/go/pluginsdk/sdk.go `handleConfiguredPluginInfo`, copy `Settings` onto
  `GetPluginInfoResponse.Settings` when the slice is non-empty. Do not write setting keys or
  values into `metadata`
- [x] T009 [US1] In sdk/go/testing/mock_plugin.go add `MockPlugin.Settings`. `GetPluginInfo`
  copies that slice onto the response when non-nil, in order, and does not validate. The zero
  value leaves the list empty
- [x] T010 [US1] Run T005 and T006 and confirm they pass

**Checkpoint**: The host can read a declared list. The mock returns one when configured.

## Phase 4: User Story 2 - Plugins and hosts that ignore settings keep working (Priority: P1)

**Goal**: No settings stays an empty list. Existing plugin-info behavior is unchanged.

**Independent Test**: `go test ./sdk/go/pluginsdk/ -run 'TestPluginSettings_Empty|TestGetPluginInfo|TestPluginInfoValidate'`
and `go test ./sdk/go/testing/ -run 'TestConformance|TestMockPluginSettings'`.

### Tests (write first)

- [x] T011 [P] [US2] In sdk/go/pluginsdk/plugin_settings_test.go add `TestPluginSettings_EmptyList`:
  `NewPluginInfo` without `WithSettings` returns an empty settings list from `GetPluginInfo`,
  and name, version, spec version, providers, and capabilities match a plugin that has no
  settings field. A `PluginInfoProvider` that leaves `settings` unset also returns an empty list
- [x] T012 [P] [US2] In sdk/go/testing/mock_plugin_settings_test.go assert the default mock (nil
  `Settings`) still satisfies the existing plugin-info shape: name, version, and spec version
  are set and settings length is 0

### Implementation

- [x] T013 [US2] Keep the empty and nil slices on the configured path and the provider path as
  an unset repeated field. Do not allocate a settings slice when none was declared. Do not add
  a capability or a legacy metadata key for settings
- [x] T014 [US2] Run T011, T012, `go test ./sdk/go/pluginsdk/ -run 'TestGetPluginInfo|TestPluginInfoValidate'`,
  and `go test ./sdk/go/testing/ -run TestConformance`. Existing plugin-info checks pass

**Checkpoint**: A plugin that advertises nothing is indistinguishable from today's plugin info,
apart from the new empty field on the wire.

## Phase 5: User Story 3 - A bad descriptor never reaches the host (Priority: P2)

**Goal**: One validator rejects every broken rule. Startup refuses a bad static list. Both
plugin-info paths refuse a bad returned list with `codes.Internal` and the client message
`plugin settings are invalid`.

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestValidatePluginSettings|BenchmarkValidatePluginSettings_Empty'`
and `go test ./sdk/go/pluginsdk/ -run 'TestPluginSettings_Reject'`.

### Tests (write first, confirm they fail)

- [x] T015 [P] [US3] Create sdk/go/testing/plugin_settings_test.go (`package testing_test`, Apache
  2.0 header) with `TestValidatePluginSettings`. `nil` and an empty slice return nil.
  `errors.Is(err, ErrInvalidPluginSettings)` for every rejection. Error text is exactly:
  `settings[0]: nil`; `settings: more than 64 entries`; `settings[1].key: duplicate "api_key"`;
  `settings[1].env: duplicate "TYPESAFE_API_KEY"`; `settings[0].key: must match ^[a-z][a-z0-9_]*$`
  (empty, `Api`, `1abc`); `settings[0].key: longer than 64 bytes`; `settings[0].type: unspecified`;
  `settings[0].env: must match ^[A-Za-z_][A-Za-z0-9_]*$` (empty, `1ENV`, a hyphen);
  `settings[0].env: longer than 128 bytes`; `settings[0].enum_values: only allowed for enum`;
  `settings[0].enum_values: enum requires 1 to 32 values`; `settings[0].enum_values: duplicate`;
  `settings[0].enum_values: longer than 128 bytes`; `settings[0].enum_values: more than 32 values`;
  `settings[0].default_value: not an integer` (`+1`, `-0`, `01`, `9223372036854775808`);
  `settings[0].default_value: not a boolean` (`True`, `1`);
  `settings[0].default_value: not an absolute URL` (`example.com`, `https://`, a value with a
  space, `/path`); `settings[0].default_value: not a duration` (`30`, `1 day`);
  `settings[0].default_value: not one of the enum values`;
  `settings[0].description: longer than 1024 bytes`;
  `settings[0].default_value: longer than 1024 bytes`. Plant default `s3cret-value`, description
  `do-not-echo`, and enum value `hidden-choice` on the failing cases and assert none of those
  strings appear in the error. Accept: required string with an empty default; required plus a
  matching default; integer `0`, `-1`, and `9223372036854775807`; bool `true` and `false`;
  duration `30s`, `0s`, and `1h30m`; URL `https://example.com`. Bounds from data-model.md: at
  most 64 settings; key 1-64 bytes `^[a-z][a-z0-9_]*$`; env 1-128 bytes
  `^[A-Za-z_][A-Za-z0-9_]*$`; description and default at most 1024 bytes; enum values only for
  enum; an enum has 1-32 unique values, each 1-128 bytes
- [x] T016 [P] [US3] In sdk/go/pluginsdk/plugin_settings_test.go add `TestPluginSettings_RejectStatic`:
  `PluginInfo.Validate` on a duplicate key wraps `ErrInvalidPluginSettings`, and `Serve` with
  that info returns before listening (same shape as `TestServe_InvalidPluginInfoReturnsError`
  in sdk/go/pluginsdk/sdk_test.go). Add `TestPluginSettings_RejectProvider`: a provider whose
  `GetPluginInfo` returns two entries with key `api_key` yields `codes.Internal`, message
  exactly `plugin settings are invalid`, and a nil response. The message does not contain a
  planted default. A provider that returns one valid entry gets that entry back unchanged.
  Connect needs no new method: `ConnectHandler.GetPluginInfo` already delegates to the server
- [x] T017 [P] [US3] In sdk/go/testing/plugin_settings_test.go add
  `BenchmarkValidatePluginSettings_Empty` and an alloc test (`testing.AllocsPerRun`) that
  reports 0 allocs/op for nil, for an empty slice, and for a valid list whose defaults are
  string, int, bool, enum, and URL (no duration). A duration default may allocate inside
  `time.ParseDuration` and is not part of this benchmark

### Implementation

- [x] T018 [US3] Create sdk/go/testing/plugin_settings.go (Apache 2.0 header) with
  `ErrInvalidPluginSettings` and `ValidatePluginSettings(settings []*pbc.PluginSetting) error`.
  Return nil immediately when `len(settings) == 0`. Reject a count above 64 before scanning.
  Then reject nil, key, type, env, description length, enum values, and a non-empty default.
  Uniqueness of keys and env names is pairwise (cap 64). Scanners are hand-rolled: no `regexp`,
  no `net/url`, no `strconv`. Integer: optional `-`, then `0` or digits with no leading zero,
  in int64; reject `+` and `-0`. Bool: `true` or `false`. URL: scheme
  `[A-Za-z][A-Za-z0-9+.-]*`, then `://`, then a non-empty host, no spaces. Duration:
  `time.ParseDuration` only when a duration default is set. Empty `default_value` means no
  default for every type, including string, and is not type-checked. The SDK does not require
  a default on a required setting. Errors wrap `ErrInvalidPluginSettings` with the exact texts
  in T015. Never quote `default_value`, description, or an enum value
- [x] T019 [US3] Create sdk/go/pluginsdk/plugin_settings.go that aliases
  `ErrInvalidPluginSettings` and calls `plugintesting.ValidatePluginSettings`. In
  `PluginInfo.Validate`, call it only when `len(Settings) > 0`. `Serve` already wraps that
  error and must not listen
- [x] T020 [US3] In sdk/go/pluginsdk/sdk.go, when `handleProviderPluginInfo` or
  `handleConfiguredPluginInfo` has a non-empty list and `ValidatePluginSettings` fails, log the
  validation error and return `codes.Internal` with client message exactly
  `plugin settings are invalid`. Do not return the list. The provider path validates the clone
  it already makes and does not replace a valid list. Do not use `InvalidArgument`
- [x] T021 [US3] Run T015-T017. `BenchmarkValidatePluginSettings_Empty` reports 0 allocs/op

**Checkpoint**: A malformed list fails startup or the plugin-info call, and the host never
receives it.

## Phase 6: User Story 4 - Conformance accepts a well-formed list and an empty one (Priority: P2)

**Goal**: The existing plugin-info check validates whatever list the plugin returned. The
harness does not go through `pluginsdk.Serve`.

**Independent Test**: `go test ./sdk/go/testing/ -run 'TestPluginSettingsConformance|TestConformance'`.

### Tests (write first)

- [x] T022 [P] [US4] In sdk/go/testing/conformance_test.go `createGetPluginInfoValidResponseTest`,
  after the required-field checks, call `plugintesting.ValidatePluginSettings(resp.GetSettings())`
  and fail the check when it returns an error. An empty list passes. The `codes.Unimplemented`
  branch stays a success and does not validate a list
- [x] T023 [P] [US4] Create sdk/go/testing/plugin_settings_conformance_test.go (`package
  testing_test`) that calls `createGetPluginInfoValidResponseTest()` on a harness. The default
  mock passes. A mock with one valid setting passes. A stub whose `GetPluginInfo` returns two
  entries with key `api_key` fails the check. Do not use `t.Parallel()` on subtests that share
  a harness

### Implementation

- [x] T024 [US4] The check in T022 is the implementation. Run T023 and
  `go test -v -run TestConformance ./sdk/go/testing/`. The default mock still passes

**Checkpoint**: Conformance gates the list the plugin returned. `Serve` remains the path that
refuses a bad list to a host (US3).

## Phase 7: User Story 5 - Every host SDK sees the same list (Priority: P3)

**Goal**: TypeScript reads the generated field through the existing `getPluginInfo` call. Docs
and the manifest state the delivery contract.

**Independent Test**: `cd sdk/typescript/packages/client && npx vitest run test/plugin-settings.test.ts`,
plus the guide and schema description.

- [x] T025 [P] [US5] Create sdk/typescript/packages/client/test/plugin-settings.test.ts. Build one
  setting with `create` (`key` `api_key`, type string, `secret` true, `env` `TYPESAFE_API_KEY`),
  round-trip with `toJson` / `fromJson`, and read key, type, secret, and env. Do not add a client
  method. Leave the existing integration `getPluginInfo` test unchanged
- [x] T026 [US5] Run `cd sdk/typescript && npm ci && npm run build && npx vitest run
  test/plugin-settings.test.ts` from packages/client. A pre-existing middleware `TS7006` is out
  of scope
- [x] T027 [P] [US5] In PLUGIN_DEVELOPER_GUIDE.md, under `GetPluginInfo`, add a short Settings
  subsection: declare with `WithSettings` or return the list from `PluginInfoProvider`; the
  host injects each value under `env`; an empty default means no default; secret values do not
  belong in the legacy metadata map
- [x] T028 [P] [US5] In sdk/go/pluginsdk/README.md section `## Plugin Info (GetPluginInfo RPC)`,
  show the `WithSettings` example from contracts/go-sdk.md and state that secret values do not
  belong in metadata
- [x] T029 [P] [US5] In schemas/plugin_manifest.schema.json, add a `description` on the
  `configuration` object: it is not the host delivery channel; hosts read settings from plugin
  info and inject each value under the advertised environment variable; the SDK does not read
  this object. Do not remove the object or add setting fields

**Checkpoint**: A TypeScript host sees the same descriptor. The manifest decision is written
down.

## Phase 8: Polish

- [x] T030 [P] Add a "Plugin Settings Pattern (601-plugin-info-settings)" note to CLAUDE.md and
  the matching Recent Changes bullet in AGENTS.md, by hand. Do not run
  `.specify/scripts/bash/update-agent-context.sh`; it strips the leading dash and space from wrapped
  bullets. Record that
  fact under Common Issues. Do not edit CHANGELOG.md
- [ ] T031 Run quickstart.md: `make generate && git diff --exit-code -- sdk/` (generation changes
  nothing beyond T003), `make buf-lint`, `buf breaking`, `make test`,
  `go test -v -tags=integration ./sdk/go/testing/`, `go test -v -run TestConformance ./sdk/go/testing/`,
  `make lint-go`, `make lint-markdown`, `make lint-yaml`, `make validate-npm`. The commit subject
  for this change is `feat(pluginsdk): advertise plugin settings in GetPluginInfo`. The pull
  request body tells core to read `settings` and inject `env` (FR-015).

The constrained pass covered buf lint, buf breaking, markdown lint, YAML lint, the manifest
schema compile, golangci-lint on `sdk/go/testing` and `sdk/go/pluginsdk`, and `go test -p 1`
of those two packages. That package run includes TestConformance. No file under
`sdk/go/testing` has an integration build tag. The client package build and
`test/plugin-settings.test.ts` passed. Full `make test`, full `make lint-go`, and
`make validate-npm` were not run.

## Dependencies and Execution Order

- Phase 1, then Phase 2 (proto and generate), then every story.
- US1 before US3: US3's serve tests edit the same plugin_settings_test.go that US1 creates.
- US2 can follow US1. Its empty-list assertions depend on the copy in T008.
- US4 depends on the validator from US3 (`ValidatePluginSettings`).
- US5's TypeScript test depends only on Phase 2. The docs tasks can run after the API names in
  US1 and US3 are stable.
- Phase 8 after all stories.

## Parallel Opportunities

- T005 and T006 (different files) after Phase 2.
- T015, T016, and T017 once US1's test file exists; T015 and T017 share
  plugin_settings_test.go in testing, so write them together.
- T025, T027, T028, and T029 touch different files.
- T022 and T023 are the same package; write T023 after T022 so the check exists.

## Implementation Strategy

MVP is Phase 2 plus US1: the field exists and a configured plugin returns its list. US2 shows
that an empty list changes nothing. US3 is the gate that keeps a bad list off the wire. US4
extends conformance. US5 covers TypeScript, the guide, and the manifest. Phase 8 is docs
context and the CI gates.
