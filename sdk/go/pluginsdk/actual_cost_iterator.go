// Copyright 2024 The FinFocus Authors
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

package pluginsdk

import (
	"context"
	"errors"
	"fmt"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ActualCostFetchFunc is the callback type for fetching a page of actual cost results.
// It is called by ActualCostIterator to retrieve each page of data.
type ActualCostFetchFunc func(ctx context.Context, pageToken string, pageSize int32) (*pbc.GetActualCostResponse, error)

// ActualCostIterator provides a standard Go iterator pattern (Next/Record/Err)
// for consuming paginated GetActualCost responses. It lazily fetches pages
// on demand and yields one ActualCostResult at a time.
//
// # Concurrency Safety
//
// ActualCostIterator is NOT safe for concurrent use from multiple goroutines
// (same contract as [database/sql.Rows]). The methods [ActualCostIterator.Next],
// [ActualCostIterator.Record], [ActualCostIterator.TotalCount], and
// [ActualCostIterator.Err] all access mutable state — calling any of these
// concurrently with Next is a data race. If multi-goroutine access is needed,
// callers must provide their own synchronization (e.g., a sync.Mutex).
//
// To verify race-freedom, run the test suite with the race detector:
//
//	go test -race ./pluginsdk/
//
// # Example Usage
//
// The recommended pattern is a single-goroutine iteration loop:
//
//	iter := pluginsdk.NewActualCostIterator(ctx,
//	    func(ctx context.Context, pageToken string, pageSize int32) (*pbc.GetActualCostResponse, error) {
//	        return client.GetActualCost(ctx, &pbc.GetActualCostRequest{
//	            ResourceId: "i-abc123",
//	            PageSize:   pageSize,
//	            PageToken:  pageToken,
//	        })
//	    },
//	    100,
//	)
//
//	for iter.Next() {
//	    record := iter.Record()
//	    // process record
//	}
//	if err := iter.Err(); err != nil {
//	    // handle error
//	}
type ActualCostIterator struct {
	ctx        context.Context
	fetchFn    ActualCostFetchFunc
	pageSize   int32
	pageToken  string
	current    []*pbc.ActualCostResult
	index      int
	totalCount int32
	done       bool
	err        error
}

// NewActualCostIterator creates a new iterator for consuming paginated actual cost responses.
//
// Parameters:
//   - ctx: Context for cancellation and deadline propagation
//   - fetchFn: Callback function that makes the GetActualCost RPC call
//   - pageSize: Number of records to request per page (0 uses server default)
func NewActualCostIterator(
	ctx context.Context,
	fetchFn ActualCostFetchFunc,
	pageSize int32,
) *ActualCostIterator {
	return &ActualCostIterator{
		ctx:      ctx,
		fetchFn:  fetchFn,
		pageSize: pageSize,
		index:    -1,
	}
}

// maxEmptyPages is the maximum number of consecutive empty pages the iterator
// will skip before treating the stream as done. This prevents infinite loops
// when a buggy backend keeps returning empty pages with continuation tokens.
const maxEmptyPages = 10

// Next advances the iterator to the next record. Returns true if a record
// is available via Record(), false when iteration is complete or an error occurred.
// Check Err() after Next returns false to distinguish between completion and error.
//
//nolint:gocognit // Pagination loop with context checks and empty-page skipping requires nested control flow.
func (it *ActualCostIterator) Next() bool {
	if it.done || it.err != nil {
		return false
	}

	// Check context cancellation
	if err := it.ctx.Err(); err != nil {
		it.err = err
		it.done = true
		return false
	}

	it.index++

	// If we have records in the current page, return the next one
	if it.index < len(it.current) {
		// Check context before yielding a buffered record
		if ctxErr := it.ctx.Err(); ctxErr != nil {
			it.err = ctxErr
			it.done = true
			return false
		}
		return true
	}

	// Need to fetch the next page
	// If we've already fetched at least one page and there's no next token, we're done
	if it.current != nil && it.pageToken == "" {
		it.done = true
		return false
	}

	// Iteratively fetch pages, skipping empty ones (up to maxEmptyPages)
	for range maxEmptyPages {
		resp, err := it.fetchFn(it.ctx, it.pageToken, it.pageSize)
		if err != nil {
			it.err = err
			it.done = true
			return false
		}

		// Re-check context after fetch (may have been canceled during RPC)
		if ctxErr := it.ctx.Err(); ctxErr != nil {
			it.err = ctxErr
			it.done = true
			return false
		}

		// Guard against nil response from fetchFn
		if resp == nil {
			it.err = errors.New("fetchFn returned nil response without error")
			it.done = true
			return false
		}

		it.current = resp.GetResults()
		it.pageToken = resp.GetNextPageToken()
		it.totalCount = resp.GetTotalCount()
		it.index = 0

		if len(it.current) > 0 {
			return true
		}

		// Empty page: if no continuation token, we're done
		if it.pageToken == "" {
			it.done = true
			return false
		}
		// Otherwise continue loop to fetch the next page
	}

	// Exceeded maxEmptyPages consecutive empty pages
	it.done = true
	if ctxErr := it.ctx.Err(); ctxErr != nil {
		it.err = ctxErr
	} else {
		it.err = fmt.Errorf(
			"pagination safety: exceeded %d consecutive empty pages with continuation tokens",
			maxEmptyPages,
		)
	}
	return false
}

// Record returns the current ActualCostResult. Only valid after Next() returns true.
func (it *ActualCostIterator) Record() *pbc.ActualCostResult {
	if it.index >= 0 && it.index < len(it.current) {
		return it.current[it.index]
	}
	return nil
}

// Err returns the first error encountered during iteration.
// Should be checked after Next() returns false.
func (it *ActualCostIterator) Err() error {
	return it.err
}

// TotalCount returns the total_count from the most recent response.
// Returns 0 if no pages have been fetched or if the server doesn't report total count.
func (it *ActualCostIterator) TotalCount() int32 {
	return it.totalCount
}
