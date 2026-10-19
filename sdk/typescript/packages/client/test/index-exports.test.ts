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

import { describe, it, expect, expectTypeOf } from 'vitest';
import { create as protobufCreate } from '@bufbuild/protobuf';
import {
  actualCostIterator,
  create,
  GetActualCostRequestSchema,
} from '../src/index.js';
import { actualCostIterator as paginationActualCostIterator } from '../src/utils/pagination.js';

import * as publicSDK from '../src/index.js';
import * as parserClient from '../src/clients/cost-parse.js';
import * as parserMessages from '../src/generated/finfocus/v1/parse_pb.js';
import type {
  ParseLimits, ParseRequest, ParseResponse, ParseFileSet, ParseFile, ParsedResource, ParseWarning,
} from '../src/index.js';

describe('package index', () => {
  it('re-exports create from @bufbuild/protobuf', () => {
    expect(create).toBe(protobufCreate);
    const request = create(GetActualCostRequestSchema, { resourceId: 'i-abc123' });
    expect(request.resourceId).toBe('i-abc123');
  });

  it('exports actualCostIterator', () => {
    expect(actualCostIterator).toBe(paginationActualCostIterator);
  });
});

describe('cost parse public exports', () => {
  it('exports the parsing client, limits reader, and metadata constants', () => {
    for (const name of [
      'CostParseClient', 'parseParseLimits', 'PARSE_FORMATS_KEY', 'PARSE_INPUT_MODES_KEY',
      'PARSE_MAX_INPUT_BYTES_KEY', 'PARSE_ERROR_DOMAIN', 'UNSUPPORTED_FORMAT_REASON',
      'UNSUPPORTED_INPUT_MODE_REASON',
    ] as const) {
      expect(publicSDK[name]).toBeDefined();
      expect(publicSDK[name]).toBe(parserClient[name]);
    }
  });

  it('exports generated parsing service, schemas, and severity', () => {
    for (const name of [
      'CostParseService', 'ParseRequestSchema', 'ParseResponseSchema', 'ParseFileSetSchema',
      'ParseFileSchema', 'ParsedResourceSchema', 'ParseWarningSchema', 'ParseWarningSeverity',
    ] as const) {
      expect(publicSDK[name]).toBeDefined();
      expect(publicSDK[name]).toBe(parserMessages[name]);
    }
  });

  it('exports public parsing types', () => {
    expectTypeOf<ParseLimits>().toEqualTypeOf<parserClient.ParseLimits>();
    expectTypeOf<ParseRequest>().toEqualTypeOf<parserMessages.ParseRequest>();
    expectTypeOf<ParseResponse>().toEqualTypeOf<parserMessages.ParseResponse>();
    expectTypeOf<ParseFileSet>().toEqualTypeOf<parserMessages.ParseFileSet>();
    expectTypeOf<ParseFile>().toEqualTypeOf<parserMessages.ParseFile>();
    expectTypeOf<ParsedResource>().toEqualTypeOf<parserMessages.ParsedResource>();
    expectTypeOf<ParseWarning>().toEqualTypeOf<parserMessages.ParseWarning>();
  });
});
