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

import { createClient, Client, Code, ConnectError } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  RecommendationScorerService,
  ScoreRecommendationsRequest,
  ScoreRecommendationsResponse,
  ScoreSignal,
} from "../generated/finfocus/v1/scoring_pb.js";
import { ClientConfig } from "./auxiliary.js";

/**
 * Client for plugins that serve RecommendationScorerService and advertise
 * PluginCapability.RECOMMENDATION_SCORING.
 *
 * Errors propagate as ConnectError with the code the scorer returned (for
 * example Code.InvalidArgument for an empty request). Requests and responses
 * are not validated client-side. Set `sessionId` on every batch of one host
 * operation so duplicate group ids match across batches; the response echoes
 * it when the scorer honors sessions. A score is a ranking signal, never approval
 * to act.
 */
export class RecommendationScorerClient {
  private client: Client<typeof RecommendationScorerService>;

  constructor(config: ClientConfig) {
    const transport = config.transport || createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(RecommendationScorerService, transport);
  }

  /** Scores each recommendation; results[i] answers recommendations[i]. */
  async scoreRecommendations(request: ScoreRecommendationsRequest): Promise<ScoreRecommendationsResponse> {
    return this.client.scoreRecommendations(request);
  }
}

export const SCORER_MAX_BATCH_SIZE_KEY = "scorer_max_batch_size";
export const SCORER_SUPPORTED_SIGNALS_KEY = "scorer_supported_signals";
export const BATCH_TOO_LARGE_REASON = "BATCH_TOO_LARGE";
export const SCORER_ERROR_DOMAIN = "finfocus.v1.RecommendationScorerService";

export interface ScorerLimits {
  maxBatchSize: number;
  supportedSignals: ScoreSignal[];
}

/**
 * Reads the limits a scorer advertised in GetPluginInfo metadata. Returns
 * undefined when the plugin advertised neither key, so callers fall back to the
 * response fields. Throws on a half-set pair or a malformed value. The
 * response fields of ScoreRecommendations stay authoritative for each call.
 */
export function parseScorerLimits(metadata: Record<string, string>): ScorerLimits | undefined {
  const rawLimit = metadata[SCORER_MAX_BATCH_SIZE_KEY];
  const rawSignals = metadata[SCORER_SUPPORTED_SIGNALS_KEY];
  if (rawLimit === undefined && rawSignals === undefined) {
    return undefined;
  }
  if (rawLimit === undefined || rawSignals === undefined) {
    throw new Error(`${SCORER_MAX_BATCH_SIZE_KEY} and ${SCORER_SUPPORTED_SIGNALS_KEY} must be set together`);
  }
  const maxBatchSize = Number(rawLimit);
  if (!/^\+?[0-9]+$/.test(rawLimit) || !Number.isSafeInteger(maxBatchSize) || maxBatchSize < 1 || maxBatchSize > 2147483647) {
    throw new Error(`${SCORER_MAX_BATCH_SIZE_KEY} is "${rawLimit}", want an integer from 1 to 2147483647`);
  }
  const supportedSignals: ScoreSignal[] = [];
  for (const name of rawSignals.split(",")) {
    const value = name === name.toLowerCase() ? ScoreSignal[name.toUpperCase() as keyof typeof ScoreSignal] : undefined;
    if (name === "" || typeof value !== "number" || value === ScoreSignal.UNSPECIFIED) {
      throw new Error(`${SCORER_SUPPORTED_SIGNALS_KEY} holds unknown signal "${name}"`);
    }
    if (supportedSignals.includes(value)) {
      throw new Error(`${SCORER_SUPPORTED_SIGNALS_KEY} repeats "${name}"`);
    }
    supportedSignals.push(value);
  }
  return { maxBatchSize, supportedSignals };
}

function readVarint(bytes: Uint8Array, start: number): { value: number; next: number } | undefined {
  let value = 0;
  let scale = 1;
  for (let i = start; i < bytes.length && i < start + 10; i++) {
    value += (bytes[i] & 0x7f) * scale;
    if ((bytes[i] & 0x80) === 0) {
      return { value, next: i + 1 };
    }
    scale *= 128;
  }
  return undefined;
}

/**
 * Reads the string fields of a protobuf message, skipping every field it does
 * not want across all wire types. The last occurrence of a field wins, as in
 * protobuf. Returns undefined when the bytes are malformed.
 */
function readStrings(bytes: Uint8Array, fields: number[]): Map<number, string> | undefined {
  const found = new Map<number, string>();
  let i = 0;
  while (i < bytes.length) {
    const tag = readVarint(bytes, i);
    if (tag === undefined) {
      return undefined;
    }
    i = tag.next;
    const field = Math.floor(tag.value / 8);
    switch (tag.value % 8) {
      case 0: {
        const v = readVarint(bytes, i);
        if (v === undefined) {
          return undefined;
        }
        i = v.next;
        break;
      }
      case 1:
        i += 8;
        break;
      case 5:
        i += 4;
        break;
      case 2: {
        const len = readVarint(bytes, i);
        if (len === undefined || len.next + len.value > bytes.length) {
          return undefined;
        }
        if (fields.includes(field)) {
          found.set(field, new TextDecoder().decode(bytes.subarray(len.next, len.next + len.value)));
        }
        i = len.next + len.value;
        break;
      }
      default:
        return undefined;
    }
    if (i > bytes.length) {
      return undefined;
    }
  }
  return found;
}

/**
 * Reports whether err is the error a scorer returns for a batch above its
 * max_batch_size: Code.InvalidArgument with a google.rpc.ErrorInfo detail whose
 * reason is BATCH_TOO_LARGE. Other InvalidArgument errors (empty request,
 * duplicate ids, unsupported signals) return false.
 */
export function isBatchTooLarge(err: unknown): boolean {
  const connectErr = ConnectError.from(err);
  if (connectErr.code !== Code.InvalidArgument) {
    return false;
  }
  return connectErr.details.some((detail) => {
    if (!("type" in detail) || detail.type !== "google.rpc.ErrorInfo") {
      return false;
    }
    const info = readStrings(detail.value, [1, 2]);
    return info?.get(1) === BATCH_TOO_LARGE_REASON && info.get(2) === SCORER_ERROR_DOMAIN;
  });
}
