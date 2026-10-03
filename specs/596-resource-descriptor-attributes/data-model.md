# Data Model: Structured Attributes on ResourceDescriptor

## ResourceDescriptor (changed)

| # | Field | Type | Change |
|---|-------|------|--------|
| 1-11 | provider … lineage | (unchanged) | none |
| 5 | `tags` | `map<string,string>` | Meaning unchanged. The value limit rises to 2048. |
| 12 | `attributes` | `google.protobuf.Struct` | **new**, optional |

### `attributes` rules

| Rule | Where enforced |
|------|----------------|
| Absent or zero fields means "host sent none". Plugins fall back to `tags`. | contract (proto comment, docs) |
| `proto.Size(attributes) <= MaxAttributesBytes` (65,536) | `pluginsdk.ValidateResourceDescriptor`, `testing.ValidateResourceDescriptor` |
| Hosts omit `__`-prefixed keys, credential-like keys, and secret-marked values | contract only (host obligation) |
| Plugins do not log the field verbatim | contract only |
| Integers above 2^53 are sent as strings | contract only (guidance) |
| The whole encoded request fits the transport limit. Hosts split batches. | contract only (documented) |

## Constants

| Package | Constant | Old | New |
|---------|----------|-----|-----|
| `pluginsdk` | `MaxTagValueLength` | 256 | 2048 |
| `testing` | `MaxTagValueLength` | 256 | 2048 |
| `pluginsdk` | `MaxAttributesBytes` | — | 65536 |
| `testing` | `MaxAttributesBytes` | — | 65536 |

These stay unchanged: `MaxTagCount` (50), `MaxTagsPerResource` (256), and `MaxTagKeyLength` (128) in both
packages.

## Errors

- `pluginsdk`: `status.Errorf(codes.InvalidArgument, "attributes size %d bytes exceeds maximum %d", n, MaxAttributesBytes)`.
- `testing`: the new sentinel `ErrAttributesTooLarge` (its text names the limit), returned as
  `NewContractError("attributes", n, ErrAttributesTooLarge)`. This follows the
  `ErrTagValueTooLong` pattern.

## Attribute accessor

`pluginsdk.AttributeValue(attrs *structpb.Struct, path string) (*structpb.Value, bool)`

| Input | Result |
|-------|--------|
| `attrs == nil`, or `path == ""` | `nil, false` |
| Each segment found (struct key, or a list index in range) | `value, true` |
| Explicit JSON null at the path | `NullValue value, true` |
| Missing key | `nil, false` |
| List with a non-numeric, negative, or out-of-range index | `nil, false` |
| A segment applied to a scalar | `nil, false` |
| An empty segment (`a..b`, a leading or trailing dot) | `nil, false` |
