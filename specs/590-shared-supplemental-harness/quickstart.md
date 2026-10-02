# Quickstart: Validate the Shared Harness and Duplicate-Key Check

Run these from the repository root.

1. Run the new helper tests (written first; they fail to compile before the helpers exist):

   ```bash
   go test -run 'TestFindDuplicate|TestBufconnHarness' -v ./sdk/go/testing/
   ```

   Expected: every case passes, including the size 64 and 65 cases reporting the same pair.

2. Run the allocation guards:

   ```bash
   go test -run 'AllocationFree' -v ./sdk/go/testing/
   ```

   Expected: `TestContractCommitmentValidatorsAllocationFree`,
   `TestInvoiceDatasetRPCValidatorsAllocationFree`, and
   `TestInvoiceDatasetValidatorsAllocationFree` pass with 0 allocs.

3. Run the full suite, the integration and conformance tests, and lint:

   ```bash
   make test
   go test -tags=integration ./sdk/go/testing/
   golangci-lint run ./...
   ```

   Expected: all pass, and lint reports 0 issues.

4. Confirm the exported API (see [contracts/exported-api.md](contracts/exported-api.md)):

   ```bash
   for h in TestHarness AllocatorHarness ScorerHarness UsageSourceHarness \
            ContractCommitmentHarness InvoiceDatasetHarness; do
     go doc ./sdk/go/testing "$h" | grep -E 'func \(.*\) (Start|Stop|Client)'
   done
   ```

   Expected: three methods are listed for each harness.

5. Confirm there is one implementation of each strategy:

   ```bash
   grep -n 'grpc.DialContext' sdk/go/testing/*.go          # one hit, in bufconn_harness.go
   grep -n 'pairwiseDuplicateLimit {' sdk/go/testing/*.go  # one hit, in findDuplicate
   ```

6. Optional performance A/B against `main` (see [research.md](research.md) R4):

   ```bash
   ./testing.base.test -test.run '^$' -test.bench 'ValidateGetContractCommitmentsResponse' -test.benchmem -test.count 6
   go test -run '^$' -bench 'ValidateGetContractCommitmentsResponse' -benchmem -count 6 ./sdk/go/testing/
   ```
