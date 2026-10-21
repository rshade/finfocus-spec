# Historical carbon reporting

`CostSourceService.GetCarbonEmissions` returns provider-reported monthly emissions
inventory. It is independent of projected `CARBON` impact metrics and resource
cost rows. Providers implement `pluginsdk.HistoricalCarbonProvider` and explicitly
advertise `PLUGIN_CAPABILITY_HISTORICAL_CARBON` through `PluginInfo` capabilities.
The SDK never infers this capability: existing plugins embedding the generated
unimplemented server also satisfy the method interface. Without a provider,
the RPC returns `UNIMPLEMENTED` in gRPC and Connect modes.

Requests use inclusive start and exclusive end timestamps, both first-of-month
midnight UTC. Optional account, region and service filters use OR within a
dimension and AND across dimensions. Unspecified method resolves to market-based;
unspecified scope resolves to total. Page size defaults to 50 and is bounded at
1000. Opaque page tokens may contain up to 16 KiB; providers must protect their
integrity and bind them to the normalized query and caller credentials.

Call `ValidateGetCarbonEmissionsRequest` before
`ApplyGetCarbonEmissionsDefaults`. The server applies these to a cloned request
before dispatch and validates the response with
`ValidateGetCarbonEmissionsResponse`. `ValidateCarbonEmissionRecord` is available
for individual rows.

Every `CarbonEmissionRecord` retains its stable ID, provider, source, usage
account, region, service, original monthly period, resolved method and scope,
model version, normalized finite nonnegative `gCO2e` value, and retrieval time.
Stable identity excludes model version: revised estimates must not become new
additive rows. Empty pages indicate no available data without asserting why.
Explicit zero is valid; providers must never invent zeros for missing data.
Hosts must avoid mixing accounting methods or overlapping scopes and must not
allocate monthly inventory to resource cost rows without supporting evidence.

The TypeScript `CostSourceClient.getCarbonEmissions` method exposes the same
contract, and the public client entry point exports its messages and enums.
