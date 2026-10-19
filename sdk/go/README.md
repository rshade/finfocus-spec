# FinFocus Go SDK

The Go SDK provides generated protocol bindings, plugin serving and validation, provider-neutral testing,
and registry helpers. See [plugin development](pluginsdk/README.md) and [testing](testing/README.md).

## Optional cost parsing

Implement CostParseProvider.Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error).
Serve registers the interface on gRPC and Connect and infers COST_PARSE alongside existing capabilities.
Configure parser limits before Serve; malformed/missing limits reject startup without invoking Parse.

```go
info := pluginsdk.NewPluginInfo("my-parser", "1.0.0",
    pluginsdk.WithParseLimits([]string{"cfn"}, []string{"content", "path"}, 1048576),
    pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE),
)
// Pass info as ServeConfig.PluginInfo and your implementation as ServeConfig.Plugin.
```

The explicit override is for parser-only operation; mixed pricing/parser plugins can retain automatic inference.
Providers enforce format options, contained filesystem reads and consumed-byte limits. Shared request/response
validators and separate mock/conformance tools are available in sdk/go/testing.
See [cost parsing](../../docs/cost-parse.md) for source-linked partial results, warnings, errors, client transport caps,
provider selection and caching. Terraform/Pulumi ingestion remain core-native.
