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
let failWith: { code: string; message: string; details?: { type: string; value: string }[] } | undefined;
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
      scorer: {
        name: "mock-rules",
        calibration: "SCORE_CALIBRATION_RANKING_ONLY",
        model: "jev-1.13.0",
        models: ["jev-1.13.0", "embed-2"],
        providerRequestId: "req-1",
        providerRequestIds: ["req-1", "req-2"],
      },
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
        omittedFields: ["resource.tags", "metadata"],
      }),
    );

    expect(lastRequest).toMatchObject({
      recommendations: [{ id: "r1" }, { id: "r2" }],
      signals: ["SCORE_SIGNAL_RISK"],
      identifierMode: "IDENTIFIER_MODE_PSEUDONYMIZED",
      omittedFields: ["resource.tags", "metadata"],
    });
    expect(resp.results.map((r) => r.recommendationId)).toEqual(["r1", "r2"]);
    expect(resp.results[0].result.case).toBe("scores");
    expect(resp.results[0].result.value).toMatchObject({ risk: 0.3, priority: 2.5, duplicateGroupId: "dup-1" });
    expect(resp.results[1].result.case).toBe("error");
    expect(resp.maxBatchSize).toBe(25);
    expect(resp.scorer?.calibration).toBe(ScoreCalibration.RANKING_ONLY);
    expect(resp.scorer?.models).toEqual(["jev-1.13.0", "embed-2"]);
    expect(resp.scorer?.model).toBe("jev-1.13.0");
    expect(resp.scorer?.providerRequestIds).toEqual(["req-1", "req-2"]);
    expect(resp.results[1].result.case === "error" && resp.results[1].result.value.message).toBe("no resource");
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

  it("accepts a leading plus sign on the batch size", () => {
    expect(parseScorerLimits({ scorer_max_batch_size: "+5", scorer_supported_signals: "risk" }).maxBatchSize).toBe(5);
  });

  it("rejects a batch size above the int32 range", () => {
    expect(() => parseScorerLimits({ scorer_max_batch_size: "2147483648", scorer_supported_signals: "risk" })).toThrow();
  });

  const enc = new TextEncoder();
  const field = (n: number, s: string) => [(n << 3) | 2, s.length, ...enc.encode(s)];
  const oversizeInfo = (extra: number[] = []) =>
    Uint8Array.from([
      ...extra,
      ...field(1, "BATCH_TOO_LARGE"),
      ...field(2, "finfocus.v1.RecommendationScorerService"),
      ...field(3, "ignored"),
    ]);
  const unknownFields = [
    0x50, 0xac, 0x02, // field 10, varint 300
    0x59, 1, 2, 3, 4, 5, 6, 7, 8, // field 11, fixed64
    0x65, 1, 2, 3, 4, // field 12, fixed32
    0xa2, 0x01, 0x03, 120, 121, 122, // field 20, length-delimited
  ];

  it("tells the oversize error from other InvalidArgument errors", () => {
    const oversize = new ConnectError("too many", Code.InvalidArgument, undefined, [
      { type: "google.rpc.ErrorInfo", value: oversizeInfo() },
    ]);
    expect(isBatchTooLarge(oversize)).toBe(true);
    expect(isBatchTooLarge(new ConnectError("dup", Code.InvalidArgument))).toBe(false);
    expect(isBatchTooLarge(new ConnectError("down", Code.Unavailable))).toBe(false);
  });

  it("skips unknown fields of every wire type when reading ErrorInfo", () => {
    const oversize = new ConnectError("too many", Code.InvalidArgument, undefined, [
      { type: "google.rpc.ErrorInfo", value: oversizeInfo(unknownFields) },
    ]);
    expect(isBatchTooLarge(oversize)).toBe(true);
  });

  it("rejects malformed or mismatching ErrorInfo bytes", () => {
    const wrongReason = Uint8Array.from([...field(1, "OTHER"), ...field(2, "finfocus.v1.RecommendationScorerService")]);
    const truncated = oversizeInfo().subarray(0, 10);
    for (const value of [wrongReason, truncated, Uint8Array.from([0x0b])]) {
      const err = new ConnectError("x", Code.InvalidArgument, undefined, [{ type: "google.rpc.ErrorInfo", value }]);
      expect(isBatchTooLarge(err)).toBe(false);
    }
  });

  it("recognizes the oversize error received through the transport", async () => {
    const value = Buffer.from(oversizeInfo(unknownFields)).toString("base64");
    failWith = {
      code: "invalid_argument",
      message: "too many",
      details: [{ type: "google.rpc.ErrorInfo", value }],
    };
    const client = new RecommendationScorerClient({ baseUrl });
    const err = await client
      .scoreRecommendations(create(ScoreRecommendationsRequestSchema, {}))
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ConnectError);
    const connectErr = err as ConnectError;
    expect(connectErr.code).toBe(Code.InvalidArgument);
    expect(connectErr.details).toHaveLength(1);
    expect(connectErr.details[0]).toMatchObject({ type: "google.rpc.ErrorInfo" });
    expect("value" in connectErr.details[0] && connectErr.details[0].value).toBeInstanceOf(Uint8Array);
    expect(isBatchTooLarge(err)).toBe(true);
  });
});
