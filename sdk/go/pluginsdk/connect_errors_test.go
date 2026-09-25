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

//nolint:testpackage // Tests the unexported toConnectError helper
package pluginsdk

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToConnectError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.NoError(t, toConnectError(nil))
	})

	t.Run("connect error unchanged", func(t *testing.T) {
		in := connect.NewError(connect.CodeNotFound, errors.New("missing"))
		assert.Same(t, in, toConnectError(in))
	})

	t.Run("plain error unchanged", func(t *testing.T) {
		in := errors.New("boom")
		assert.Same(t, in, toConnectError(in))
	})

	for code := codes.Canceled; code <= codes.Unauthenticated; code++ {
		t.Run("grpc "+code.String(), func(t *testing.T) {
			got := toConnectError(status.Error(code, "msg"))

			var connectErr *connect.Error
			require.ErrorAs(t, got, &connectErr)
			assert.Equal(t, connect.Code(code), connect.CodeOf(got))
			assert.Equal(t, "msg", connectErr.Message())
		})
	}
}

func BenchmarkToConnectError(b *testing.B) {
	err := status.Error(codes.PermissionDenied, "cannot list pods")
	b.ReportAllocs()
	for b.Loop() {
		_ = toConnectError(err)
	}
}
