# Quickstart: ResolveResourceTypes Hardening

**Feature**: 050-resolve-resource-types-hardening

This guide shows plugin developers how to use the six hardening improvements on top of
the `ResolveResourceTypes` RPC introduced in spec 049. See that spec's own
`quickstart.md` for the base `TypeRegistry`/`ResolveResourceTypesProvider` setup this
extends.

## 1. Configuring the request-size limit

```go
pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
    Plugin:         &MyPlugin{},
    PluginInfo:     info,
    TypeRegistry:   registry,
    MaxSourceTypes: 500, // optional; defaults to pluginsdk.DefaultMaxSourceTypes (200)
})
```

A request with more than the configured (or default) number of `source_types` entries
is rejected with `InvalidArgument` before any resolution work happens — no code change
needed in your plugin or `TypeRegistry` usage beyond this one config field.

## 2. Setting a caching hint

```go
// Registry-wide default -- recommended for most plugins, since type mappings
// rarely change between calls.
registry := pluginsdk.NewTypeRegistry(
    pluginsdk.WithDefaultTTL(24 * time.Hour),
)
registry.RegisterMappings(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]string{
    "aws_instance": "aws:ec2/instance:Instance",
})
// Every Resolve() response now carries expires_at = now + 24h automatically.
```

For a custom `ResolveResourceTypesProvider` implementation instead of `TypeRegistry`,
set it per response:

```go
resp := pluginsdk.NewResolveResourceTypesResponse(
    pluginsdk.WithResolveResourceTypesExpiresAt(time.Now().Add(24 * time.Hour)),
)
resp.Mappings = mappings
```

Callers on the receiving end check it the same way as the other three cost RPCs:

```go
if pluginsdk.IsResolveResourceTypesExpired(resp, time.Now()) {
    // re-fetch
}
```

## 3. Registering property overrides in bulk

```go
registry.RegisterMappingsWithProperties(pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM, map[string]pluginsdk.TypeMapping{
    "aws_instance": {
        PulumiToken: "aws:ec2/instance:Instance",
        // Mechanical snake_to_camel conversion handles this one -- no override needed.
    },
    "aws_some_resource": {
        PulumiToken: "aws:service/someResource:SomeResource",
        PropertyMappings: map[string]string{
            // A hypothetical non-mechanical rename that plain snake_to_camel would get wrong.
            "legacy_field_name": "modernFieldName",
        },
    },
})
```

`property_mappings` is stored and round-trips over the wire correctly, but — as in spec
049 — finfocus-spec does not itself apply these overrides during resource-shape
translation; that remains the core's responsibility.

## 4. Loading mappings from a data file

```go
registry := pluginsdk.NewTypeRegistry()
if err := registry.LoadMappingsFromFile("mappings/terraform-aws.json"); err != nil {
    log.Fatal(err)
}
```

`mappings/terraform-aws.json`:

```json
{
  "source_format": "TERRAFORM",
  "mappings": {
    "aws_instance": {
      "pulumi_token": "aws:ec2/instance:Instance",
      "supported": true
    }
  }
}
```

finfocus-spec does not ship or maintain any mapping file of its own — this loader is
purely a mechanism for plugin authors (or a separate community-maintained mapping
project) to keep their own data in a versioned file instead of hardcoded Go calls. A
plugin supporting both Terraform and CloudFormation loads two files, one per format:

```go
registry.LoadMappingsFromFile("mappings/terraform-aws.json")
registry.LoadMappingsFromFile("mappings/cloudformation-aws.json")
```

## 5. Verifying conformance

```bash
go test -v ./sdk/go/testing/ -run TestConformance
```

The Basic conformance tier now includes `ResolveResourceTypes_EmptyPlugin` (passes for
any plugin, with or without type-resolution support) and `ResolveResourceTypes_Basic`
(validates well-formed mappings when support is present, using the SDK's `MockPlugin`
fixture, which ships one seeded mapping by default).

## 6. Reading the new metrics

If metrics are enabled (`pluginsdk.MetricsInterceptorWithRegistry`), two new counters
are exported automatically, no plugin code changes required:

```text
finfocus_plugin_resolve_resource_types_resolved_total{plugin_name="my-plugin"}
finfocus_plugin_resolve_resource_types_unresolved_total{plugin_name="my-plugin"}
```

A rising `unresolved` count relative to `resolved` is a signal that your plugin's
mapping table needs more entries.

## Edge Cases to Handle

| Scenario | Expected Behavior |
|----------|-------------------|
| `source_types` exceeds the configured limit | `InvalidArgument` error, no provider/registry invocation |
| `TypeRegistry` has no default TTL configured | `expires_at` is absent, exactly as in spec 049 |
| Mapping file has an unrecognized `source_format` | `LoadMappingsFromJSON`/`File` returns an error, no entries registered |
| Mapping file entry is missing `pulumi_token` | Load fails with a descriptive error |
| Same source type loaded twice (file + `RegisterMapping`, or two files) | Last write wins |
