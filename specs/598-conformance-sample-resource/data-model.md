# Data Model: Plugin-Supplied Sample Resource for Conformance

No wire or proto change. The model is in-memory configuration for one conformance run.

## SuiteConfig (existing, extended)

| Field | Type | Default | Rule |
| --- | --- | --- | --- |
| `SampleResource` (new) | `*pbc.ResourceDescriptor` | nil, meaning `DefaultSampleResource()` | Must pass `ValidateResourceDescriptor` when set; checked before any check runs |

All other fields (`TargetLevel`, `Timeout`, `ParallelRequests`, `EnableBenchmarks`, `BenchmarkDuration`)
are unchanged.

## Sample resource

- One `ResourceDescriptor` that the author asserts their plugin prices.
- Required by the contract rules: `provider` (one of `ValidProviders`), `resource_type`.
- Optional: `sku`, `region`, `tags`, `attributes`, `id`, `arn`, and the other descriptor fields.
- Ownership: copied when the run starts (validation) and again each time a check reads it, so neither
  the author nor any check can change what another check sends.

## ConformanceOption (new)

- `func(*SuiteConfig)`, applied in order to the level's preset configuration.
- `WithSampleResource(r)` sets `SampleResource` to a copy of `r` (nil restores the default).

## TestHarness (existing, extended)

- Unexported `sampleResource *pbc.ResourceDescriptor`, set by `ConformanceSuite.Run` and `RunCategory`.
- `SampleResource()` returns a deep copy, or a fresh default when unset (harnesses created directly
  with `NewTestHarness`).

## Level presets (existing values, now in one place)

| Level | Timeout | Parallel | Benchmarks | Benchmark duration | Categories registered |
| --- | --- | --- | --- | --- | --- |
| Basic | 60 s | 10 | off | (unset) | Spec Validation, RPC Correctness |
| Standard | 60 s | 10 | on | (unset) | all four |
| Advanced | 120 s | 50 | on | 10 s | all four |

These are the exact values the three runners use today; `RunConformance` must reproduce them so
existing results do not change (SC-002).
