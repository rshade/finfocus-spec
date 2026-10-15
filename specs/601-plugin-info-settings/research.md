# Research: Plugin Settings on Plugin Info

No item in the Technical Context needed clarification. Each decision records a design choice
and the alternative it beat.

## R1: Message, enum, and field number

- **Decision**: Add `PluginSetting` and `PluginSettingType` to `costsource.proto` immediately
  before `GetPluginInfoResponse`. Add `repeated PluginSetting settings = 7` on that response.
  The enum zero value is `PLUGIN_SETTING_TYPE_UNSPECIFIED`. The other values are string,
  integer, boolean, enum, URL, and duration, in that order.
- **Rationale**: Field 6 is `capabilities`, so 7 is the next free number. The descriptor is
  used only by this response, so it sits beside it rather than in `enums.proto`. Unspecified
  is rejected, which matches every other enum in this repo.
- **Alternatives considered**: A metadata convention (`setting.api_key=...`). Rejected in the
  issue: metadata is an untyped string map, and secret defaults would land in a map hosts
  already log. `optional string default_value` so a string can default to empty. Rejected:
  the issue uses a plain string, and empty means no default for every type.

## R2: Where the rules live

- **Decision**: `plugintesting.ValidatePluginSettings` in `sdk/go/testing/plugin_settings.go`.
  `pluginsdk` delegates and re-exports `ErrInvalidPluginSettings`. `PluginInfo.Validate`
  calls it when `len(Settings) > 0`. Both `GetPluginInfo` builders call it when the response
  list is non-empty.
- **Rationale**: `pluginsdk` already imports `testing`. `testing` cannot import `pluginsdk`.
  Conformance and the SDK then share one rule, as allocation and scoring do.
- **Alternatives considered**: A private copy in each package. Rejected: the two copies drift,
  and this repo's later features stopped doing that.

## R3: What the validator accepts, and what it costs

- **Decision**: Return nil immediately when the slice is empty or nil. Otherwise reject more
  than 64 entries before inspecting them. Per entry, reject nil, then check key, type,
  environment name, description length, enum values, and default. Uniqueness of keys and
  environment names is a pairwise scan (the cap is 64). Scanners are hand-rolled. No
  `regexp`, no `net/url`, no `strconv`. `time.ParseDuration` runs only when a duration
  default is set.
- **Rationale**: The empty guard allocates nothing, so `PluginInfo.Validate` on info with no
  settings stays where it is today (FR-008). A valid list with no URL or duration default
  also allocates nothing. A duration default may allocate inside `time.ParseDuration`; that
  path is not the empty benchmark.
- **Key**: 1 to 64 bytes, `^[a-z][a-z0-9_]*$`.
- **Env**: 1 to 128 bytes, `^[A-Za-z_][A-Za-z0-9_]*$`, unique.
- **Description and default**: at most 1024 bytes. Empty default means no default and is not
  type-checked.
- **Enum**: allowed values only when the type is enum; an enum has 1 to 32 values, each 1 to
  128 bytes, unique. A default must be one of them.
- **Integer default**: optional `-`, then `0` or digits without a leading zero, in int64.
- **Boolean default**: `true` or `false`.
- **URL default**: a scheme of letters, digits, `+`, `-`, or `.`, then `://`, then a non-empty
  host with no spaces. No `net/url` parse, so a valid URL default does not allocate.
- **Duration default**: `time.ParseDuration`.
- **Alternatives considered**: `regexp.MustCompile` at init. Rejected: the scanners are
  shorter than the patterns and allocate nothing per call.

## R4: Error text and the RPC failure

- **Decision**: `ValidatePluginSettings` returns a plain error wrapping
  `ErrInvalidPluginSettings`. The text is `settings[<i>].<field>: <rule>` or
  `settings: more than 64 entries`. It may quote a key or an environment name. It never
  quotes `default_value`, a description, or an enum value. `Serve` already wraps
  `PluginInfo.Validate`, so a bad static list fails startup with that text.
  `handleProviderPluginInfo` and `handleConfiguredPluginInfo`, when the list is non-empty
  and invalid, log the validation error and return `codes.Internal` with the client message
  `plugin settings are invalid`.
- **Rationale**: The host did not send a bad request, so `InvalidArgument` would be a lie.
  Incomplete plugin info already uses Internal. The operator log has the rule; the client
  message cannot leak a secret default. Keys are not secrets.
- **Alternatives considered**: Returning the validation text to the host. Rejected: a future
  default might be echoed by a sloppy format string, and the client cannot fix a plugin's
  descriptor.

## R5: Declaration and copies

- **Decision**: `PluginInfo.Settings` plus `WithSettings(settings ...*pbc.PluginSetting)`.
  The option clones each entry with `proto.CloneOf` and drops nothing, including nils, so
  validation can report `settings[i]: nil`. The response builder copies that slice again.
  A `PluginInfoProvider` returns its own list; the server validates the clone it already
  makes and does not replace the list.
- **Rationale**: Matches `WithProviders`: the caller can mutate its slice after construction
  without racing the server. Order is declaration order.
- **Alternatives considered**: Struct tags on a config struct. The spec records that as out
  of scope.

## R6: Conformance sees the plugin, Serve protects the host

- **Decision**: Extend `createGetPluginInfoValidResponseTest` so a successful response with
  a settings list must pass `ValidatePluginSettings`. An empty list passes, including
  Unimplemented for legacy plugins. A plugin that returns a duplicate key fails this check.
  That is a testing-package test with a one-method stub on the harness, not a change to the
  default mock. `MockPlugin.Settings`, when non-nil, is copied into `GetPluginInfo` in order.
  The default stays nil, so current conformance is unchanged.
- **Rationale**: `TestHarness` registers the plugin on a bare gRPC server. It does not call
  `pluginsdk.Serve`, so the server-side refusal is not what conformance observes. The check
  has to validate the returned list itself. The refusal is covered by a `pluginsdk` serve
  test on both the static path and the provider path.
- **Alternatives considered**: Running conformance through `Serve`. Rejected: it would change
  the harness for every existing check.

## R7: Manifest configuration

- **Decision**: Change the `configuration` description in
  `schemas/plugin_manifest.schema.json` to say the object is not the host delivery channel
  and that hosts read settings from plugin info. Do not remove the object, do not add
  setting fields to the manifest, and do not read it from the SDK.
- **Rationale**: The spec's clarification. Removing an optional object is a schema break with
  no consumer.

## R8: TypeScript

- **Decision**: No new client method. `make generate` adds `settings` to
  `GetPluginInfoResponse`. A vitest in `plugin-settings.test.ts` builds one setting with
  `create`, round-trips it with `toJson` / `fromJson`, and reads key, type, secret, and env.
  The existing `getPluginInfo` integration test is left unchanged (the mock returns no
  settings).
- **Rationale**: Constitution XIII is met by the generated field on the call that already
  exists. A client-side copy of the Go validator is not required: hosts render, they do not
  re-validate a plugin's own descriptor. Go is the gate.

## R9: Docs and the release note

- **Decision**: Proto field comments state the injection contract and the secret rule. The
  plugin developer guide gains a short "Settings" subsection under plugin info. The
  pluginsdk README's plugin-info section shows `WithSettings` and says secret values do not
  belong in metadata. release-please writes the changelog from the `feat(pluginsdk):` commit.
  The pull request body tells core that the hop is "read `settings`, inject `env`".
- **Rationale**: FR-013, FR-014, and FR-015. This repo does not contain core's
  `pluginupgrade` guide.
