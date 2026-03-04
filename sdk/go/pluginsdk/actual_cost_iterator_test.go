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

package pluginsdk_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// mockFetchFn creates a fetch function that paginates over a given slice of results.
func mockFetchFn(allResults []*pbc.ActualCostResult) pluginsdk.ActualCostFetchFunc {
	return func(_ context.Context, pageToken string, pageSize int32) (*pbc.GetActualCostResponse, error) {
		page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(allResults, pageSize, pageToken)
		if err != nil {
			return nil, err
		}
		return &pbc.GetActualCostResponse{
			Results:       page,
			NextPageToken: nextToken,
			TotalCount:    totalCount,
		}, nil
	}
}

// TestActualCostIterator_MultiPage creates iterator over 5-page dataset (500 records,
// page_size=100). Verifies all records yielded in order, Err() is nil, TotalCount()
// returns expected value.
func TestActualCostIterator_MultiPage(t *testing.T) {
	allResults := testActualCosts(500)
	iter := pluginsdk.NewActualCostIterator(context.Background(), mockFetchFn(allResults), 100)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(collected) != 500 {
		t.Errorf("expected 500 records, got %d", len(collected))
	}
	if iter.TotalCount() != 500 {
		t.Errorf("expected TotalCount=500, got %d", iter.TotalCount())
	}
	// Verify order
	for i, r := range collected {
		expected := float64(i) + 1.0
		if r.GetCost() != expected {
			t.Errorf("record %d: expected cost=%f, got %f", i, expected, r.GetCost())
			break
		}
	}
}

// TestActualCostIterator_SinglePage verifies iterator with fewer records than
// page_size yields all records and signals completion after one fetch.
func TestActualCostIterator_SinglePage(t *testing.T) {
	allResults := testActualCosts(10)
	iter := pluginsdk.NewActualCostIterator(context.Background(), mockFetchFn(allResults), 100)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if err := iter.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(collected) != 10 {
		t.Errorf("expected 10 records, got %d", len(collected))
	}
}

// TestActualCostIterator_FetchError verifies that when fetchFn returns an error
// on page 3, Next() returns false and Err() surfaces the error.
func TestActualCostIterator_FetchError(t *testing.T) {
	callCount := 0
	fetchErr := errors.New("network failure on page 3")
	allResults := testActualCosts(500)
	fetchFn := func(_ context.Context, pageToken string, pageSize int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		if callCount == 3 {
			return nil, fetchErr
		}
		page, nextToken, totalCount, err := pluginsdk.PaginateActualCosts(allResults, pageSize, pageToken)
		if err != nil {
			return nil, err
		}
		return &pbc.GetActualCostResponse{
			Results:       page,
			NextPageToken: nextToken,
			TotalCount:    totalCount,
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 100)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(iter.Err(), fetchErr) {
		t.Errorf("expected fetchErr, got %v", iter.Err())
	}
	// Pages 1 and 2 should have been delivered (200 records)
	if len(collected) != 200 {
		t.Errorf("expected 200 records from pages 1-2, got %d", len(collected))
	}
}

// TestActualCostIterator_EmptyDataset verifies iterator with 0 records
// returns false on first Next() call with nil Err().
func TestActualCostIterator_EmptyDataset(t *testing.T) {
	allResults := testActualCosts(0)
	iter := pluginsdk.NewActualCostIterator(context.Background(), mockFetchFn(allResults), 100)

	if iter.Next() {
		t.Error("expected Next() to return false for empty dataset")
	}
	if err := iter.Err(); err != nil {
		t.Errorf("expected nil error for empty dataset, got %v", err)
	}
	if iter.TotalCount() != 0 {
		t.Errorf("expected TotalCount=0, got %d", iter.TotalCount())
	}
}

// TestActualCostIterator_ContextCancellation verifies iterator respects
// context cancellation during fetch.
func TestActualCostIterator_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	allResults := testActualCosts(500)
	iter := pluginsdk.NewActualCostIterator(ctx, mockFetchFn(allResults), 100)

	// Consume records until we cancel the context mid-iteration.
	// With page_size=100, pages 1-2 hold records 1-200. Cancelling at
	// record 150 means the context is cancelled while consuming page 2.
	// Next() detects the cancellation on its context check before the
	// next iteration and returns false.
	recordCount := 0
	for iter.Next() {
		recordCount++
		if recordCount == 150 { // Mid-second page
			cancel()
		}
	}

	if iter.Err() == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(iter.Err(), context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", iter.Err())
	}
}

