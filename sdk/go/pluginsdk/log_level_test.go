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

//nolint:testpackage // Exercises the unexported classifier.
package pluginsdk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// allNonOKCodes lists the 16 non-OK gRPC codes, Canceled (1) through Unauthenticated (16).
func allNonOKCodes() []codes.Code {
	all := make([]codes.Code, 0, 16)
	for c := codes.Canceled; c <= codes.Unauthenticated; c++ {
		all = append(all, c)
	}
	return all
}

func isDeclineCode(c codes.Code) bool {
	switch c { //nolint:exhaustive // Every other code is a fault.
	case codes.InvalidArgument, codes.Unimplemented, codes.NotFound, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}

func TestHandlerLogLevel(t *testing.T) {
	assert.Len(t, allNonOKCodes(), 16)

	for _, code := range allNonOKCodes() {
		want := zerolog.ErrorLevel
		if isDeclineCode(code) {
			want = zerolog.DebugLevel
		}
		t.Run(code.String(), func(t *testing.T) {
			bare := status.Error(code, "x")
			assert.Equal(t, want, handlerLogLevel(bare), "bare status")
			assert.Equal(t, want, handlerLogLevel(fmt.Errorf("context: %w", bare)), "wrapped status")
		})
	}

	t.Run("plain error", func(t *testing.T) {
		assert.Equal(t, zerolog.ErrorLevel, handlerLogLevel(errors.New("boom")))
	})
}

func TestHandlerLogLevelAllocationFree(t *testing.T) {
	err := status.Error(codes.InvalidArgument, "x")
	allocs := testing.AllocsPerRun(100, func() { _ = handlerLogLevel(err) })
	assert.Zero(t, allocs)
}

func BenchmarkHandlerLogLevel(b *testing.B) {
	err := status.Error(codes.InvalidArgument, "x")
	b.ReportAllocs()
	for range b.N {
		_ = handlerLogLevel(err)
	}
}
