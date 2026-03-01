package jsonld_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/jsonld"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSerializeStream_BasicBatch(t *testing.T) {
	serializer := jsonld.NewSerializer()

	records := make([]*pbc.FocusCostRecord, 100)
	for i := range 100 {
		records[i] = &pbc.FocusCostRecord{
			BillingAccountId: "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{
				Seconds: int64(1735689600 + i*86400),
			},
			ServiceName:     "Amazon EC2",
			BilledCost:      float64(100 + i),
			BillingCurrency: "USD",
		}
	}

	var buf bytes.Buffer
	result, err := serializer.SerializeSlice(context.Background(), records, &buf)
	if err != nil {
		t.Fatalf("SerializeSlice() failed: %v", err)
	}

	if result.RecordsWritten != 100 {
		t.Errorf("Expected 100 records written, got %d", result.RecordsWritten)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
	}

	// Verify output is valid JSON array
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != 100 {
		t.Errorf("Expected 100 records in output, got %d", len(parsed))
	}
}

func TestSerializeStream_LargeBatch(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Test with 10,000 records as per spec
	numRecords := 10000
	records := make([]*pbc.FocusCostRecord, numRecords)
	for i := range numRecords {
		records[i] = &pbc.FocusCostRecord{
			BillingAccountId: "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{
				Seconds: int64(1735689600 + i*86400),
			},
			ServiceName:     "Amazon EC2",
			BilledCost:      float64(i),
			BillingCurrency: "USD",
		}
	}

	var buf bytes.Buffer
	result, err := serializer.SerializeSlice(context.Background(), records, &buf)
	if err != nil {
		t.Fatalf("SerializeSlice() failed: %v", err)
	}

	if result.RecordsWritten != numRecords {
		t.Errorf("Expected %d records written, got %d", numRecords, result.RecordsWritten)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
	}

	// Verify output is valid JSON
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != numRecords {
		t.Errorf("Expected %d records in output, got %d", numRecords, len(parsed))
	}
}

func TestSerializeStream_ChannelInput(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Create a channel with records
	ch := make(chan *pbc.FocusCostRecord, 5)

	// Add valid records
	for i := range 5 {
		ch <- &pbc.FocusCostRecord{
			BillingAccountId: "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{
				Seconds: int64(1735689600 + i*86400),
			},
			ServiceName:     "Amazon EC2",
			BilledCost:      float64(100 + i),
			BillingCurrency: "USD",
		}
	}
	close(ch)

	var buf bytes.Buffer
	result, err := serializer.SerializeStream(context.Background(), ch, &buf)
	if err != nil {
		t.Fatalf("SerializeStream() failed: %v", err)
	}

	if result.RecordsWritten != 5 {
		t.Errorf("Expected 5 records written, got %d", result.RecordsWritten)
	}

	// Verify output is valid JSON
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}
}

func TestSerializeBatch(t *testing.T) {
	serializer := jsonld.NewSerializer()

	records := []*pbc.FocusCostRecord{
		{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735689600},
			ServiceName:       "Amazon EC2",
			BilledCost:        100.0,
			BillingCurrency:   "USD",
		},
		{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735776000},
			ServiceName:       "Amazon S3",
			BilledCost:        50.0,
			BillingCurrency:   "USD",
		},
	}

	data, result, err := serializer.SerializeBatch(context.Background(), records)
	if err != nil {
		t.Fatalf("SerializeBatch() failed: %v", err)
	}

	if result.RecordsWritten != 2 {
		t.Errorf("Expected 2 records written, got %d", result.RecordsWritten)
	}

	// Verify output is valid JSON array
	var parsed []map[string]interface{}
	if unmarshalErr := json.Unmarshal(data, &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != 2 {
		t.Errorf("Expected 2 records in output, got %d", len(parsed))
	}

	// Check first record has expected fields
	if parsed[0]["serviceName"] != "Amazon EC2" {
		t.Errorf("First record serviceName = %v, want Amazon EC2", parsed[0]["serviceName"])
	}

	// Check second record has expected fields
	if parsed[1]["serviceName"] != "Amazon S3" {
		t.Errorf("Second record serviceName = %v, want Amazon S3", parsed[1]["serviceName"])
	}
}

func TestSerializeStream_EmptyInput(t *testing.T) {
	serializer := jsonld.NewSerializer()

	ch := make(chan *pbc.FocusCostRecord)
	close(ch) // Empty channel

	var buf bytes.Buffer
	result, err := serializer.SerializeStream(context.Background(), ch, &buf)
	if err != nil {
		t.Fatalf("SerializeStream() failed: %v", err)
	}

	if result.RecordsWritten != 0 {
		t.Errorf("Expected 0 records written, got %d", result.RecordsWritten)
	}

	// Verify output is empty JSON array
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != 0 {
		t.Errorf("Expected empty array, got %d elements", len(parsed))
	}
}

func TestStreamResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []*jsonld.StreamError
		expected bool
	}{
		{
			name:     "no errors",
			errors:   nil,
			expected: false,
		},
		{
			name:     "empty errors",
			errors:   []*jsonld.StreamError{},
			expected: false,
		},
		{
			name: "with errors",
			errors: []*jsonld.StreamError{
				{Index: 0, Message: "test error"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &jsonld.StreamResult{Errors: tt.errors}
			if got := result.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStreamError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *jsonld.StreamError
		expected string
	}{
		{
			name: "error without wrapped error",
			err: &jsonld.StreamError{
				Index:   0,
				Message: "test error",
				Err:     nil,
			},
			expected: "stream error at index 0: test error",
		},
		{
			name: "error with wrapped error",
			err: &jsonld.StreamError{
				Index:   5,
				Message: "serialization failed",
				Err:     errors.New("underlying error"),
			},
			expected: "stream error at index 5: serialization failed: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStreamError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")

	tests := []struct {
		name     string
		err      *jsonld.StreamError
		expected error
	}{
		{
			name: "unwrap nil error",
			err: &jsonld.StreamError{
				Index:   0,
				Message: "test",
				Err:     nil,
			},
			expected: nil,
		},
		{
			name: "unwrap wrapped error",
			err: &jsonld.StreamError{
				Index:   1,
				Message: "test",
				Err:     underlyingErr,
			},
			expected: underlyingErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Unwrap()
			// Use errors.Is for proper error comparison (handles wrapped errors correctly)
			if tt.expected == nil {
				if got != nil {
					t.Errorf("Unwrap() = %v, want nil", got)
				}
			} else if !errors.Is(got, tt.expected) {
				t.Errorf("Unwrap() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSerializeStream_ContextCancellation(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Use a buffered channel to allow producer and consumer to run concurrently
	// without blocking each other
	ch := make(chan *pbc.FocusCostRecord, 10)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup

	// Result channel to collect SerializeStream output
	type streamResult struct {
		result *jsonld.StreamResult
		err    error
		buf    bytes.Buffer
	}
	done := make(chan streamResult, 1)

	// Start consumer goroutine (SerializeStream)
	go func() {
		var sr streamResult
		sr.result, sr.err = serializer.SerializeStream(ctx, ch, &sr.buf)
		done <- sr
	}()

	// Channel to signal when first record has been sent (deterministic sync)
	firstSent := make(chan struct{})
	var firstSentOnce sync.Once

	// Start producer goroutine
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		defer close(ch)
		for i := range 10 {
			// Check if context is cancelled before sending
			select {
			case <-ctx.Done():
				return
			default:
			}
			select {
			case ch <- &pbc.FocusCostRecord{
				BillingAccountId: "123456789012",
				ChargePeriodStart: &timestamppb.Timestamp{
					Seconds: int64(1735689600 + i*86400),
				},
				ServiceName:     "Amazon EC2",
				BilledCost:      float64(100 + i),
				BillingCurrency: "USD",
			}:
				// Signal that first record was sent (only once)
				firstSentOnce.Do(func() { close(firstSent) })
			case <-ctx.Done():
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Wait deterministically for first record to be sent before cancelling
	<-firstSent

	// Cancel the context to trigger mid-stream cancellation
	cancel()

	// Wait for SerializeStream to complete
	sr := <-done

	// Wait for producer to exit (cleanup)
	<-producerDone

	// Should return context.Canceled error
	if !errors.Is(sr.err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", sr.err)
	}

	// Should have written some records (at least 1-2 depending on timing)
	if sr.result.RecordsWritten == 0 {
		t.Log("Warning: no records were written before cancellation")
	}

	// Output should still be valid JSON (with closing bracket)
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(sr.buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array after cancellation: %v", unmarshalErr)
	}
}

func TestSerializeStream_ContextTimeout(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Use small buffered channel to reduce contention
	ch := make(chan *pbc.FocusCostRecord, 2)
	producerDone := make(chan struct{}) // Synchronization signal

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Start producer goroutine that's slow
	go func() {
		defer close(producerDone)
		defer close(ch)
		for i := range 10 {
			record := &pbc.FocusCostRecord{
				BillingAccountId: "123456789012",
				ChargePeriodStart: &timestamppb.Timestamp{
					Seconds: int64(1735689600 + i*86400),
				},
				ServiceName:     "Amazon EC2",
				BilledCost:      float64(100 + i),
				BillingCurrency: "USD",
			}
			// Use select to make send respect context cancellation
			select {
			case <-ctx.Done():
				return
			case ch <- record:
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	var buf bytes.Buffer
	result, err := serializer.SerializeStream(ctx, ch, &buf)

	// Wait for producer to exit cleanly (prevents goroutine leak)
	<-producerDone

	// Should return context.DeadlineExceeded error
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}

	// Should have written some records
	t.Logf("Records written before timeout: %d", result.RecordsWritten)

	// Output should still be valid JSON
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array after timeout: %v", unmarshalErr)
	}
}

func TestSerializeStream_PrettyPrint(t *testing.T) {
	serializer := jsonld.NewSerializer(jsonld.WithPrettyPrint(true))

	records := []*pbc.FocusCostRecord{
		{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735689600},
			ServiceName:       "Amazon EC2",
			BilledCost:        100.0,
			BillingCurrency:   "USD",
		},
	}

	data, _, err := serializer.SerializeBatch(context.Background(), records)
	if err != nil {
		t.Fatalf("SerializeBatch() failed: %v", err)
	}

	// Pretty print should include newlines
	output := string(data)
	if !bytes.Contains(data, []byte("\n")) {
		t.Error("Expected pretty-printed output to contain newlines")
	}

	// Should still be valid JSON
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal([]byte(output), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON: %v", unmarshalErr)
	}
}

func TestSerializerConcurrentUse(t *testing.T) {
	t.Parallel()

	// Verify thread-safety of Serializer - multiple goroutines can use the same
	// Serializer instance concurrently without external synchronization.
	serializer := jsonld.NewSerializer()
	const numGoroutines = 10
	const recordsPerGoroutine = 100

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine serializes its own set of records
			records := make([]*pbc.FocusCostRecord, recordsPerGoroutine)
			for j := range recordsPerGoroutine {
				records[j] = &pbc.FocusCostRecord{
					BillingAccountId: "123456789012",
					ChargePeriodStart: &timestamppb.Timestamp{
						Seconds: int64(1735689600 + goroutineID*1000 + j),
					},
					ServiceName:     "Amazon EC2",
					BilledCost:      float64(goroutineID*100 + j),
					BillingCurrency: "USD",
				}
			}

			var buf bytes.Buffer
			result, err := serializer.SerializeSlice(context.Background(), records, &buf)
			if err != nil {
				errChan <- err
				return
			}

			if result.RecordsWritten != recordsPerGoroutine {
				errChan <- errors.New("incorrect record count")
				return
			}

			// Verify output is valid JSON
			var parsed []interface{}
			if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
				errChan <- unmarshalErr
				return
			}

			if len(parsed) != recordsPerGoroutine {
				errChan <- errors.New("incorrect parsed record count")
				return
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent serialization error: %v", err)
	}
}

func TestSerializeStream_ImmediateCancellation(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Use an UNBUFFERED channel that blocks on send - this ensures the context
	// cancellation is detected before any records can be read
	ch := make(chan *pbc.FocusCostRecord)
	// Don't send any records - just close the channel after a delay
	go func() {
		// Small delay to ensure SerializeStream has time to check context first
		time.Sleep(10 * time.Millisecond)
		close(ch)
	}()

	var buf bytes.Buffer
	result, err := serializer.SerializeStream(ctx, ch, &buf)

	// Should return context.Canceled error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Should have written 0 records
	if result.RecordsWritten != 0 {
		t.Errorf("Expected 0 records written, got %d", result.RecordsWritten)
	}

	// CorruptedOnCancel should be false (no records written before cancel)
	if result.CorruptedOnCancel {
		t.Error("Expected CorruptedOnCancel to be false for immediate cancellation")
	}

	// Output should be valid JSON (empty array)
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != 0 {
		t.Errorf("Expected empty array, got %d elements", len(parsed))
	}
}

func TestSerializeStream_CorruptedOnCancel(t *testing.T) {
	serializer := jsonld.NewSerializer()

	// Test that CorruptedOnCancel is set when records were written before cancel
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan *pbc.FocusCostRecord, 10)

	// Channel for synchronizing
	recordSent := make(chan struct{})

	go func() {
		defer close(ch)
		// Send first record
		ch <- &pbc.FocusCostRecord{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735689600},
			ServiceName:       "Amazon EC2",
			BilledCost:        100.0,
			BillingCurrency:   "USD",
		}
		close(recordSent)
		// Wait a bit then cancel
		time.Sleep(10 * time.Millisecond)
		cancel()
		// Send more records that may or may not be processed
		for i := range 5 {
			select {
			case <-ctx.Done():
				return
			case ch <- &pbc.FocusCostRecord{
				BillingAccountId:  "123456789012",
				ChargePeriodStart: &timestamppb.Timestamp{Seconds: int64(1735689600 + (i+1)*86400)},
				ServiceName:       "Amazon EC2",
				BilledCost:        float64(100 + i + 1),
				BillingCurrency:   "USD",
			}:
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Wait for first record to be sent
	<-recordSent

	var buf bytes.Buffer
	result, err := serializer.SerializeStream(ctx, ch, &buf)

	// Should return context.Canceled error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// At least one record should have been written
	if result.RecordsWritten == 0 {
		t.Skip("No records written before cancellation - timing-dependent test")
	}

	// CorruptedOnCancel should be true (records were written before cancel)
	if !result.CorruptedOnCancel {
		t.Error("Expected CorruptedOnCancel to be true when records were written before cancel")
	}
}

func TestSerializeStream_MaxRecordsLimit(t *testing.T) {
	serializer := jsonld.NewSerializer(jsonld.WithStreamLimits(jsonld.StreamLimits{
		MaxRecords: 5,
	}))

	// Create 10 records but limit to 5
	records := make([]*pbc.FocusCostRecord, 10)
	for i := range 10 {
		records[i] = &pbc.FocusCostRecord{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: int64(1735689600 + i*86400)},
			ServiceName:       "Amazon EC2",
			BilledCost:        float64(100 + i),
			BillingCurrency:   "USD",
		}
	}

	var buf bytes.Buffer
	result, err := serializer.SerializeSlice(context.Background(), records, &buf)

	// Should return max records exceeded error
	if !errors.Is(err, jsonld.ErrMaxRecordsExceeded) {
		t.Errorf("Expected ErrMaxRecordsExceeded, got %v", err)
	}

	// Should have written exactly 5 records
	if result.RecordsWritten != 5 {
		t.Errorf("Expected 5 records written, got %d", result.RecordsWritten)
	}

	// Output should still be valid JSON
	var parsed []interface{}
	if unmarshalErr := json.Unmarshal(buf.Bytes(), &parsed); unmarshalErr != nil {
		t.Errorf("Output is not valid JSON array: %v", unmarshalErr)
	}

	if len(parsed) != 5 {
		t.Errorf("Expected 5 records in output, got %d", len(parsed))
	}
}

func TestSerializeStream_MaxRecordSizeLimit(t *testing.T) {
	// Use a very small size limit to trigger the error
	serializer := jsonld.NewSerializer(jsonld.WithStreamLimits(jsonld.StreamLimits{
		MaxRecordSize: 100, // Very small - typical record is ~300+ bytes
	}))

	records := []*pbc.FocusCostRecord{
		{
			BillingAccountId:  "123456789012",
			ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735689600},
			ServiceName:       "Amazon EC2 with a very long service name to exceed the limit",
			BilledCost:        100.0,
			BillingCurrency:   "USD",
		},
	}

	var buf bytes.Buffer
	result, err := serializer.SerializeSlice(context.Background(), records, &buf)

	// Should complete without fatal error
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should have written 0 records (all exceeded size)
	if result.RecordsWritten != 0 {
		t.Errorf("Expected 0 records written, got %d", result.RecordsWritten)
	}

	// Should have 1 error for the oversized record
	if !result.HasErrors() {
		t.Error("Expected errors for oversized record")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	// Check the error is ErrRecordTooLarge
	if !errors.Is(result.Errors[0].Err, jsonld.ErrRecordTooLarge) {
		t.Errorf("Expected ErrRecordTooLarge, got %v", result.Errors[0].Err)
	}
}

func TestStreamResult_IsOutputValid(t *testing.T) {
	tests := []struct {
		name              string
		corruptedOnCancel bool
		expected          bool
	}{
		{
			name:              "valid output - not corrupted",
			corruptedOnCancel: false,
			expected:          true,
		},
		{
			name:              "invalid output - corrupted on cancel",
			corruptedOnCancel: true,
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &jsonld.StreamResult{CorruptedOnCancel: tt.corruptedOnCancel}
			if got := result.IsOutputValid(); got != tt.expected {
				t.Errorf("IsOutputValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStreamResult_ValidateOutputJSON(t *testing.T) {
	tests := []struct {
		name              string
		corruptedOnCancel bool
		data              []byte
		wantErr           bool
	}{
		{
			name:              "not corrupted - valid JSON",
			corruptedOnCancel: false,
			data:              []byte(`[{"key": "value"}]`),
			wantErr:           false,
		},
		{
			name:              "not corrupted - invalid JSON",
			corruptedOnCancel: false,
			data:              []byte(`[{"key": "value"`),
			wantErr:           false, // Should not validate if not corrupted
		},
		{
			name:              "corrupted - valid JSON",
			corruptedOnCancel: true,
			data:              []byte(`[{"key": "value"}]`),
			wantErr:           false, // Valid JSON passes validation
		},
		{
			name:              "corrupted - invalid JSON",
			corruptedOnCancel: true,
			data:              []byte(`[{"key": "value"`),
			wantErr:           true,
		},
		{
			name:              "corrupted - empty data",
			corruptedOnCancel: true,
			data:              []byte(``),
			wantErr:           true,
		},
		{
			name:              "corrupted - partial record",
			corruptedOnCancel: true,
			data:              []byte(`[{"key": "val`),
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &jsonld.StreamResult{CorruptedOnCancel: tt.corruptedOnCancel}
			err := result.ValidateOutputJSON(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !errors.Is(err, jsonld.ErrOutputCorrupted) {
					t.Errorf("ValidateOutputJSON() error = %v, want %v", err, jsonld.ErrOutputCorrupted)
				}
			}
		})
	}
}

func TestStreamResult_ValidateOutputJSON_Integration(t *testing.T) {
	// Integration test: test the usage pattern from the issue description.
	// Uses an unbuffered channel + signal writer to ensure deterministic ordering:
	// 1. SerializeStream reads the record from the unbuffered channel
	// 2. The record is written to output (signalWriter notifies)
	// 3. Only THEN is context cancelled
	// This eliminates the race between ctx.Done() and channel reads in the select.
	serializer := jsonld.NewSerializer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan *pbc.FocusCostRecord) // unbuffered: send blocks until received
	recordWritten := make(chan struct{})
	sw := &signalWriter{signal: recordWritten}

	type streamResult struct {
		result *jsonld.StreamResult
		err    error
	}
	done := make(chan streamResult, 1)

	// Start serialization first so it's ready to receive from the channel
	go func() {
		result, err := serializer.SerializeStream(ctx, ch, sw)
		done <- streamResult{result, err}
	}()

	// Send one record — blocks until SerializeStream reads it (unbuffered channel)
	ch <- &pbc.FocusCostRecord{
		BillingAccountId:  "123456789012",
		ChargePeriodStart: &timestamppb.Timestamp{Seconds: 1735689600},
		ServiceName:       "Amazon EC2",
		BilledCost:        100.0,
		BillingCurrency:   "USD",
	}

	// Wait until the record data has been written to output (RecordsWritten > 0 guaranteed)
	<-recordWritten

	// Cancel context after record is written — CorruptedOnCancel will be true
	cancel()

	// Wait for SerializeStream to return, then close the channel
	sr := <-done
	close(ch)

	// Should return context.Canceled error
	if !errors.Is(sr.err, context.Canceled) {
		t.Fatalf("Expected context.Canceled error, got %v", sr.err)
	}

	// CorruptedOnCancel must be true since records were written before cancel
	if !sr.result.CorruptedOnCancel {
		t.Fatal("Expected CorruptedOnCancel to be true when records were written before cancel")
	}

	// Validate usage pattern from issue description
	if valErr := sr.result.ValidateOutputJSON(sw.Bytes()); valErr != nil {
		if !errors.Is(valErr, jsonld.ErrOutputCorrupted) {
			t.Errorf("Expected ErrOutputCorrupted, got %v", valErr)
		}
		t.Logf("Output corrupted and invalid - would retry: %v", valErr)
	} else {
		t.Log("Output is valid despite cancellation")
	}

	// Assert that IsOutputValid() reports false for cancelled stream
	if sr.result.IsOutputValid() {
		t.Fatal("Expected IsOutputValid() to return false for cancelled stream, got true")
	}
}

// signalWriter wraps bytes.Buffer and signals after the first record data is written.
// The first Write is the opening bracket "[\n"; subsequent writes contain record data.
type signalWriter struct {
	bytes.Buffer

	writes int
	signal chan struct{}
	once   sync.Once
}

func (w *signalWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.writes++
	// Signal after the first record data write (write #1 is "[\n", write #2+ is record data)
	if w.writes > 1 {
		w.once.Do(func() { close(w.signal) })
	}
	return n, err
}
