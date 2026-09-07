# Quickstart: ResolveResourceTypes RPC

**Feature**: 049-resolve-resource-types

This guide shows plugin developers how to add Terraform/IaC type resolution to their plugin.

## Option 1: TypeRegistry (Recommended)

The simplest approach -- declare your type mappings and the SDK handles the RPC automatically.

```go
package main

import (
    "context"

    "github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func main() {
    ctx := context.Background()

    // Create a type registry with Terraform -> Pulumi mappings
    registry := pluginsdk.NewTypeRegistry()
    registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
        "aws_instance":   "aws:ec2/instance:Instance",
        "aws_s3_bucket":  "aws:s3/bucket:Bucket",
        "aws_lambda_function": "aws:lambda/function:Function",
    })

    // Create plugin info (capability is auto-detected from registry)
    info := pluginsdk.NewPluginInfo("aws-public", "v1.0.0",
        pluginsdk.WithProviders("aws"),
    )

    // Serve -- the SDK handles ResolveResourceTypes requests using the registry
    pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
        Plugin:       &MyPlugin{},
        PluginInfo:   info,
        TypeRegistry: registry,
    })
}
```

That's it -- fewer than 10 lines of type-resolution code. The SDK:

1. Auto-detects `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES` from the registry
2. Handles incoming `ResolveResourceTypes` RPC requests
3. Returns only the types your plugin knows (omits unknown types)
4. Reports the capability in `GetPluginInfo` responses

## Option 2: Custom Interface Implementation

For advanced use cases (e.g., dynamic mapping lookup, external mapping sources), implement
the `ResolveResourceTypesProvider` interface directly.

```go
type MyPlugin struct {
    // ...
}

// ResolveResourceTypes implements pluginsdk.ResolveResourceTypesProvider.
// This takes precedence over TypeRegistry if both are configured.
func (p *MyPlugin) ResolveResourceTypes(
    ctx context.Context,
    req *pbc.ResolveResourceTypesRequest,
) (*pbc.ResolveResourceTypesResponse, error) {
    if req.GetSourceFormat() != pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM {
        return &pbc.ResolveResourceTypesResponse{}, nil
    }

    mappings := make(map[string]*pbc.ResourceTypeMapping)
    for _, sourceType := range req.GetSourceTypes() {
        if token, ok := p.lookupPulumiToken(sourceType); ok {
            mappings[sourceType] = &pbc.ResourceTypeMapping{
                PulumiToken: token,
                Supported:   true,
            }
        }
        // Unknown types are simply not added to the map
    }

    return &pbc.ResolveResourceTypesResponse{
        Mappings: mappings,
    }, nil
}
```

The SDK auto-detects the interface and reports the capability -- no manual configuration
needed.

## Testing Your Plugin

### Unit Test with TypeRegistry

```go
func TestTypeRegistry_Resolve(t *testing.T) {
    registry := pluginsdk.NewTypeRegistry()
    registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
        "aws_instance":  "aws:ec2/instance:Instance",
        "aws_s3_bucket": "aws:s3/bucket:Bucket",
    })

    req := &pbc.ResolveResourceTypesRequest{
        SourceFormat: pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
        SourceTypes:  []string{"aws_instance", "aws_s3_bucket", "unknown_type"},
    }
    resp := registry.Resolve(req)

    // Only known types are returned
    require.Len(t, resp.GetMappings(), 2)
    assert.Equal(t, "aws:ec2/instance:Instance", resp.Mappings["aws_instance"].PulumiToken)
    assert.True(t, resp.Mappings["aws_instance"].Supported)

    // Unknown type is omitted
    _, exists := resp.Mappings["unknown_type"]
    assert.False(t, exists)
}
```

### Capability Auto-Detection Test

```go
func TestCapabilityAutoDetection(t *testing.T) {
    plugin := &MyPlugin{}
    info := pluginsdk.NewPluginInfo("test", "1.0.0")

    // If MyPlugin implements ResolveResourceTypesProvider, the capability is auto-detected
    server := pluginsdk.NewServerWithOptions(plugin, nil, nil, info)
    resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
    require.NoError(t, err)

    assert.Contains(t, resp.GetCapabilities(),
        pbc.PluginCapability_PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES)
}
```

## Edge Cases to Handle

| Scenario | Expected Behavior |
|----------|-------------------|
| Empty `source_types` list | Return empty response (no error) |
| `SOURCE_FORMAT_UNSPECIFIED` | Return empty response (no error) |
| Plugin doesn't implement interface or registry | Return empty response (no error) |
| Known type but can't price it | Return mapping with `supported=false` |
| Duplicate types in request | Map semantics deduplicate automatically |
