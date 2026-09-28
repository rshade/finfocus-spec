# TypeScript API: SupplementalDatasetClient

Package: `@finfocus/client` (`sdk/typescript/packages/client`).

```ts
export class SupplementalDatasetClient {
  constructor(config: ClientConfig);

  /** One page of contract commitments. */
  getContractCommitments(
    request: GetContractCommitmentsRequest,
  ): Promise<GetContractCommitmentsResponse>;

  /**
   * Every commitment across all pages, following nextPageToken. The request is
   * cloned, a missing or non-positive pageSize becomes 50, and more than 10
   * consecutive empty pages that still carry a token throw.
   */
  contractCommitments(
    request: GetContractCommitmentsRequest,
  ): AsyncGenerator<ContractCommitment, void, unknown>;
}
```

Also exported: the regenerated `supplemental_pb.ts` (`SupplementalDatasetService`,
`GetContractCommitmentsRequest(Schema)`, `GetContractCommitmentsResponse(Schema)`) and
`PluginCapability.CONTRACT_COMMITMENTS` from `enums_pb.ts`.

Errors propagate as `ConnectError` with the code the plugin returned. Requests are not validated
client-side.
