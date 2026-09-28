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

import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  ActualCostResultSchema,
  GetActualCostResponseSchema,
  type CostSourceClient,
  type GetActualCostRequest,
} from "@rshade/finfocus-client";
import type { RESTGatewayConfig } from "finfocus-middleware";

export const RPC_PATH = "/finfocus.v1.CostSourceService/GetActualCost";
export const RPC_BODY = { resourceId: "i-123", start: "2026-01-01T00:00:00Z" };
export const RPC_RESULT = {
  results: [{ timestamp: "2026-01-01T00:00:00Z", cost: 3.5, source: "i-123" }],
};

/** A CostSourceClient stand-in; the transport itself is covered by the middleware tests. */
export const stubConfig: RESTGatewayConfig = {
  costSourceClient: {
    getActualCost: async (req: GetActualCostRequest) =>
      create(GetActualCostResponseSchema, {
        results: [
          create(ActualCostResultSchema, {
            timestamp: req.start ?? timestampFromDate(new Date(0)),
            cost: 3.5,
            source: req.resourceId,
          }),
        ],
      }),
  } as unknown as CostSourceClient,
};
