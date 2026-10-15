# Data Model: Plugin Settings on Plugin Info

## GetPluginInfoResponse (changed)

| # | Field | Type | Change |
| --- | --- | --- | --- |
| 1 | `name` | string | Unchanged, still required |
| 2 | `version` | string | Unchanged, still required |
| 3 | `spec_version` | string | Unchanged, still required |
| 4 | `providers` | repeated string | Unchanged |
| 5 | `metadata` | map<string,string> | Unchanged. Settings are not copied here |
| 6 | `capabilities` | repeated PluginCapability | Unchanged. No new capability |
| 7 | `settings` | repeated PluginSetting | **New**, optional |

An empty `settings` list means the plugin advertises none. Legacy plugins that return
Unimplemented are unchanged.

## PluginSetting (new)

| # | Field | Type | Rule |
| --- | --- | --- | --- |
| 1 | `key` | string | 1-64 bytes, `^[a-z][a-z0-9_]*$`, unique in the list |
| 2 | `type` | PluginSettingType | Not unspecified |
| 3 | `required` | bool | The plugin needs a value. A non-empty default satisfies it when unset |
| 4 | `secret` | bool | Advisory. Hosts must not display, log, or echo the value |
| 5 | `default_value` | string | Empty means no default. Otherwise it matches `type`. At most 1024 bytes |
| 6 | `description` | string | At most 1024 bytes. May be empty |
| 7 | `env` | string | 1-128 bytes, `^[A-Za-z_][A-Za-z0-9_]*$`, unique in the list |
| 8 | `enum_values` | repeated string | Only for enum. Enum has 1-32 unique values, each 1-128 bytes |

At most 64 entries. Order is declaration order. A nil entry is invalid.

## PluginSettingType (new)

| # | Name | Default check when `default_value` is non-empty |
| --- | --- | --- |
| 0 | UNSPECIFIED | Always invalid |
| 1 | STRING | Any string up to the length cap |
| 2 | INT | Optional `-`, then `0` or digits with no leading zero, in int64 |
| 3 | BOOL | `true` or `false` |
| 4 | ENUM | Equal to one `enum_values` entry |
| 5 | URL | Scheme, then `://`, then a non-empty host, no spaces |
| 6 | DURATION | `time.ParseDuration` syntax (`30s`, `1h30m`) |

## Relationships

- A host reads `settings` from plugin info and writes each value to the process environment
  under `env`. This repository does not do that write.
- Static plugin info (`PluginInfo.Settings`, set with `WithSettings`) is copied onto the
  response by the configured plugin-info path.
- A plugin that implements `PluginInfoProvider` supplies the list itself. The server checks
  it and does not replace it.
- The manifest `configuration` object is not a source for this list.

## Validation

| Call site | Empty list | Non-empty list |
| --- | --- | --- |
| `PluginInfo.Validate` (Serve startup) | skipped, no alloc | `ValidatePluginSettings`; failure fails startup |
| `handleConfiguredPluginInfo` | skipped | same check; failure is Internal, list not returned |
| `handleProviderPluginInfo` | skipped | same check on the cloned response |
| Conformance plugin-info check | pass | fail the check when the returned list is invalid |

`ValidatePluginSettings` errors wrap `ErrInvalidPluginSettings`. The text names the index and
the rule and does not include `default_value`. The RPC client message is
`plugin settings are invalid`.