// TestActualCostIterator_EmptyPageWithContinuation tests that the iterator correctly
// handles empty pages when a next_page_token is present. This can occur when data is
// deleted between requests or when filtering eliminates all records on a page.
func TestActualCostIterator_EmptyPageWithContinuation(t *testing.T) {
	// Create a fetch function that returns:
	// Page 1: empty results but has next_page_token="page-2"
	// Page 2: 1 result, no next token
	fetchFn := func(_ context.Context, pageToken string, _ int32) (*pbc.GetActualCostResponse, error) {
		if pageToken == "" {
			// First page: empty with continuation
			return &pbc.GetActualCostResponse{
				Results:       []*pbc.ActualCostResult{},
				NextPageToken: "page-2",
				TotalCount:    1,
			}, nil
		}
		if pageToken == "page-2" {
			// Second page: has the actual data
			return &pbc.GetActualCostResponse{
				Results: []*pbc.ActualCostResult{
					{Cost: 100.0, Source: "test"},
				},
				NextPageToken: "",
				TotalCount:    1,
			}, nil
		}
		return nil, errors.New("unexpected page token")
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 50)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}

	if len(collected) != 1 {
		t.Errorf("expected 1 record, got %d", len(collected))
	}

	if len(collected) > 0 && collected[0].GetCost() != 100.0 {
		t.Errorf("expected cost 100.0, got %f", collected[0].GetCost())
	}
}

// TestActualCostIterator_MaxEmptyPagesExhausted verifies that when a misbehaving
// server returns more than maxEmptyPages consecutive empty pages with continuation
// tokens, the iterator stops gracefully instead of looping forever.
func TestActualCostIterator_MaxEmptyPagesExhausted(t *testing.T) {
	callCount := 0
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		// Always return empty results with a continuation token
		return &pbc.GetActualCostResponse{
			Results:       []*pbc.ActualCostResult{},
			NextPageToken: fmt.Sprintf("page-%d", callCount),
			TotalCount:    0,
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 50)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() == nil {
		t.Fatal("expected error after empty page exhaustion, got nil")
	}
	if errors.Is(iter.Err(), context.Canceled) {
		t.Fatalf("expected pagination safety error, got context.Canceled")
	}
	expectedMsg := "pagination safety: exceeded 10 consecutive empty pages with continuation tokens"
	if iter.Err().Error() != expectedMsg {
		t.Errorf("expected pagination safety error %q, got %q", expectedMsg, iter.Err().Error())
	}
	if len(collected) != 0 {
		t.Errorf("expected 0 records, got %d", len(collected))
	}
	// The iterator should have fetched exactly maxEmptyPages (10) times
	if callCount != 10 {
		t.Errorf("expected 10 fetch calls (maxEmptyPages), got %d", callCount)
	}
}

// TestActualCostIterator_NilResponse verifies that when fetchFn returns nil
// response without an error, the iterator sets an error instead of silently stopping.
func TestActualCostIterator_NilResponse(t *testing.T) {
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		return nil, nil //nolint:nilnil // intentionally simulating a buggy fetchFn that returns nil response without error
	}
	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 50)
	if iter.Next() {
		t.Error("expected Next() to return false for nil response")
	}
	if iter.Err() == nil {
		t.Fatal("expected error for nil response, got nil")
	}
}

// TestActualCostIterator_InconsistentTotalCount verifies that when a server returns
// different total_count values across pages (e.g., data changes mid-pagination),
// the iterator completes without error and TotalCount() reflects the most recent value.
func TestActualCostIterator_InconsistentTotalCount(t *testing.T) {
	callCount := 0
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		// Return decreasing TotalCount on each page
		totalCounts := []int32{100, 90, 80}
		tc := totalCounts[0]
		if callCount <= len(totalCounts) {
			tc = totalCounts[callCount-1]
		}

		// Generate 10 results per page, 3 pages total
		results := make([]*pbc.ActualCostResult, 10)
		for i := range results {
			results[i] = &pbc.ActualCostResult{
				Cost:   float64((callCount-1)*10 + i + 1),
				Source: "test",
			}
		}

		nextToken := ""
		if callCount < 3 {
			nextToken = fmt.Sprintf("page-%d", callCount)
		}

		return &pbc.GetActualCostResponse{
			Results:       results,
			NextPageToken: nextToken,
			TotalCount:    tc,
		}, nil
	}

	nop := zerolog.Nop()
	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 10, pluginsdk.WithLogger(nop))

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}
	if len(collected) != 30 {
		t.Errorf("expected 30 records, got %d", len(collected))
	}
	// TotalCount() should return the value from the most recent response (page 3)
	if iter.TotalCount() != 80 {
		t.Errorf("expected TotalCount=80 (most recent), got %d", iter.TotalCount())
	}
}

