# Quickstart: Validating the Manifest Fix

Run from the repository root.

## 1. Round trip (User Story 1)

```bash
go test ./sdk/go/pluginsdk/ -run 'Manifest' -v
```

Expected: the round-trip test saves a fully populated manifest as `.json`, `.yaml`, and `.yml`; each file
passes `registry.ValidatePluginManifest` and the manifest JSON schema; loading returns a manifest equal to
the original; saving twice gives identical bytes. Legacy-format fixtures (camelCase JSON, prefixed enums,
old YAML field names, integer enums) load.

## 2. Long resource types and supported_resources (User Stories 2 and 3)

```bash
go test ./sdk/go/registry/ -run 'TestValidateSupportedResources|TestSchemaAgreement' -v
```

Expected: a 60-character `azure-native:` token under `azure` passes; each invalid case is rejected by both
the SDK validator and the JSON schema, and the error names the path.

## 3. Methods and capabilities (User Story 4)

```bash
go test ./sdk/go/registry/ -run 'TestServiceMethods|Capabilit|BillingModes|TestSchemaDrift' -v
```

Expected: every `CostSourceService` RPC and every protocol capability validates; unknown names are
rejected; the drift test passes. To see the guard work, delete one method from the schema's `methods`
enum and rerun: the drift test fails and names it.

## 4. Examples and schemas

```bash
go test ./sdk/go/registry/ -run 'TestExampleManifests' -v
npx ajv validate --spec=draft2020 --strict=false -c ajv-formats \
  -s schemas/plugin_manifest.schema.json -d examples/plugins/minimal-plugin.json
make validate-npm
```

Expected: the four baseline example manifests pass; npm schema validation passes.

## 5. Full gates

```bash
make test
make lint-go
make lint-markdown
go test -bench='IsValidPluginCapability|IsValidServiceMethod' -benchmem ./sdk/go/registry/
```

Expected: all pass; the two benchmarks report 0 allocs/op.
