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

import { protoInt64 } from "@bufbuild/protobuf";
import { createClient, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  CostParseService,
  type ParseRequest,
  type ParseResponse,
} from "../generated/finfocus/v1/parse_pb.js";
import type { ClientConfig } from "./auxiliary.js";

/** Client for the optional CostParseService advertised by a parser plugin. */
export class CostParseClient {
  private client: Client<typeof CostParseService>;

  constructor(config: ClientConfig) {
    const transport = config.transport ?? createConnectTransport({
      baseUrl: config.baseUrl,
      useBinaryFormat: false,
    });
    this.client = createClient(CostParseService, transport);
  }

  /** Parses content or a host-exposed root into source-linked resources and diagnostics.
   * ConnectError codes and details from the provider propagate to the caller.
   */
  async parse(request: ParseRequest): Promise<ParseResponse> {
    return this.client.parse(request);
  }
}

/** GetPluginInfo metadata key for comma-separated parser format names. */
export const PARSE_FORMATS_KEY = "parse_formats";
/** GetPluginInfo metadata key for content and/or path input modes. */
export const PARSE_INPUT_MODES_KEY = "parse_input_modes";
/** GetPluginInfo metadata key for the positive raw input byte cap. */
export const PARSE_MAX_INPUT_BYTES_KEY = "parse_max_input_bytes";
/** ErrorInfo domain for CostParseService errors. */
export const PARSE_ERROR_DOMAIN = "finfocus.v1.CostParseService";
/** ErrorInfo reason for a syntactically valid but unadvertised format. */
export const UNSUPPORTED_FORMAT_REASON = "UNSUPPORTED_FORMAT";
/** ErrorInfo reason for an unadvertised input mode. */
export const UNSUPPORTED_INPUT_MODE_REASON = "UNSUPPORTED_INPUT_MODE";

/** Authoritative parser limits read from GetPluginInfo metadata. */
export interface ParseLimits {
  formats: string[];
  inputModes: string[];
  maxInputBytes: bigint;
}

/**
 * Reads canonical parser metadata. Returns undefined only when all three keys
 * are absent; partial or malformed advertisements throw. Names retain their
 * advertised order, and the byte limit remains an exact positive signed int64.
 */
export function parseParseLimits(metadata: Record<string, string>): ParseLimits | undefined {
  const rawFormats = metadata[PARSE_FORMATS_KEY];
  const rawModes = metadata[PARSE_INPUT_MODES_KEY];
  const rawLimit = metadata[PARSE_MAX_INPUT_BYTES_KEY];
  if (rawFormats === undefined && rawModes === undefined && rawLimit === undefined) {
    return undefined;
  }
  if (rawFormats === undefined || rawModes === undefined || rawLimit === undefined) {
    throw new Error("all three parse metadata keys must be set together");
  }

  const formats = rawFormats.split(",");
  const uniqueFormats = new Set<string>();
  for (const format of formats) {
    // Reject every non-ASCII name character, including final newlines; a JS
    // regular-expression $ anchor alone also matches before a final newline.
    if (format.length < 1 || format.length > 32 || !/^[a-z]/.test(format) || /[^a-z0-9_-]/.test(format)) {
      throw new Error(`${PARSE_FORMATS_KEY} holds invalid format "${format}"`);
    }
    if (uniqueFormats.has(format)) {
      throw new Error(`${PARSE_FORMATS_KEY} repeats "${format}"`);
    }
    uniqueFormats.add(format);
  }

  const inputModes = rawModes.split(",");
  const uniqueModes = new Set<string>();
  for (const mode of inputModes) {
    if (mode !== "content" && mode !== "path") {
      throw new Error(`${PARSE_INPUT_MODES_KEY} holds invalid mode "${mode}"`);
    }
    if (uniqueModes.has(mode)) {
      throw new Error(`${PARSE_INPUT_MODES_KEY} repeats "${mode}"`);
    }
    uniqueModes.add(mode);
  }

  // Bound the decimal length before int64 parsing and reject signs,
  // whitespace and leading zeroes to match Go's canonical int64 spelling.
  if (rawLimit.length > 19 || !/^[1-9]/.test(rawLimit) || /[^0-9]/.test(rawLimit)) {
    throw new Error(`${PARSE_MAX_INPUT_BYTES_KEY} must be a canonical positive int64`);
  }
  if (rawLimit.length === 19 && rawLimit > "9223372036854775807") {
    throw new Error(`${PARSE_MAX_INPUT_BYTES_KEY} exceeds the signed int64 range`);
  }
  const maxInputBytes = protoInt64.parse(rawLimit);
  if (typeof maxInputBytes !== "bigint") {
    throw new Error("parser limits require bigint support");
  }
  return { formats, inputModes, maxInputBytes };
}
