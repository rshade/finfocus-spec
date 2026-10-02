import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest';
import { setupServer } from 'msw/node';
import { http, HttpResponse } from 'msw';
import { create } from '@bufbuild/protobuf';
import { CostSourceClient } from '../src/clients/cost-source.js';
import { actualCostIterator, recommendationsIterator } from '../src/utils/pagination.js';
import {
  GetActualCostRequestSchema,
  GetRecommendationsRequestSchema,
} from '../src/generated/finfocus/v1/costsource_pb.js';

// Helper to create mock actual cost results
function createMockResults(count: number, startIndex: number = 0): Array<{
  cost: number;
  usageAmount: number;
  usageUnit: string;
  source: string;
  impactMetrics: never[];
}> {
  return Array.from({ length: count }, (_, i) => ({
    cost: startIndex + i + 1,
    usageAmount: 1,
    usageUnit: "hours",
    source: "test-plugin",
    impactMetrics: [],
  }));
}

// Paginated GetActualCost handler that returns pages of results
function paginatedActualCostHandler(totalRecords: number, pageSize: number) {
  return http.post(
    'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
    async ({ request }) => {
      const body = await request.json() as Record<string, unknown>;
      const requestedPageSize = (body.pageSize as number) || pageSize;
      const pageToken = (body.pageToken as string) || "";

      let offset = 0;
      if (pageToken) {
        offset = parseInt(Buffer.from(pageToken, 'base64').toString(), 10);
      }

      const end = Math.min(offset + requestedPageSize, totalRecords);
      const results = createMockResults(end - offset, offset);

      let nextPageToken = "";
      if (end < totalRecords) {
        nextPageToken = Buffer.from(end.toString()).toString('base64');
      }

      return HttpResponse.json({
        results,
        nextPageToken,
        totalCount: totalRecords,
      });
    }
  );
}

// Handler that returns an error on a specific page
function errorOnPageHandler(errorOnPage: number, totalRecords: number = 200) {
  let callCount = 0;
  return http.post(
    'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
    async () => {
      callCount++;
      if (callCount === errorOnPage) {
        return new HttpResponse(null, { status: 500 });
      }
      const results = createMockResults(50, (callCount - 1) * 50);
      // Only emit nextPageToken when more pages exist
      let nextPageToken = "";
      if (callCount * 50 < totalRecords) {
        nextPageToken = Buffer.from((callCount * 50).toString()).toString('base64');
      }
      return HttpResponse.json({
        results,
        nextPageToken,
        totalCount: totalRecords,
      });
    }
  );
}

