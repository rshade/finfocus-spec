# Feature Specification: Plugin Settings on Plugin Info

**Feature Branch**: `601-plugin-info-settings`

**Created**: 2026-10-15

**Status**: Draft

**Input**: User description: "feat(pluginsdk): plugins advertise their settings in GetPluginInfo.
A plugin describes each setting it reads (stable key, type, required, secret, default,
description, and the environment variable it reads) so a host can render a typed settings
form and inject values at launch without hard-coding that plugin's setting names. Closes #635"

## Clarifications

`/speckit-clarify` was not required. The issue fixes the descriptor fields, the environment
variable as the delivery contract, the validation rules, and what is out of scope. Three
choices the issue left open are decided here:

- The plugin-info settings list is the only host delivery channel. The manifest
  `configuration` object stays in the schema and is documented as not that channel. Nothing
  ships it, and core does not parse it, so removing it is a schema break with no consumer.
  The two are not kept in sync.
- An empty `default_value` means the setting has no default. A string setting cannot
  advertise an empty-string default, because proto3 cannot tell that apart from "unset"
  without a presence flag this slice does not add.
- Environment variable names are unique across the list. Two settings that share a name
  cannot both be injected.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A host learns a plugin's settings from the plugin (Priority: P1)

A host (a CLI or a web settings page) asks a plugin for its info. The plugin returns one
entry per setting it reads: a stable key, a type, whether a value is required, whether the
value is a secret, an optional default, a description, and the environment variable the
plugin reads. The host renders a form from that list and, when it launches the plugin, puts
each value in the process environment under the advertised name. The host does not hard-code
any plugin's setting names.

**Why this priority**: This is the gap the issue reports. Today each plugin reads its own
environment variables, and users only learn the names from READMEs.

**Independent Test**: Configure the reference plugin with two settings, one of them secret,
and call plugin info. The response lists both, in the declared order, with the environment
names the host would inject.

**Acceptance Scenarios**:

1. **Given** a plugin that declares a string setting `api_key` (required, secret, environment
   `TYPESAFE_API_KEY`) and a duration setting `timeout` (default `30s`, environment
   `JEV_TIMEOUT`), **When** the host calls plugin info, **Then** both entries are returned
   with those attributes, in declaration order.
2. **Given** that response, **When** a host follows the delivery contract, **Then** it injects
   each value under the entry's environment name and does not need a plugin-specific name
   table.
3. **Given** a secret entry, **When** the host displays or logs the form, **Then** it does not
   show, log, or echo the value. The secret flag is advisory: hosts still apply their own
   credential rules.

---

### User Story 2 - Plugins and hosts that ignore settings keep working (Priority: P1)

A plugin built before this change returns no settings. A host built before this change does
not read the new list. A plugin that implements its own plugin-info method and returns no
settings is unchanged.

**Why this priority**: The addition is wire-compatible. Every current plugin and host is in
this state.

**Independent Test**: Call plugin info on the reference plugin with no settings declared, and
on a plugin that returns info with no settings list. Both succeed with an empty list, and
existing plugin-info tests pass unchanged.

**Acceptance Scenarios**:

1. **Given** a plugin that declares no settings, **When** the host calls plugin info,
   **Then** the settings list is empty and the rest of the response is unchanged.
2. **Given** a host that does not know the new field, **When** it calls a plugin that returns
   settings, **Then** the call succeeds and the host ignores the unknown field.
3. **Given** a plugin that builds its own plugin-info response and leaves settings unset,
   **When** the host calls it, **Then** the response is unchanged apart from the empty list.

---

### User Story 3 - A bad descriptor never reaches the host (Priority: P2)

A plugin author declares a settings list that breaks the rules: a duplicate or empty key, a
missing or illegal environment name, two entries sharing an environment name, enum values on
a non-enum setting, an enum with no values, a default that does not match the type, or an
unspecified type. The host never receives that list.

**Why this priority**: A host renders a form from the list. A malformed entry produces a form
the plugin cannot read, or a secret value copied into a place the host logs.

**Independent Test**: Validate one descriptor per broken rule. Each fails, the error names
the entry and the rule, and the error text does not contain the default value.

**Acceptance Scenarios**:

1. **Given** a declared list with a duplicate key, **When** it is validated, **Then**
   validation fails and names that key.
2. **Given** an enum setting with no allowed values, or a string setting that carries allowed
   values, **When** it is validated, **Then** validation fails.
3. **Given** a secret setting whose default does not match its type, **When** validation
   fails, **Then** the error names the entry and the rule and does not include the default.
4. **Given** a plugin that builds its own plugin-info response with a malformed list,
   **When** the host calls plugin info, **Then** the call fails and the malformed list is not
   returned.

