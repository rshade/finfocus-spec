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
  UsageSourceService,
  GetStatsRequest,
  GetStatsResponse,
} from "../generated/finfocus/v1/usage_pb.js";
import { ClientConfig } from "./auxiliary.js";

/**
 * Client for plugins that serve UsageSourceService.
 *
 * Errors propagate as ConnectError with the code the source returned
 * (for example Code.PermissionDenied). Requests are not validated client-side.
 */
export class UsageSourceClient {
  private client: Client<typeof UsageSourceService>;

  constructor(config: ClientConfig) {
    const transport = config.transport || createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(UsageSourceService, transport);
  }

  /** Returns workload and node usage plus the priceable resources it runs on. */
  async getStats(request: GetStatsRequest): Promise<GetStatsResponse> {
    return this.client.getStats(request);
  }
}