const server = setupServer();

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('actualCostIterator', () => {
  const client = new CostSourceClient({
    baseUrl: 'https://plugin-test.example.com',
  });

  it('iterates through multiple pages of actual cost results', async () => {
    server.use(paginatedActualCostHandler(200, 50));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    const results: unknown[] = [];
    for await (const result of actualCostIterator(client, request)) {
      results.push(result);
    }

    expect(results).toHaveLength(200);
  });

  it('sends billingAccountId on every page request', async () => {
    const seenBillingAccountIds: unknown[] = [];
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
        async ({ request }) => {
          const body = await request.json() as Record<string, unknown>;
          seenBillingAccountIds.push(body.billingAccountId);
          const offset = body.pageToken
            ? parseInt(Buffer.from(body.pageToken as string, 'base64').toString(), 10)
            : 0;
          const end = Math.min(offset + 50, 120);
          return HttpResponse.json({
            results: createMockResults(end - offset, offset),
            nextPageToken: end < 120 ? Buffer.from(end.toString()).toString('base64') : "",
            totalCount: 120,
          });
        }
      )
    );

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
      billingAccountId: 'ba-123',
    });

    let count = 0;
    for await (const _ of actualCostIterator(client, request)) {
      count++;
    }

    expect(count).toBe(120);
    expect(seenBillingAccountIds).toEqual(['ba-123', 'ba-123', 'ba-123']);
  });

  it('handles single-page results', async () => {
    server.use(paginatedActualCostHandler(10, 50));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    const results: unknown[] = [];
    for await (const result of actualCostIterator(client, request)) {
      results.push(result);
    }

    expect(results).toHaveLength(10);
  });

  it('handles empty results', async () => {
    server.use(paginatedActualCostHandler(0, 50));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    const results: unknown[] = [];
    for await (const result of actualCostIterator(client, request)) {
      results.push(result);
    }

    expect(results).toHaveLength(0);
  });

  it('propagates errors from the client', async () => {
    server.use(errorOnPageHandler(1));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    await expect(async () => {
      for await (const _ of actualCostIterator(client, request)) {
        // consume
      }
    }).rejects.toThrow();
  });

  it('propagates errors mid-pagination', async () => {
    // Page 1 succeeds (50 records), page 2 fails with 500
    server.use(errorOnPageHandler(2));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    const results: unknown[] = [];
    await expect(async () => {
      for await (const result of actualCostIterator(client, request)) {
        results.push(result);
      }
    }).rejects.toThrow();

    // Page 1 should have been delivered before the error on page 2
    expect(results).toHaveLength(50);
  });

  it('aborts after too many consecutive empty pages', async () => {
    let callCount = 0;
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
        async () => {
          callCount++;
          // Always return empty results with a continuation token
          return HttpResponse.json({
            results: [],
            nextPageToken: Buffer.from(callCount.toString()).toString('base64'),
            totalCount: 0,
          });
        }
      )
    );

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    await expect(async () => {
      for await (const _ of actualCostIterator(client, request)) {
        // consume
      }
    }).rejects.toThrow('Pagination safety: exceeded 10 consecutive empty pages');
  });

  it('resets empty page counter when results are returned', async () => {
    let callCount = 0;
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
        async () => {
          callCount++;
          // Alternate: 5 empty pages, then 1 page with results, then done
          if (callCount <= 5) {
            return HttpResponse.json({
              results: [],
              nextPageToken: Buffer.from(callCount.toString()).toString('base64'),
              totalCount: 10,
            });
          }
          return HttpResponse.json({
            results: createMockResults(10),
            nextPageToken: "",
            totalCount: 10,
          });
        }
      )
    );

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    const results: unknown[] = [];
    for await (const result of actualCostIterator(client, request)) {
      results.push(result);
    }

    expect(results).toHaveLength(10);
  });

  it('does not mutate the original request', async () => {
    server.use(paginatedActualCostHandler(100, 50));

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
      pageToken: '',
    });

    const originalToken = request.pageToken;

    for await (const _ of actualCostIterator(client, request)) {
      // consume all results
    }

    expect(request.pageToken).toBe(originalToken);
  });

  it('throws when server returns null response body', async () => {
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
        async () => {
          return HttpResponse.json(null);
        }
      )
    );

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      pageSize: 50,
    });

    await expect(async () => {
      for await (const _ of actualCostIterator(client, request)) {
        // consume
      }
    }).rejects.toThrow(); // Transport or null guard catches null response
  });

  it('defaults pageSize to 50 when not specified', async () => {
    // Track what pageSize the server receives
    let receivedPageSize = 0;
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetActualCost',
        async ({ request }) => {
          const body = await request.json() as Record<string, unknown>;
          receivedPageSize = (body.pageSize as number) || 0;

          return HttpResponse.json({
            results: createMockResults(10),
            nextPageToken: "",
            totalCount: 10,
          });
        }
      )
    );

    const request = create(GetActualCostRequestSchema, {
      resourceId: 'i-abc123',
      // pageSize intentionally omitted (defaults to 0)
    });

    const results: unknown[] = [];
    for await (const result of actualCostIterator(client, request)) {
      results.push(result);
    }

    // The iterator should have defaulted pageSize to 50
    expect(receivedPageSize).toBe(50);
    expect(results).toHaveLength(10);
  });
});

describe('recommendationsIterator', () => {
  const client = new CostSourceClient({
    baseUrl: 'https://plugin-test.example.com',
  });

  it('iterates through multiple pages of recommendations', async () => {
    let callCount = 0;
    const totalRecords = 120;
    const pageSize = 50;

    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetRecommendations',
        async () => {
          callCount++;
          const offset = (callCount - 1) * pageSize;
          const end = Math.min(offset + pageSize, totalRecords);
          const count = end - offset;

          const recommendations = Array.from({ length: count }, (_, i) => ({
            id: `rec-${offset + i + 1}`,
            description: `Recommendation ${offset + i + 1}`,
            source: 'test-plugin',
          }));

          let nextPageToken = "";
          if (end < totalRecords) {
            nextPageToken = Buffer.from(end.toString()).toString('base64');
          }

          return HttpResponse.json({
            recommendations,
            nextPageToken,
          });
        }
      )
    );

    const request = create(GetRecommendationsRequestSchema, {
      pageSize: 50,
    });

    const results: unknown[] = [];
    for await (const rec of recommendationsIterator(client, request)) {
      results.push(rec);
    }

    expect(results).toHaveLength(120);
  });

  it('throws when server returns null response body', async () => {
    server.use(
      http.post(
        'https://plugin-test.example.com/finfocus.v1.CostSourceService/GetRecommendations',
        async () => {
          return HttpResponse.json(null);
        }
      )
    );

    const request = create(GetRecommendationsRequestSchema, {
      pageSize: 50,
    });

    await expect(async () => {
      for await (const _ of recommendationsIterator(client, request)) {
        // consume
      }
    }).rejects.toThrow(); // Transport or null guard catches null response
  });
});
