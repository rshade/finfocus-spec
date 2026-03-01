// Package grpcconv provides internal utilities for converting gRPC types.
//
// This package exists to break circular dependencies between sdk/go/pluginsdk and
// sdk/go/testing. Both packages need gRPC code-to-int32 conversion logic, but
// pluginsdk imports testing (via conformance.go), which would create a circular
// import if testing imported pluginsdk directly.
//
// # Usage
//
// This is an internal package. External consumers should use:
//   - [github.com/rshade/finfocus-spec/sdk/go/pluginsdk.NewResourceError]
//   - [github.com/rshade/finfocus-spec/sdk/go/testing] (batchResourceErrorFromErr)
package grpcconv

import (
	"math"

	"google.golang.org/grpc/codes"
)

// CodeToInt32 converts a gRPC codes.Code to its int32 numeric value for use in protobuf fields.
// If the code's numeric value is outside the int32 range, it returns the numeric value for codes.Internal.
func CodeToInt32(code codes.Code) int32 {
	codeValue := int64(code)
	if codeValue > math.MaxInt32 {
		return int32(codes.Internal)
	}
	//nolint:gosec // Overflow impossible: codeValue is in [0, math.MaxInt32] after the bounds check above.
	return int32(codeValue)
}
