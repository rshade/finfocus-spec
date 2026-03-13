# Data Model: Test Helper for ResourceDescriptor Creation

## Entities

### newTestDescriptor (unexported test helper)

This is not a data model entity but an unexported function in `pluginsdk_test` package.
No new types, structs, or state are introduced.

**Function Signature**:

```go
func newTestDescriptor(id, arn string, opts ...pluginsdk.ResourceDescriptorOption) *pbc.ResourceDescriptor
```

**Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | `string` | Resource ID; only applied when non-empty |
| `arn` | `string` | Resource ARN; only applied when non-empty |
| `opts` | `...pluginsdk.ResourceDescriptorOption` | Additional options appended after defaults |

**Default Values Applied**:

| Field | Default Value | Source |
|-------|--------------|--------|
| `Provider` | `"aws"` | Most common test provider |
| `ResourceType` | `"ec2"` | Most common test resource type |
| `SKU` | `"t3.micro"` | Standard test instance type |
| `Region` | `"us-east-1"` | Standard test region |

**Option Ordering**: Base defaults are applied first, then `id`/`arn` (if non-empty), then
caller-provided `opts`. This ensures callers can override any default.

## Relationships

- **Uses**: `pluginsdk.NewResourceDescriptor()` — delegates to the production constructor
- **Uses**: `pluginsdk.WithID()`, `pluginsdk.WithARN()`, `pluginsdk.WithSKU()`,
  `pluginsdk.WithRegion()` — existing functional options
- **Returns**: `*pbc.ResourceDescriptor` — existing protobuf message type

## Validation Rules

- No validation needed — the helper delegates to `NewResourceDescriptor` which handles
  all construction
- Empty `id` and `arn` strings are handled by conditional `WithID`/`WithARN` application

## State Transitions

N/A — stateless function, no state transitions.
