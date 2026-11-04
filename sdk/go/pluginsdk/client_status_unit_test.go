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

//nolint:testpackage // Exercises the unexported wrapRPCError and withGRPCStatus.
package pluginsdk

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWrapRPCErrorReportsConnectCodeAsGRPCStatus(t *testing.T) {
	for code := codes.Canceled; code <= codes.Unauthenticated; code++ {
		t.Run(code.String(), func(t *testing.T) {
			connectErr := connect.NewError(connect.Code(code), errors.New("child said no"))
			err := wrapRPCError(context.Background(), "GetProjectedCost", connectErr)

			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, code, st.Code())
			assert.Equal(t, "child said no", st.Message(), "the status carries the child's own message")
			assert.Equal(t, "GetProjectedCost RPC failed: "+connectErr.Error(), err.Error(), "Error text is unchanged")

			var still *connect.Error
			require.ErrorAs(t, err, &still, "the connect error stays reachable")
			assert.Same(t, connectErr, still)
		})
	}
}

func TestWrapRPCErrorKeepsDetails(t *testing.T) {
	connectErr := connect.NewError(connect.CodeInvalidArgument, errors.New("too many"))
	detail, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "BATCH_TOO_LARGE"})
	require.NoError(t, err)
	connectErr.AddDetail(detail)

	st := status.Convert(wrapRPCError(context.Background(), "BatchCost", connectErr))
	require.Len(t, st.Details(), 1)
	info, ok := st.Details()[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	assert.Equal(t, "BATCH_TOO_LARGE", info.GetReason())
}

func TestWrapRPCErrorWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	connectErr := connect.NewError(connect.CodeCanceled, errors.New("canceled"))

	err := wrapRPCError(ctx, "Name", connectErr)
	assert.Equal(t, codes.Canceled, status.Code(err))
	require.ErrorIs(t, err, context.Canceled)
}

func TestWithGRPCStatusLeavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("connection refused")
	assert.Same(t, plain, withGRPCStatus(plain), "an error without a connect code is not wrapped")
	require.NoError(t, withGRPCStatus(nil))

	withStatus := status.Error(codes.NotFound, "x")
	assert.Equal(t, withStatus, withGRPCStatus(withStatus), "an error that already has a status is not wrapped")
}
