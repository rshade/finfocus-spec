package jsonld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// StreamError represents an error that occurred during streaming serialization.
type StreamError struct {
	Index   int
	Message string
	Err     error
}

func (e *StreamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("stream error at index %d: %s: %v", e.Index, e.Message, e.Err)
	}
	return fmt.Sprintf("stream error at index %d: %s", e.Index, e.Message)
}

func (e *StreamError) Unwrap() error {
	return e.Err
}

// StreamResult contains the result of a streaming operation.
type StreamResult struct {
	RecordsWritten int
	Errors         []*StreamError
	// CorruptedOnCancel is set to true if context cancellation occurred after
	// records were written. In this case, the output MUST be treated as invalid
	// JSON - it may contain a partially written record followed by the closing
	// bracket.
	//
	// Callers MUST either:
	//   1. Discard the output entirely and retry the operation, OR
	//   2. Validate output with json.Valid() before any use, OR
	//   3. Use a transactional write pattern (write to temp file, rename on success)
	//
	// Do NOT assume the output is usable when this flag is true.
	CorruptedOnCancel bool
}

// StreamLimits configures optional limits for streaming serialization.
// Zero values mean unlimited (no limit enforced).
type StreamLimits struct {
	// MaxRecords is the maximum number of records to serialize.
	// When exceeded, serialization stops and returns ErrMaxRecordsExceeded.
	// 0 = unlimited (default).
	MaxRecords int
	// MaxRecordSize is the maximum size in bytes for a single serialized record.
	// When exceeded, the record is skipped and an error is added to StreamResult.Errors.
	// 0 = unlimited (default).
	MaxRecordSize int
}

// Stream limit sentinel errors.
var (
	// ErrMaxRecordsExceeded is returned when the max records limit is reached.
	ErrMaxRecordsExceeded = errors.New("max records limit exceeded")
	// ErrRecordTooLarge is returned when a single record exceeds the size limit.
	ErrRecordTooLarge = errors.New("record size exceeds limit")
	// ErrOutputCorrupted is returned when context cancellation corrupted the output JSON.
	ErrOutputCorrupted = errors.New("output corrupted during cancellation: JSON is invalid")
)

// HasErrors returns true if any errors occurred during streaming.
func (r *StreamResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// IsOutputValid returns true if the output can be safely used.
// Returns false if context cancellation may have corrupted the output.
func (r *StreamResult) IsOutputValid() bool {
	return !r.CorruptedOnCancel
}

// ValidateOutputJSON validates the output if corruption may have occurred.
// Returns nil if output is valid, error if corrupted and invalid JSON.
//
// Usage pattern after context cancellation:
//
//	result, err := serializer.SerializeStream(ctx, ch, &buf)
//	if err != nil && result.CorruptedOnCancel {
//	    if valErr := result.ValidateOutputJSON(buf.Bytes()); valErr != nil {
//	        // Must retry from scratch
//	        return retry()
//	    }
//	    // Output is valid despite cancellation
//	}
func (r *StreamResult) ValidateOutputJSON(data []byte) error {
	if r.CorruptedOnCancel {
		if !json.Valid(data) {
			return ErrOutputCorrupted
		}
	}
	return nil
}

// Buffer pool configuration constants.
const (
	// defaultBufferSize is the initial capacity for pooled buffers (4KB).
	defaultBufferSize = 4096
	// maxPooledBufferSize is the maximum buffer size to return to pool (64KB).
	maxPooledBufferSize = 65536
)

// bufferPool is a sync.Pool for reusing byte buffers during streaming.
//
//nolint:gochecknoglobals // Intentional optimization for buffer reuse
var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	},
}

func getBuffer() *bytes.Buffer {
	buf, ok := bufferPool.Get().(*bytes.Buffer)
	if !ok {
		return bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	}
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > maxPooledBufferSize {
		// Don't pool very large buffers
		return
	}
	// Defense-in-depth: Reset buffer before returning to pool.
	// While getBuffer() also resets on retrieval, this ensures the pool never
	// contains dirty buffers even if getBuffer() has a bug that skips Reset().
	// This prevents data corruption where previous JSON content could leak
	// into subsequent serialization operations.
	buf.Reset()
	bufferPool.Put(buf)
}

