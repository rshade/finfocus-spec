package grpcconv_test

import (
	"math"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/rshade/finfocus-spec/sdk/go/internal/grpcconv"
)

func TestCodeToInt32(t *testing.T) {
	tests := []struct {
		name     string
		code     codes.Code
		expected int32
	}{
		{
			name:     "OK code (0)",
			code:     codes.OK,
			expected: 0,
		},
		{
			name:     "NotFound code (5)",
			code:     codes.NotFound,
			expected: 5,
		},
		{
			name:     "Internal code (13)",
			code:     codes.Internal,
			expected: 13,
		},
		{
			name:     "Unauthenticated code (16)",
			code:     codes.Unauthenticated,
			expected: 16,
		},
		{
			name:     "Max valid int32 code",
			code:     codes.Code(math.MaxInt32),
			expected: math.MaxInt32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grpcconv.CodeToInt32(tt.code)
			if result != tt.expected {
				t.Errorf("CodeToInt32(%v) = %v, want %v", tt.code, result, tt.expected)
			}
		})
	}
}

func TestCodeToInt32_Overflow(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
	}{
		{"MaxInt32+1", codes.Code(math.MaxInt32 + 1)},
		{"MaxUint32", codes.Code(math.MaxUint32)},
	}
	expected := int32(codes.Internal)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := grpcconv.CodeToInt32(tt.code)
			if result != expected {
				t.Errorf("CodeToInt32(%v) = %v, want %v (codes.Internal)", tt.code, result, expected)
			}
		})
	}
}

// BenchmarkCodeToInt32 measures the performance of the conversion function.
func BenchmarkCodeToInt32(b *testing.B) {
	b.ReportAllocs()
	code := codes.NotFound
	for range b.N {
		_ = grpcconv.CodeToInt32(code)
	}
}
