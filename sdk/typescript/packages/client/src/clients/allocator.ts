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
  AllocatorService,
  AllocateRequest,
  AllocateResponse,
} from "../generated/finfocus/v1/allocation_pb.js";
import { ClientConfig } from "./auxiliary.js";

/**
 * Client for plugins that serve AllocatorService.
 *
 * Errors propagate as ConnectError with the code the allocator returned
 * (for example Code.InvalidArgument for a bad policy). Requests are not
 * validated client-side, and conservation is not checked client-side.
 */
export class AllocatorClient {
  private client: Client<typeof AllocatorService>;

  constructor(config: ClientConfig) {
    const transport = config.transport || createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(AllocatorService, transport);
  }

  /** Divides priced resources across workloads according to the policy. */
  async allocate(request: AllocateRequest): Promise<AllocateResponse> {
    return this.client.allocate(request);
  }
}
