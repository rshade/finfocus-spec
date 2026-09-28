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

import type { Transport } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import type * as http from "http";
import type * as https from "https";

export interface NodeTransportConfig {
  baseUrl: string;
  /** HTTP version used to reach the plugin. Defaults to "1.1". */
  httpVersion?: "1.1" | "2";
  /** Node.js request options for HTTP/1.1, such as a custom `agent` or TLS settings. */
  nodeOptions?: Omit<http.RequestOptions, "signal"> | Omit<https.RequestOptions, "signal">;
  /** Per-call deadline in milliseconds; an elapsed deadline rejects with `Code.DeadlineExceeded`. */
  timeout?: number;
}

/**
 * Creates a Node.js Connect transport for FinFocus clients.
 * Supports both standard HTTP and secure HTTPS connections.
 *
 * @example
 * ```typescript
 * const transport = createNodeTransport({
 *   baseUrl: 'https://plugin.example.com',
 *   timeout: 30000
 * });
 * const client = new CostSourceClient({ baseUrl: 'https://plugin.example.com', transport });
 * ```
 */
export function createNodeTransport(config: NodeTransportConfig): Transport {
  const baseOptions = {
    baseUrl: config.baseUrl,
    defaultTimeoutMs: config.timeout,
  };
  if (config.httpVersion === "2") {
    return createConnectTransport({ ...baseOptions, httpVersion: "2" });
  }
  return createConnectTransport({ ...baseOptions, httpVersion: "1.1", nodeOptions: config.nodeOptions });
}