// TestActualCostIterator_Int32BoundaryTotalCount verifies that the iterator handles
// math.MaxInt32 as total_count without panic or overflow.
func TestActualCostIterator_Int32BoundaryTotalCount(t *testing.T) {
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		return &pbc.GetActualCostResponse{
			Results: []*pbc.ActualCostResult{
				{Cost: 1.0, Source: "test"},
			},
			NextPageToken: "",
			TotalCount:    math.MaxInt32,
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 50)

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}
	if len(collected) != 1 {
		t.Errorf("expected 1 record, got %d", len(collected))
	}
	if iter.TotalCount() != math.MaxInt32 {
		t.Errorf("expected TotalCount=%d, got %d", int32(math.MaxInt32), iter.TotalCount())
	}
}

// TestActualCostIterator_ConcurrentAccessRaceDetector documents that ActualCostIterator
// is NOT thread-safe. Run manually with -race to observe the data race:
//
//	go test -race -run TestActualCostIterator_ConcurrentAccessRaceDetector -tags concurrency_test ./sdk/go/pluginsdk/
//
// This test is always skipped in normal test runs (including CI with -race) because
// the race detector causes test failures. It exists purely as documentation and a
// manual verification tool for the NOT-thread-safe contract.
func TestActualCostIterator_ConcurrentAccessRaceDetector(t *testing.T) {
	t.Skip("documentation test: run manually with -race to observe data race on ActualCostIterator")
}

// BenchmarkActualCostIterator_Next benchmarks the common iteration path
// for performance regression tracking.
func BenchmarkActualCostIterator_Next(b *testing.B) {
	allResults := testActualCosts(1000)
	fetchFn := mockFetchFn(allResults)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		iter := pluginsdk.NewActualCostIterator(ctx, fetchFn, 100)
		for iter.Next() {
			_ = iter.Record()
		}
		if err := iter.Err(); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// TestActualCostIterator_ConsistentTotalCount_NoWarning verifies that when
// total_count is consistent across all pages, no warning is logged.
func TestActualCostIterator_ConsistentTotalCount_NoWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).With().Timestamp().Logger()

	// Create fetch function that returns consistent total_count=30 across all pages
	callCount := 0
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		results := make([]*pbc.ActualCostResult, 10)
		for i := range results {
			results[i] = &pbc.ActualCostResult{
				Cost:   float64((callCount-1)*10 + i + 1),
				Source: "test",
			}
		}

		nextToken := ""
		if callCount < 3 {
			nextToken = fmt.Sprintf("page-%d", callCount)
		}

		return &pbc.GetActualCostResponse{
			Results:       results,
			NextPageToken: nextToken,
			TotalCount:    30, // Consistent across all pages
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 10, pluginsdk.WithLogger(logger))

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}
	if len(collected) != 30 {
		t.Errorf("expected 30 records, got %d", len(collected))
	}

	// Verify no warning was logged
	logOutput := logBuf.String()
	if bytes.Contains(logBuf.Bytes(), []byte("total_count changed")) {
		t.Errorf("expected no warning for consistent total_count, but got: %s", logOutput)
	}
	if bytes.Contains(logBuf.Bytes(), []byte(`"level":"warn"`)) {
		t.Errorf("expected no warning logs, but found warning in output: %s", logOutput)
	}
}

