# Contract: Go SDK settings declaration

## Declare

```go
info := pluginsdk.NewPluginInfo("jev", "v1.0.0",
    pluginsdk.WithSettings(
        &pbc.PluginSetting{
            Key:      "api_key",
            Type:     pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
            Required: true,
            Secret:   true,
            Env:      "TYPESAFE_API_KEY",
        },
        &pbc.PluginSetting{
            Key:          "timeout",
            Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION,
            DefaultValue: "30s",
            Env:          "JEV_TIMEOUT",
        },
    ),
)
```

`WithSettings` clones each entry. Declaration order is the response order.

`PluginInfo.Validate` rejects a malformed list. `Serve` returns that error and does not
listen.

## Returned to the host

`GetPluginInfoResponse.settings` on both paths:

- Configured `PluginInfo`: the cloned list.
- `PluginInfoProvider`: the list the plugin returned, unchanged when valid.

A non-empty invalid list fails the call with `codes.Internal` and the message
`plugin settings are invalid`. The log line is the `ValidatePluginSettings` error, which
does not contain `default_value`.

An empty list is omitted in the same way an unset repeated field is omitted. Metadata is not
given setting keys or values.

## Validator

`plugintesting.ValidatePluginSettings(settings []*pbc.PluginSetting) error`

Rules are the table in [data-model.md](../data-model.md). `pluginsdk.ErrInvalidPluginSettings`
is that sentinel.

## Mock

`MockPlugin.Settings` is copied into the mock's `GetPluginInfo` response when non-nil. The
zero value leaves the list empty. The mock does not validate; the conformance check and
`pluginsdk.Serve` do.