---

### User Story 4 - Conformance accepts a well-formed list and an empty one (Priority: P2)

The conformance suite checks a plugin that advertises settings. When the list is present,
every entry is well formed. When the list is absent, the plugin still passes.

**Why this priority**: The issue requires the suite to check a declared descriptor, and an
absent list is the normal case for current plugins.

**Independent Test**: Run the plugin-info conformance check against the reference plugin
with a valid list and with no list. Both pass. A plugin that returns a duplicate key fails
the check.

**Acceptance Scenarios**:

1. **Given** a plugin whose plugin info includes a well-formed settings list, **When** the
   conformance suite runs, **Then** that check passes.
2. **Given** a plugin that returns no settings, **When** the same check runs, **Then** it
   passes.
3. **Given** a plugin that returns two entries with the same key, **When** the check runs,
   **Then** it fails.

---

### User Story 5 - Every host SDK sees the same list (Priority: P3)

A TypeScript host calls the existing plugin-info method. The response carries the same
settings a Go host receives. Documentation tells plugin authors how to declare settings, how
the host injects each environment variable, and how to treat secrets. The manifest
`configuration` section is described as not the delivery channel.

**Why this priority**: The constitution requires the TypeScript SDK to stay in step with the
protocol, and the issue requires the contract and the manifest decision to be written down.

**Independent Test**: Decode a plugin-info response that contains one setting in the
TypeScript client and read its key, type, and environment name. The plugin developer guide
and the schema description state the delivery contract and the manifest decision.

**Acceptance Scenarios**:

1. **Given** a plugin-info response with one setting, **When** a TypeScript host reads it
   through the existing plugin-info call, **Then** the key, type, secret flag, and
   environment name are present.
2. **Given** the plugin developer guide, **When** an author looks up settings, **Then** it
   explains declaration, environment injection, and that secret values do not belong in the
   legacy metadata map.
3. **Given** the manifest schema, **When** an author reads `configuration`, **Then** the
   description says it is not the host delivery channel.

---

### Edge Cases

- **No settings declared**: The list is empty. This is never an error. Legacy plugins that
  do not implement plugin info still return the unimplemented status they return today.
- **Empty default**: The setting has no default. The type check runs only when the default
  is non-empty. A string setting cannot advertise `""` as its default.
- **Required and default both set**: Required means the plugin does not operate correctly
  without a value. A non-empty default is what the plugin uses when the variable is unset,
  so it satisfies required when the host injects nothing. The SDK does not require a default
  on a required setting.
- **Secret default**: Allowed. Hosts must not display it. The SDK does not copy any setting
  into the legacy metadata map, and validation errors do not quote `default_value`.
- **Unspecified type**: Rejected. The zero enum value is not a setting type.
- **Enum default**: When set, it is one of the allowed values. Allowed values are unique and
  non-empty, and they appear only on an enum setting. An enum setting has at least one.
- **Integer default**: Optional leading minus, then digits, in the signed 64-bit range. `0`
  is a real default. Leading zeros are rejected.
- **Boolean default**: The lowercase words `true` and `false` only.
- **URL default**: An absolute URI with a non-empty scheme and host.
- **Duration default**: A Go duration: a signed sequence of decimal numbers, each with a unit
  suffix `ns`, `us`, `µs`, `ms`, `s`, `m`, or `h`. `0s` is a real default.
- **Shared environment name**: Rejected, even when the keys differ.
- **Bounds**: At most 64 settings. Each key is 1 to 64 bytes matching
  `^[a-z][a-z0-9_]*$`. Each environment name is 1 to 128 bytes matching
  `^[A-Za-z_][A-Za-z0-9_]*$`. Description and default are at most 1024 bytes. An enum has 1
  to 32 values, each 1 to 128 bytes.
- **Declaration order**: The response preserves it.
- **Own plugin-info method**: The plugin returns the list itself. The SDK validates it with
  the same rules before the host sees it, and does not replace it. A malformed list fails
  that call with an internal error, the same class as incomplete plugin info, because the
  host did not send a bad request.
- **Startup**: A list declared with the plugin's static info is validated before the plugin
  serves. A malformed list fails startup.
- **Metadata**: Settings are not written into the legacy metadata map. Authors must not put
  secret values there. The SDK does not scan metadata for secrets.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Plugin info MUST carry an optional list of setting descriptors as a new field
  numbered 7. No existing plugin-info field changes number, type, or meaning.
- **FR-002**: Each descriptor MUST carry a stable key (field 1), a type (field 2), a required
  flag (field 3), a secret flag (field 4), a default (field 5), a description (field 6), an
  environment variable name (field 7), and allowed values (field 8).
