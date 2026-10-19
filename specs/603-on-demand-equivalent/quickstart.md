# Quickstart: Validate On-Demand Equivalent Responses

Run commands from the repository root with the Go, Buf, npm, and lint tools required by
`AGENTS.md` installed. The working tree must include the proto and SDK changes.

1. Regenerate bindings and confirm presence types:

   ```bash
   make generate
   rg -n 'OnDemandEquivalentMonthlyCost \*float64|onDemandEquivalentMonthlyCost\?: number' \
     sdk/go/proto/finfocus/v1/costsource.pb.go \
     sdk/typescript/packages/client/src/generated/finfocus/v1/costsource_pb.ts
   ```

   Expect two Go pointer fields and two optional TypeScript properties.

2. Check the wire contract:

   ```bash
   buf lint
   buf breaking --against '.git#branch=main'
   ```

   Both commands should exit without findings.

3. Run the focused Go tests:

   ```bash
   go test ./sdk/go ./sdk/go/pluginsdk ./sdk/go/testing \
     -run 'OnDemandEquivalent|LegacyMockLeaves' -count=1
   ```

   Expect validation, bufconn, batch, legacy, and old-decoder cases to pass.

4. Build and test the TypeScript client:

   ```bash
   npm --prefix sdk/typescript/packages/client run build
   npm --prefix sdk/typescript/packages/client test
   ```

5. Run the repository gate and allocation benchmarks:

   ```bash
   make validate
   go test ./sdk/go/pluginsdk -run '^$' \
     -bench 'BenchmarkValidate(EstimateCostResponse|GetProjectedCostResponse)_Valid$' \
     -benchmem -count=1
   ```

   Expect 0 B/op and 0 allocs/op. If `yamllint` is not globally installed, put the real
   `yamllint` executable on `PATH` before `make validate`.
