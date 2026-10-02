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
import { Code, ConnectError } from "@connectrpc/connect";

import {
  RecommendationScorerClient,
  isBatchTooLarge,
  parseScorerLimits,
} from "../src/clients/recommendation-scorer.js";
import {
  IdentifierMode,
  ScoreCalibration,
  ScoreRecommendationsRequestSchema,
  ScoreSignal,
} from "../src/generated/finfocus/v1/scoring_pb.js";
import { PluginCapability } from "../src/generated/finfocus/v1/enums_pb.js";

const baseUrl = "https://scorer.example.com";
const endpoint = `${baseUrl}/finfocus.v1.RecommendationScorerService/ScoreRecommendations`;

let lastRequest: Record<string, unknown> | undefined;
let failWith: { code: string; message: string } | undefined;
let echoSession = "";

const server = setupServer(
  http.post(endpoint, async ({ request }) => {
    lastRequest = (await request.json()) as Record<string, unknown>;
    if (failWith) {
      return HttpResponse.json(failWith, { status: 400 });
    }
    return HttpResponse.json({
      results: [
        { recommendationId: "r1", scores: { risk: 0.3, priority: 2.5, duplicateGroupId: "dup-1" } },
        { recommendationId: "r2", error: { code: 3, message: "no resource" } },
      ],
      maxBatchSize: 25,
      scorer: { name: "mock-rules", calibration: "SCORE_CALIBRATION_RANKING_ONLY" },
      supportedSignals: ["SCORE_SIGNAL_RISK", "SCORE_SIGNAL_PRIORITY", "SCORE_SIGNAL_DUPLICATE_GROUP"],
      sessionId: echoSession,
    });
  }),
);

beforeAll(() => server.listen());
afterEach(() => {
  server.resetHandlers();
  lastRequest = undefined;
  failWith = undefined;
  echoSession = "";
});
afterAll(() => server.close());

describe("RecommendationScorerClient", () => {
  const client = new RecommendationScorerClient({ baseUrl });

  it("sends recommendations and options and returns index-aligned results", async () => {
    const resp = await client.scoreRecommendations(
      create(ScoreRecommendationsRequestSchema, {
        recommendations: [{ id: "r1" }, { id: "r2" }],
        signals: [ScoreSignal.RISK],
        identifierMode: IdentifierMode.PSEUDONYMIZED,
      }),
    );

    expect(lastRequest).toMatchObject({
      recommendations: [{ id: "r1" }, { id: "r2" }],
      signals: ["SCORE_SIGNAL_RISK"],
      identifierMode: "IDENTIFIER_MODE_PSEUDONYMIZED",
    });
    expect(resp.results.map((r) => r.recommendationId)).toEqual(["r1", "r2"]);
    expect(resp.results[0].result.case).toBe("scores");
    expect(resp.results[0].result.value).toMatchObject({ risk: 0.3, priority: 2.5, duplicateGroupId: "dup-1" });
    expect(resp.results[1].result.case).toBe("error");
    expect(resp.maxBatchSize).toBe(25);
    expect(resp.scorer?.calibration).toBe(ScoreCalibration.RANKING_ONLY);
    expect(resp.supportedSignals).toContain(ScoreSignal.DUPLICATE_GROUP);
  });

  it("sends a session id and reads the echo", async () => {
    echoSession = "host-operation-1";
    const resp = await client.scoreRecommendations(
      create(ScoreRecommendationsRequestSchema, {
        recommendations: [{ id: "r1" }, { id: "r2" }],
        sessionId: "host-operation-1",
      }),
    );

    expect(lastRequest).toMatchObject({ sessionId: "host-operation-1" });
    expect(resp.sessionId).toBe("host-operation-1");
  });

  it("leaves the session id empty when none is sent", async () => {
    const resp = await client.scoreRecommendations(
      create(ScoreRecommendationsRequestSchema, { recommendations: [{ id: "r1" }] }),
    );

    expect(lastRequest).not.toHaveProperty("sessionId");
    expect(resp.sessionId).toBe("");
  });

  it("exposes the capability enum", () => {
    expect(PluginCapability.RECOMMENDATION_SCORING).toBe(18);
  });

  it("propagates the scorer's error code", async () => {
    failWith = { code: "invalid_argument", message: "no recommendations to score" };
    await expect(
      client.scoreRecommendations(create(ScoreRecommendationsRequestSchema, {})),
    ).rejects.toSatisfy((e) => e instanceof ConnectError && e.code === Code.InvalidArgument);
  });
});

describe("scorer advertised limits", () => {
  it("parses advertised metadata", () => {
    expect(
      parseScorerLimits({ scorer_max_batch_size: "40", scorer_supported_signals: "risk,duplicate_group" }),
    ).toEqual({ maxBatchSize: 40, supportedSignals: [ScoreSignal.RISK, ScoreSignal.DUPLICATE_GROUP] });
  });

  it("returns undefined when nothing is advertised", () => {
    expect(parseScorerLimits({ other: "x" })).toBeUndefined();
  });

  it.each([
    [{ scorer_max_batch_size: "5" }],
    [{ scorer_max_batch_size: "0", scorer_supported_signals: "risk" }],
    [{ scorer_max_batch_size: "5", scorer_supported_signals: "risk,vibes" }],
    [{ scorer_max_batch_size: "5", scorer_supported_signals: "risk,risk" }],
    [{ scorer_max_batch_size: "5", scorer_supported_signals: "RISK" }],
    [{ scorer_max_batch_size: "5", scorer_supported_signals: "unspecified" }],
  ])("rejects malformed metadata %j", (md) => {
    expect(() => parseScorerLimits(md)).toThrow();
  });

  it("tells the oversize error from other InvalidArgument errors", () => {
    const enc = new TextEncoder();
    const field = (n: number, s: string) => [(n << 3) | 2, s.length, ...enc.encode(s)];
    const info = Uint8Array.from([
      ...field(1, "BATCH_TOO_LARGE"),
      ...field(2, "finfocus.v1.RecommendationScorerService"),
    ]);
    const oversize = new ConnectError("too many", Code.InvalidArgument, undefined, [
      { type: "google.rpc.ErrorInfo", value: info },
    ]);
    expect(isBatchTooLarge(oversize)).toBe(true);
    expect(isBatchTooLarge(new ConnectError("dup", Code.InvalidArgument))).toBe(false);
    expect(isBatchTooLarge(new ConnectError("down", Code.Unavailable))).toBe(false);
  });
});
