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
import type { BillingPeriod, ContractCommitment, InvoiceDetail } from "../generated/finfocus/v1/focus_pb.js";
import {
  SupplementalDatasetService,
  GetBillingPeriodsRequest,
  GetBillingPeriodsRequestSchema,
  GetBillingPeriodsResponse,
  GetContractCommitmentsRequest,
  GetContractCommitmentsRequestSchema,
  GetContractCommitmentsResponse,
  GetInvoiceDetailsRequest,
  GetInvoiceDetailsRequestSchema,
  GetInvoiceDetailsResponse,
} from "../generated/finfocus/v1/supplemental_pb.js";
import { ClientConfig } from "./auxiliary.js";

/** Page size the iterator uses when the request leaves it unset (the Go SDK's DefaultPageSize). */
const DEFAULT_PAGE_SIZE = 50;

/** Consecutive empty pages that still carry a token before the iterator gives up. */
const MAX_EMPTY_PAGES = 10;

/**
 * Client for plugins that serve SupplementalDatasetService (FOCUS
 * supplemental datasets). Check PluginCapability.CONTRACT_COMMITMENTS before
 * getContractCommitments, and PluginCapability.INVOICE_DATA before
 * getBillingPeriods and getInvoiceDetails.
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
    yield* followPages(next, () => this.getContractCommitments(next), (page) => page.commitments);
  }

  /** Returns one page of FOCUS Billing Period records. */
  async getBillingPeriods(request: GetBillingPeriodsRequest): Promise<GetBillingPeriodsResponse> {
    return this.client.getBillingPeriods(request);
  }

  /**
   * Yields every billing period across all pages. The request is cloned, not
   * modified; a missing or non-positive pageSize becomes 50.
   */
  async *billingPeriods(
    request: GetBillingPeriodsRequest,
  ): AsyncGenerator<BillingPeriod, void, unknown> {
    const next = clone(GetBillingPeriodsRequestSchema, request);
    yield* followPages(next, () => this.getBillingPeriods(next), (page) => page.billingPeriods);
  }

  /** Returns one page of FOCUS Invoice Detail records. */
  async getInvoiceDetails(request: GetInvoiceDetailsRequest): Promise<GetInvoiceDetailsResponse> {
    return this.client.getInvoiceDetails(request);
  }

  /**
   * Yields every invoice line across all pages. The request is cloned, not
   * modified; a missing or non-positive pageSize becomes 50.
   */
  async *invoiceDetails(
    request: GetInvoiceDetailsRequest,
  ): AsyncGenerator<InvoiceDetail, void, unknown> {
    const next = clone(GetInvoiceDetailsRequestSchema, request);
    yield* followPages(next, () => this.getInvoiceDetails(next), (page) => page.invoiceDetails);
  }
}

interface PageRequest {
  pageSize: number;
  pageToken: string;
}

interface PageResponse {
  nextPageToken: string;
}

async function* followPages<TReq extends PageRequest, TResp extends PageResponse, TItem>(
  request: TReq,
  fetchPage: () => Promise<TResp>,
  items: (page: TResp) => readonly TItem[],
): AsyncGenerator<TItem, void, unknown> {
  if (request.pageSize == null || request.pageSize <= 0) {
    request.pageSize = DEFAULT_PAGE_SIZE;
  }
  let emptyPages = 0;
  for (;;) {
    const response = await fetchPage();
    const page = items(response);
    if (page.length === 0 && response.nextPageToken) {
      emptyPages++;
      if (emptyPages >= MAX_EMPTY_PAGES) {
        throw new Error(`Pagination safety: exceeded ${MAX_EMPTY_PAGES} consecutive empty pages`);
      }
    } else {
      emptyPages = 0;
    }
    yield* page;
    if (!response.nextPageToken) {
      return;
    }
    request.pageToken = response.nextPageToken;
  }
}