// SerializeStream writes multiple FocusCostRecords to a writer as a JSON-LD array.
//
// The output format is a JSON array: [record1, record2, ...]
// Memory usage is bounded - only one record is held in memory at a time.
//
// Context Cancellation:
//   - The context is checked on each iteration of the loop
//   - When cancelled, returns ctx.Err() with partial results in StreamResult
//   - RecordsWritten reflects records successfully written before cancellation
//   - Cancellation latency: <1ms (checked every iteration via select)
//   - On cancellation, the closing bracket is written to attempt valid JSON output
//   - CRITICAL: If cancellation occurs during a write syscall, the output MUST be
//     treated as invalid JSON. Check StreamResult.CorruptedOnCancel and either:
//     (a) discard output and retry, (b) validate with json.Valid(), or
//     (c) use transactional writes (temp file + rename on success).
//
// Thread Safety:
//   - The Serializer is safe for concurrent use - each SerializeStream call uses
//     pooled buffers from a sync.Pool and writes to its own io.Writer
//   - Multiple goroutines can call SerializeStream concurrently on the same
//     Serializer instance without external synchronization
//
// Error Handling:
//   - Returns immediately if the opening bracket cannot be written
//   - Serialization errors (invalid records) are collected in StreamResult.Errors
//     and processing continues with the next record
//   - Write errors are also collected in StreamResult.Errors, allowing partial
//     output when the writer recovers (e.g., temporary network issues)
//   - Returns immediately if the closing bracket cannot be written
//   - Call StreamResult.HasErrors() to check if any errors occurred
//
//nolint:gocognit // Complexity is inherent to streaming pattern with context cancellation
func (s *Serializer) SerializeStream(
	ctx context.Context,
	records <-chan *pbc.FocusCostRecord,
	w io.Writer,
) (*StreamResult, error) {
	result := &StreamResult{
		Errors: make([]*StreamError, 0),
	}

	// Write opening bracket
	if _, err := w.Write([]byte("[\n")); err != nil {
		return result, fmt.Errorf("failed to write opening bracket: %w", err)
	}

	index := 0
	first := true
	limits := s.options.StreamLimits

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - set corruption flag if any records were written
			// (cancellation during write may leave partial record in output)
			if result.RecordsWritten > 0 {
				result.CorruptedOnCancel = true
			}
			// Write closing bracket and return
			if _, err := w.Write([]byte("\n]")); err != nil {
				return result, fmt.Errorf("failed to write closing bracket on cancellation: %w", err)
			}
			return result, ctx.Err()
		case record, ok := <-records:
			if !ok {
				// Channel closed, we're done
				if _, err := w.Write([]byte("\n]")); err != nil {
					return result, fmt.Errorf("failed to write closing bracket: %w", err)
				}
				return result, nil
			}

			// Check MaxRecords limit before processing
			if limits.MaxRecords > 0 && result.RecordsWritten >= limits.MaxRecords {
				// Write closing bracket and return limit error
				if _, err := w.Write([]byte("\n]")); err != nil {
					return result, fmt.Errorf("failed to write closing bracket: %w", err)
				}
				return result, ErrMaxRecordsExceeded
			}

			// Get a buffer from the pool
			buf := getBuffer()

			// Serialize the record
			data, err := s.Serialize(record)
			if err != nil {
				result.Errors = append(result.Errors, &StreamError{
					Index:   index,
					Message: "serialization failed",
					Err:     err,
				})
				putBuffer(buf)
				index++
				continue
			}

			// Check MaxRecordSize limit after serialization
			if limits.MaxRecordSize > 0 && len(data) > limits.MaxRecordSize {
				result.Errors = append(result.Errors, &StreamError{
					Index:   index,
					Message: fmt.Sprintf("record size %d exceeds limit %d", len(data), limits.MaxRecordSize),
					Err:     ErrRecordTooLarge,
				})
				putBuffer(buf)
				index++
				continue
			}

			// Write separator if not first record
			if !first {
				buf.WriteString(",\n")
			}
			first = false

			// Indent the record for readability in streaming output
			if s.options.PrettyPrint {
				var indented bytes.Buffer
				if indentErr := json.Indent(&indented, data, "  ", "  "); indentErr == nil {
					buf.Write(indented.Bytes())
				} else {
					buf.Write(data)
				}
			} else {
				buf.Write(data)
			}

			// Write to output
			if _, writeErr := w.Write(buf.Bytes()); writeErr != nil {
				result.Errors = append(result.Errors, &StreamError{
					Index:   index,
					Message: "write failed",
					Err:     writeErr,
				})
				putBuffer(buf)
				index++
				continue
			}

			putBuffer(buf)
			result.RecordsWritten++
			index++
		}
	}
}

// SerializeSlice serializes a slice of FocusCostRecords to JSON-LD array.
//
// This is a convenience method that creates a channel from the slice
// and calls SerializeStream.
//
// Context cancellation is supported - if the context is cancelled during
// serialization, partial results will be returned along with ctx.Err().
func (s *Serializer) SerializeSlice(
	ctx context.Context,
	records []*pbc.FocusCostRecord,
	w io.Writer,
) (*StreamResult, error) {
	ch := make(chan *pbc.FocusCostRecord, len(records))
	for _, r := range records {
		ch <- r
	}
	close(ch)
	return s.SerializeStream(ctx, ch, w)
}

// SerializeBatch serializes multiple records and returns the complete JSON-LD array.
//
// Unlike SerializeStream, this method buffers all output in memory.
// Use SerializeStream for very large datasets.
//
// Context cancellation is supported - if the context is cancelled during
// serialization, partial results will be returned along with ctx.Err().
func (s *Serializer) SerializeBatch(
	ctx context.Context,
	records []*pbc.FocusCostRecord,
) ([]byte, *StreamResult, error) {
	var buf bytes.Buffer
	result, err := s.SerializeSlice(ctx, records, &buf)
	if err != nil {
		return nil, result, err
	}
	return buf.Bytes(), result, nil
}
