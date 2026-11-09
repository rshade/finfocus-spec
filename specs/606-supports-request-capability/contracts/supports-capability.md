# Contract: Capability-Scoped Supports

## Protocol (`proto/finfocus/v1/costsource.proto`)

```proto
// SupportsRequest asks whether the plugin supports a resource.
message SupportsRequest {
  // resource contains the resource descriptor to check support for
  ResourceDescriptor resource = 1;

  // capability is the operation the host is about to call for this resource,
  // such as PLUGIN_CAPABILITY_PROJECTED_COSTS or PLUGIN_CAPABILITY_ACTUAL_COSTS.
  // PLUGIN_CAPABILITY_UNSPECIFIED means "any capability": answer as before this
  // field existed. Hosts must treat an answer to an unspecified request as
  // covering every capability. A plugin that does not recognize the value should
  // answer as if it were unspecified.
  PluginCapability capability = 2;
}
```

## Go SDK: plugin side (`sdk/go/pluginsdk`)

```go
// RequestedCapability returns the capability a Supports request asks about.
// Known values are returned unchanged. PLUGIN_CAPABILITY_UNSPECIFIED and a nil request
// return PLUGIN_CAPABILITY_UNSPECIFIED. Any other value is treated as
// PLUGIN_CAPABILITY_UNSPECIFIED, so newer hosts keep working, and is logged as a
// warning the first time each distinct value is seen.
func RequestedCapability(req *pbc.SupportsRequest) pbc.PluginCapability
```

- `SupportsProvider` is unchanged.
- `Server.Supports` and `ConnectHandler.Supports` forward the request unchanged. The "Supports handler error"
  log gains `FieldCapability` (`"capability"`).

## Go SDK: host side (`pluginsdk.Client`)

```go
// SupportsFor asks whether the plugin supports resource for one capability.
// It applies the same input checks as Supports.
func (c *Client) SupportsFor(
    ctx context.Context,
    resource *pbc.ResourceDescriptor,
    capability pbc.PluginCapability,
) (*pbc.SupportsResponse, error)

// Supports is unchanged in signature and sends PLUGIN_CAPABILITY_UNSPECIFIED.
func (c *Client) Supports(ctx context.Context, resource *pbc.ResourceDescriptor) (*pbc.SupportsResponse, error)
```

`SupportsResourceType` is unchanged.

## Go SDK: testing (`sdk/go/testing`)

- `MockPlugin.UnsupportedCapabilities []pbc.PluginCapability` is a new field. Its zero value keeps today's
  behavior.
- The Standard conformance check `RPCCorrectness_SupportsCapability` is new; see `data-model.md`.
- `ValidateSupportsRequest` is unchanged and accepts any capability value.

## TypeScript SDK (`@rshade/finfocus-client`)

- The generated `SupportsRequest` gains `capability: PluginCapability`.
- `CostSourceClient.supports(req)` is unchanged and forwards the field.

```ts
const resp = await client.supports(create(SupportsRequestSchema, {
  resource,
  capability: PluginCapability.PROJECTED_COSTS,
}));
```

## Behavior guarantees

| Rule | Source |
|------|--------|
| `UNSPECIFIED` = any capability, the same as before the field existed | FR-002 |
| Old plugins and old hosts interoperate unchanged; `buf breaking` passes | FR-003 |
| Unknown values are read as `UNSPECIFIED` by the helper and accepted by validators | FR-005, FR-012 |
| The SDK never answers per capability on a plugin's behalf; the default answer is unchanged | Assumptions |
| A plugin answering "supported" for a specific capability should also answer "supported" to unspecified | Edge Cases (guidance) |
