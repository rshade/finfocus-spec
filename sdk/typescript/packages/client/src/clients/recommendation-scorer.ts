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

import { createClient, Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  RecommendationScorerService,
  ScoreRecommendationsRequest,
  ScoreRecommendationsResponse,
} from "../generated/finfocus/v1/scoring_pb.js";
import { ClientConfig } from "./auxiliary.js";

/**
 * Client for plugins that serve RecommendationScorerService and advertise
 * PluginCapability.RECOMMENDATION_SCORING.
 *
 * Errors propagate as ConnectError with the code the scorer returned (for
 * example Code.InvalidArgument for an empty request). Requests and responses
 * are not validated client-side. A score is a ranking signal, never approval
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
