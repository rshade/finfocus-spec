// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { clone } from "@bufbuild/protobuf";
import { createClient, Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import type { ContractCommitment } from "../generated/finfocus/v1/focus_pb.js";
import {
  SupplementalDatasetService,
  GetContractCommitmentsRequest,
  GetContractCommitmentsRequestSchema,
  GetContractCommitmentsResponse,
} from "../generated/finfocus/v1/supplemental_pb.js";
import { ClientConfig } from "./auxiliary.js";

/** Page size the iterator uses when the request leaves it unset (the Go SDK's DefaultPageSize). */
const DEFAULT_PAGE_SIZE = 50;

/** Consecutive empty pages that still carry a token before the iterator gives up. */
const MAX_EMPTY_PAGES = 10;

/**
 * Client for plugins that serve SupplementalDatasetService (FOCUS
 * supplemental datasets). Check for PluginCapability.CONTRACT_COMMITMENTS
 * before calling getContractCommitments.
 *
 * Errors propagate as ConnectError with the code the plugin returned (for
 * example Code.InvalidArgument). Requests are not validated client-side.
 */
export class SupplementalDatasetClient {
  private client: Client<typeof SupplementalDatasetService>;

  constructor(config: ClientConfig) {
    const transport = config.transport || createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(SupplementalDatasetService, transport);
  }

  /** Returns one page of FOCUS Contract Commitment records. */
  async getContractCommitments(
    request: GetContractCommitmentsRequest,
  ): Promise<GetContractCommitmentsResponse> {
    return this.client.getContractCommitments(request);
  }

  /**
   * Yields every commitment across all pages, following nextPageToken with the
   * same window and page size. The request is cloned, not modified; a missing
   * or non-positive pageSize becomes 50. Throws after 10 consecutive empty
   * pages that still carry a token.
   */
  async *contractCommitments(
    request: GetContractCommitmentsRequest,
  ): AsyncGenerator<ContractCommitment, void, unknown> {
    const next = clone(GetContractCommitmentsRequestSchema, request);
    if (next.pageSize == null || next.pageSize <= 0) {
      next.pageSize = DEFAULT_PAGE_SIZE;
    }
    let emptyPages = 0;
    for (;;) {
      const response = await this.getContractCommitments(next);
      if (response.commitments.length === 0 && response.nextPageToken) {
        emptyPages++;
        if (emptyPages >= MAX_EMPTY_PAGES) {
          throw new Error(`Pagination safety: exceeded ${MAX_EMPTY_PAGES} consecutive empty pages`);
        }
      } else {
        emptyPages = 0;
      }
      yield* response.commitments;
      if (!response.nextPageToken) {
        return;
      }
      next.pageToken = response.nextPageToken;
    }
  }
}