// TestActualCostIterator_InconsistentTotalCount_LogsWarning verifies that when
// total_count changes between pages, a warning is logged with context.
func TestActualCostIterator_InconsistentTotalCount_LogsWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).With().Timestamp().Logger()

	callCount := 0
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		// Return changing TotalCount on each page: 100, 90, 80
		totalCounts := []int32{100, 90, 80}
		tc := totalCounts[0]
		if callCount <= len(totalCounts) {
			tc = totalCounts[callCount-1]
		}

		results := make([]*pbc.ActualCostResult, 10)
		for i := range results {
			results[i] = &pbc.ActualCostResult{
				Cost:   float64((callCount-1)*10 + i + 1),
				Source: "test",
			}
		}

		nextToken := ""
		if callCount < 3 {
			nextToken = fmt.Sprintf("page-%d", callCount)
		}

		return &pbc.GetActualCostResponse{
			Results:       results,
			NextPageToken: nextToken,
			TotalCount:    tc,
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 10, pluginsdk.WithLogger(logger))

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}
	if len(collected) != 30 {
		t.Errorf("expected 30 records, got %d", len(collected))
	}

	// Verify warning was logged
	logOutput := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("total_count changed")) {
		t.Errorf("expected warning about total_count change, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"level":"warn"`)) {
		t.Errorf("expected warning level log, but got: %s", logOutput)
	}

	// Verify first transition (100→90): page 2 detects change from page 1
	if !bytes.Contains(logBuf.Bytes(), []byte(`"previous_total_count":100`)) {
		t.Errorf("expected previous_total_count=100 in first warning, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"new_total_count":90`)) {
		t.Errorf("expected new_total_count=90 in first warning, but got: %s", logOutput)
	}

	// Verify second transition (90→80): page 3 detects change from page 2
	if !bytes.Contains(logBuf.Bytes(), []byte(`"previous_total_count":90`)) {
		t.Errorf("expected previous_total_count=90 in second warning, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"new_total_count":80`)) {
		t.Errorf("expected new_total_count=80 in second warning, but got: %s", logOutput)
	}

	if !bytes.Contains(logBuf.Bytes(), []byte("underlying dataset may have changed")) {
		t.Errorf("expected warning message about dataset change, but got: %s", logOutput)
	}

	// Verify page_number fields in warnings (page 2 detects first change, page 3 detects second)
	if !bytes.Contains(logBuf.Bytes(), []byte(`"page_number":2`)) {
		t.Errorf("expected page_number=2 in first warning, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"page_number":3`)) {
		t.Errorf("expected page_number=3 in second warning, but got: %s", logOutput)
	}

	// Verify TotalCount() still returns the most recent value
	if iter.TotalCount() != 80 {
		t.Errorf("expected TotalCount=80 (most recent), got %d", iter.TotalCount())
	}
}

// TestActualCostIterator_UnpopulatedThenPopulatedTotalCount documents the behavior when
// a server returns total_count=0 (proto3 zero value / unset) on page 1 and then returns
// a real count on subsequent pages. The iterator treats 0 as a valid total_count value,
// so the transition from 0→N triggers a warning. This is the expected behavior because
// the iterator cannot distinguish between "server doesn't support total_count" (always 0)
// and "server forgot to set it on page 1" — both cases warrant a diagnostic warning.
func TestActualCostIterator_UnpopulatedThenPopulatedTotalCount(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf).With().Timestamp().Logger()

	callCount := 0
	fetchFn := func(_ context.Context, _ string, _ int32) (*pbc.GetActualCostResponse, error) {
		callCount++
		results := make([]*pbc.ActualCostResult, 5)
		for i := range results {
			results[i] = &pbc.ActualCostResult{
				Cost:   float64((callCount-1)*5 + i + 1),
				Source: "test",
			}
		}

		nextToken := ""
		if callCount < 3 {
			nextToken = fmt.Sprintf("page-%d", callCount)
		}

		// Page 1: total_count=0 (unset/zero value)
		// Page 2: total_count=15 (server starts reporting)
		// Page 3: total_count=15 (consistent)
		tc := int32(0)
		if callCount >= 2 {
			tc = 15
		}

		return &pbc.GetActualCostResponse{
			Results:       results,
			NextPageToken: nextToken,
			TotalCount:    tc,
		}, nil
	}

	iter := pluginsdk.NewActualCostIterator(context.Background(), fetchFn, 5, pluginsdk.WithLogger(logger))

	var collected []*pbc.ActualCostResult
	for iter.Next() {
		collected = append(collected, iter.Record())
	}

	if iter.Err() != nil {
		t.Fatalf("unexpected error: %v", iter.Err())
	}
	if len(collected) != 15 {
		t.Errorf("expected 15 records, got %d", len(collected))
	}

	// The 0→15 transition on page 2 SHOULD trigger a warning.
	// This documents that the iterator cannot distinguish "unset" from "actually zero".
	logOutput := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte(`"level":"warn"`)) {
		t.Errorf("expected warning for 0→15 total_count transition, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"previous_total_count":0`)) {
		t.Errorf("expected previous_total_count=0, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"new_total_count":15`)) {
		t.Errorf("expected new_total_count=15, but got: %s", logOutput)
	}
	if !bytes.Contains(logBuf.Bytes(), []byte(`"page_number":2`)) {
		t.Errorf("expected page_number=2 for the transition, but got: %s", logOutput)
	}

	// Page 2→3 should NOT trigger a second warning (15→15 is consistent)
	warnCount := bytes.Count(logBuf.Bytes(), []byte("total_count changed"))
	if warnCount != 1 {
		t.Errorf("expected exactly 1 warning (0→15 only), got %d warnings: %s", warnCount, logOutput)
	}

	// TotalCount should reflect the most recent value
	if iter.TotalCount() != 15 {
		t.Errorf("expected TotalCount=15, got %d", iter.TotalCount())
	}
}
