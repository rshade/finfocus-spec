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

import { describe, it, expect } from 'vitest';
import { create as protobufCreate } from '@bufbuild/protobuf';
import {
  actualCostIterator,
  create,
  GetActualCostRequestSchema,
} from '../src/index.js';
import { actualCostIterator as paginationActualCostIterator } from '../src/utils/pagination.js';

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
