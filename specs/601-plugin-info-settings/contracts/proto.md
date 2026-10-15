# Contract: Plugin settings on plugin info

Additive change to `proto/finfocus/v1/costsource.proto`. No RPC is added. No existing field
changes number, type, or meaning.

## PluginSettingType

```protobuf
enum PluginSettingType {
  PLUGIN_SETTING_TYPE_UNSPECIFIED = 0;
  PLUGIN_SETTING_TYPE_STRING = 1;
  PLUGIN_SETTING_TYPE_INT = 2;
  PLUGIN_SETTING_TYPE_BOOL = 3;
  PLUGIN_SETTING_TYPE_ENUM = 4;
  PLUGIN_SETTING_TYPE_URL = 5;
  PLUGIN_SETTING_TYPE_DURATION = 6;
}
```

## PluginSetting

```protobuf
message PluginSetting {
  string key = 1;
  PluginSettingType type = 2;
  bool required = 3;
  bool secret = 4;
  string default_value = 5;
  string description = 6;
  string env = 7;
  repeated string enum_values = 8;
}
```

Comments on `env` state that the host injects the value into the plugin process environment
under that name. Comments on `secret` state that the flag is advisory and that hosts must not
display, log, or echo the value. Comments on `default_value` state that empty means no
default.

## GetPluginInfoResponse

```protobuf
repeated PluginSetting settings = 7;
```

The field comment states that an empty list means the plugin advertises no settings, and that
entries are not copied into `metadata`.

## Compatibility

- `buf breaking` against main reports no breaking change.
- A host that does not know field 7 ignores it.
- A plugin that leaves field 7 unset is valid.
