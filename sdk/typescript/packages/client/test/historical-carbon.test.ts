import { afterAll, afterEach, beforeAll, expect, it } from "vitest";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import { CostSourceClient } from "../src/clients/cost-source.js";
import { CarbonEmissionRecordSchema, GetCarbonEmissionsRequestSchema, CarbonAccountingMethod, CarbonEmissionsScope, PluginCapability } from "../src/index.js";

const baseUrl = "https://carbon.example.com";
const server = setupServer(http.post(`${baseUrl}/finfocus.v1.CostSourceService/GetCarbonEmissions`, async ({ request }) => {
  expect(await request.json()).toMatchObject({ accountIds: ["123"], pageToken: "opaque" });
  return HttpResponse.json({ records: [{ id: "stable", provider: "aws", source: "aws-sustainability", accountId: "123", region: "us-east-1", service: "AmazonEC2", start: "2026-01-01T00:00:00Z", end: "2026-02-01T00:00:00Z", method: "CARBON_ACCOUNTING_METHOD_MARKET_BASED", scope: "CARBON_EMISSIONS_SCOPE_TOTAL", modelVersion: "v1", value: 0, unit: "gCO2e", retrievedAt: "2026-03-01T00:00:00Z" }], nextPageToken: "next" });
}));
beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
it("exposes historical capability and preserves monthly zero and provenance", async () => {
  expect(PluginCapability.HISTORICAL_CARBON).toBe(20);
  expect(PluginCapability.HISTORICAL_CARBON).not.toBe(PluginCapability.CARBON);
  const client = new CostSourceClient({ baseUrl });
  const response = await client.getCarbonEmissions(create(GetCarbonEmissionsRequestSchema, { accountIds: ["123"], pageToken: "opaque" }));
  expect(response.nextPageToken).toBe("next");
  expect(response.records[0].value).toBe(0);
  expect(response.records[0]).toMatchObject({ id: "stable", value: 0, unit: "gCO2e", source: "aws-sustainability", modelVersion: "v1", method: CarbonAccountingMethod.MARKET_BASED, scope: CarbonEmissionsScope.TOTAL });
});

it("distinguishes omitted emissions from explicit zero in protobuf JSON", () => {
  const omitted = fromJson(CarbonEmissionRecordSchema, {});
  const zero = fromJson(CarbonEmissionRecordSchema, { value: 0 });
  expect(omitted.value).toBeUndefined();
  expect(zero.value).toBe(0);
  expect(toJson(CarbonEmissionRecordSchema, omitted)).not.toHaveProperty("value");
  expect(toJson(CarbonEmissionRecordSchema, zero)).toHaveProperty("value", 0);
});