- **FR-003**: The type MUST be one of string, integer, boolean, enum, URL, or duration. The
  zero value MUST mean unspecified and MUST be rejected.
- **FR-004**: Field comments MUST state that the environment name is the delivery contract:
  the host injects the value into the plugin process environment under that name. They MUST
  state that secret is advisory and that hosts must not display, log, or echo the value.
- **FR-005**: An empty list MUST be valid and MUST mean the plugin advertises no settings.
- **FR-006**: The SDK MUST validate a list declared with static plugin info before the plugin
  serves, and MUST validate a list returned by a plugin's own plugin-info method before
  sending it. A malformed list MUST NOT be returned to the host.
- **FR-007**: Validation MUST reject a list that breaks any rule in Edge Cases: key shape and
  uniqueness, environment name shape and uniqueness, type, enum values, default shape,
  bounds, and unspecified type. The error MUST name the entry index and the rule, and MUST
  NOT include the default value.
- **FR-008**: A plugin-info response with no settings MUST stay as cheap to validate and
  build as it is today. The empty-list check MUST allocate nothing.
- **FR-009**: The SDK MUST NOT copy settings, including defaults, into the legacy metadata
  map. Documentation MUST say that secret values do not belong in that map.
- **FR-010**: The reference plugin MUST return a configured settings list in declaration
  order, and MUST return an empty list when none is configured.
- **FR-011**: The conformance suite MUST accept a plugin whose plugin info has a well-formed
  settings list and a plugin whose list is empty, and MUST reject a plugin that returns a
  duplicate key.
- **FR-012**: The generated Go and TypeScript plugin-info types MUST expose the settings
  list. The existing TypeScript plugin-info call MUST return it without a new method.
- **FR-013**: The plugin developer guide and the Go SDK README MUST explain how an author
  declares settings, the environment injection contract, secret handling, and that an empty
  default means no default.
- **FR-014**: The manifest schema's `configuration` description MUST state that it is not
  the host delivery channel. The property MUST remain. This slice MUST NOT read settings
  from a manifest.
- **FR-015**: The release note for this change MUST name the settings list and the
  environment injection contract, so core can write its upgrade hop. This repository MUST
  NOT hand-edit the generated changelog.

### Key Entities

- **Plugin info**: The response a host reads once to learn a plugin's name, version,
  providers, and capabilities. Gains an optional settings list.
- **Setting descriptor**: One configurable input. Key, type, required, secret, optional
  default, description, environment variable, and, for an enum, the allowed values.
- **Environment variable**: The name the host writes and the plugin reads. It is the only
  delivery channel in this slice.
- **Manifest configuration**: An existing unused schema object. It stays, and it is not the
  delivery channel.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A host can render a form for a plugin that declares settings using only the
  plugin-info response: 100% of the declared entries are present, in order, with key, type,
  required, secret, default, description, and environment name.
- **SC-002**: 100% of existing plugin-info tests pass unchanged, which shows that plugins
  and hosts that do not use the list are unaffected.
- **SC-003**: Every malformed rule in Edge Cases is rejected by a test, and none of those
  failures include the default value in the error text.
- **SC-004**: The conformance suite passes a plugin with a well-formed list and a plugin
  with an empty list, and fails a plugin that repeats a key.
- **SC-005**: The protocol compatibility check against the current main branch reports no
  breaking change.
- **SC-006**: A reader of the plugin developer guide can name the delivery channel and the
  manifest decision without opening the schema or the protocol file.

## Assumptions

- The host renders the form and injects environment variables. That work is finfocus core
  ([rshade/finfocus#1765](https://github.com/rshade/finfocus/issues/1765)), not this change.
- Declaration is an explicit list on static plugin info, or the list a plugin returns from
  its own plugin-info method. Deriving settings from struct tags is out of scope.
- No new capability flag is needed. Any plugin may advertise settings on the existing
  plugin-info call, and a plugin that advertises none stays compliant.
- Empty `default_value` means no default, for every type, including string.
- Integer range is signed 64-bit. Duration syntax is Go's duration syntax, because that is
  what the plugins named in the issue already parse.
- URL defaults require a scheme and a host. A path-only value is rejected.
- At most 64 settings. The bounds in Edge Cases are the contract, not an implementation
  suggestion.
- Per-request credentials (`specs/056-per-request-credentials`) are out of scope. Settings
  are process environment at launch, not call metadata.
- The SDK does not read current environment values back into the descriptor.
- A plugin that fails static validation does not serve. A plugin whose own method returns a
  bad list fails that call with an internal error and a client message that does not quote
  the default.
- release-please owns the changelog. The conventional commit and the pull request body carry
  the hop note core needs. Core writes the upgrade guide.
