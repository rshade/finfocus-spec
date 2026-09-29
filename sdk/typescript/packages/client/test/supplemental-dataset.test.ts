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

import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";

import { SupplementalDatasetClient } from "../src/clients/supplemental-dataset.js";
import {
  GetBillingPeriodsRequestSchema,
  GetContractCommitmentsRequestSchema,
  GetInvoiceDetailsRequestSchema,
} from "../src/generated/finfocus/v1/supplemental_pb.js";
import { FocusContractCommitmentCategory } from "../src/generated/finfocus/v1/focus_pb.js";
import { PluginCapability } from "../src/generated/finfocus/v1/enums_pb.js";

const baseUrl = "https://commitments.example.com";
const service = `${baseUrl}/finfocus.v1.SupplementalDatasetService`;
const endpoint = `${service}/GetContractCommitments`;
const periodsEndpoint = `${service}/GetBillingPeriods`;
const detailsEndpoint = `${service}/GetInvoiceDetails`;

function wireCommitment(id: string): Record<string, unknown> {
  return {
    contractCommitmentId: id,
    contractId: "ea-2024",
    contractCommitmentCategory: "FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND",
    contractCommitmentType: "Reserved Instance",
    contractCommitmentPeriodStart: "2025-01-01T00:00:00Z",
    contractCommitmentPeriodEnd: "2026-01-01T00:00:00Z",
    contractCommitmentCost: 12000,
    billingCurrency: "USD",
  };
}

const requests: Record<string, unknown>[] = [];
const periodRequests: Record<string, unknown>[] = [];
const detailRequests: Record<string, unknown>[] = [];

function paged(
  body: Record<string, unknown>,
  field: string,
  first: Record<string, unknown>,
  second: Record<string, unknown>,
): Record<string, unknown> {
  if (body.pageToken === "page-2") {
    return { [field]: [second], totalCount: 2 };
  }
  return { [field]: [first], nextPageToken: "page-2", totalCount: 2 };
}

// Two pages: ri-1, ri-2 then ri-3.
const server = setupServer(
  http.post(periodsEndpoint, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    periodRequests.push(body);
    return HttpResponse.json(paged(body, "billingPeriods",
      { invoiceIssuerName: "A", billingPeriodStart: "2025-06-01T00:00:00Z" },
      { invoiceIssuerName: "B", billingPeriodStart: "2025-07-01T00:00:00Z" }));
  }),
  http.post(detailsEndpoint, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    detailRequests.push(body);
    return HttpResponse.json(paged(body, "invoiceDetails",
      { invoiceDetailId: "line-1", billedCost: 10 },
      { invoiceDetailId: "line-2", billedCost: 0 }));
  }),
  http.post(endpoint, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>;
    requests.push(body);
    if (body.pageToken === "page-2") {
      return HttpResponse.json({ commitments: [wireCommitment("ri-3")], totalCount: 3 });
    }
    return HttpResponse.json({
      commitments: [wireCommitment("ri-1"), wireCommitment("ri-2")],
      nextPageToken: "page-2",
      totalCount: 3,
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  requests.length = 0;
  periodRequests.length = 0;
  detailRequests.length = 0;
});
afterAll(() => server.close());

describe("SupplementalDatasetClient", () => {
  const client = new SupplementalDatasetClient({ baseUrl });

  it("returns one page of commitments with the window sent", async () => {
    const start = timestampFromDate(new Date("2025-06-01T00:00:00Z"));
    const end = timestampFromDate(new Date("2025-07-01T00:00:00Z"));
    const resp = await client.getContractCommitments(
      create(GetContractCommitmentsRequestSchema, { start, end, pageSize: 2 }),
    );

    expect(requests[0]).toMatchObject({
      start: "2025-06-01T00:00:00Z",
      end: "2025-07-01T00:00:00Z",
      pageSize: 2,
    });
    expect(resp.commitments).toHaveLength(2);
    expect(resp.commitments[0].contractCommitmentId).toBe("ri-1");
    expect(resp.commitments[0].contractCommitmentCategory).toBe(
      FocusContractCommitmentCategory.SPEND,
    );
    expect(resp.nextPageToken).toBe("page-2");
    expect(resp.totalCount).toBe(3);
  });

  it("iterates every page in order and defaults the page size to 50", async () => {
    const request = create(GetContractCommitmentsRequestSchema);
    const ids: string[] = [];
    for await (const commitment of client.contractCommitments(request)) {
      ids.push(commitment.contractCommitmentId);
    }

    expect(ids).toEqual(["ri-1", "ri-2", "ri-3"]);
    expect(requests).toHaveLength(2);
    expect(requests[0]).toMatchObject({ pageSize: 50 });
    expect(requests[1]).toMatchObject({ pageSize: 50, pageToken: "page-2" });
    expect(request.pageSize).toBe(0);
    expect(request.pageToken).toBe("");
  });

  it("stops after too many consecutive empty pages", async () => {
    server.use(
      http.post(endpoint, () => HttpResponse.json({ commitments: [], nextPageToken: "again" })),
    );

    const iterate = async () => {
      for await (const _ of client.contractCommitments(create(GetContractCommitmentsRequestSchema))) {
        // no records are ever returned
      }
    };
    await expect(iterate()).rejects.toThrow(/consecutive empty pages/);
  });

  it("propagates Connect errors with their code", async () => {
    server.use(
      http.post(endpoint, () =>
        HttpResponse.json(
          { code: "invalid_argument", message: "invalid contract commitments request: page_size must not be negative, got -1" },
          { status: 400 },
        ),
      ),
    );

    const err = await client
      .getContractCommitments(create(GetContractCommitmentsRequestSchema, { pageSize: -1 }))
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ConnectError);
    expect((err as ConnectError).code).toBe(Code.InvalidArgument);
    expect((err as ConnectError).rawMessage).toContain("page_size must not be negative");
  });

  it("exposes the contract commitments capability", () => {
    expect(PluginCapability.CONTRACT_COMMITMENTS).toBe(16);
  });

  it("pages billing periods and invoice lines", async () => {
    const periods: string[] = [];
    for await (const period of client.billingPeriods(create(GetBillingPeriodsRequestSchema))) {
      periods.push(period.invoiceIssuerName);
    }
    expect(periods).toEqual(["A", "B"]);
    expect(periodRequests[0]).toMatchObject({ pageSize: 50 });
    expect(periodRequests[1]).toMatchObject({ pageToken: "page-2" });

    const ids: string[] = [];
    for await (const line of client.invoiceDetails(create(GetInvoiceDetailsRequestSchema, { pageSize: 1 }))) {
      ids.push(line.invoiceDetailId);
    }
    expect(ids).toEqual(["line-1", "line-2"]);
    expect(detailRequests[0]).toMatchObject({ pageSize: 1 });
    expect(detailRequests[1]).toMatchObject({ pageSize: 1, pageToken: "page-2" });
  });

  it("exposes the invoice data capability", () => {
    expect(PluginCapability.INVOICE_DATA).toBe(17);
  });
});
