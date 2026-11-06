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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
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

// unregisteredDetail builds a detail whose message type is not in the global registry, as
// the child's own detail types are not linked into a router.
func unregisteredDetail(t *testing.T, value string) (*connect.ErrorDetail, protoreflect.FullName) {
	t.Helper()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    proto.String("child_detail_test.proto"),
		Package: proto.String("childtest"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("ChildDetail"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("note"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				JsonName: proto.String("note"),
			}},
		}},
	}, nil)
	require.NoError(t, err)

	descriptor := file.Messages().Get(0)
	msg := dynamicpb.NewMessage(descriptor)
	msg.Set(descriptor.Fields().Get(0), protoreflect.ValueOfString(value))
	// Only the *anypb.Any is kept, as for a detail read off the wire, so Value must decode it.
	packed, err := anypb.New(msg)
	require.NoError(t, err)
	detail, err := connect.NewErrorDetail(packed)
	require.NoError(t, err)
	return detail, descriptor.FullName()
}

func TestWrapRPCErrorKeepsUndecodableDetailsInOrder(t *testing.T) {
	first, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "FIRST"})
	require.NoError(t, err)
	unknown, unknownName := unregisteredDetail(t, "from the child")
	last, err := connect.NewErrorDetail(&errdetails.ErrorInfo{Reason: "LAST"})
	require.NoError(t, err)

	connectErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("not configured"))
	connectErr.AddDetail(first)
	connectErr.AddDetail(unknown)
	connectErr.AddDetail(last)

	got := status.Convert(wrapRPCError(context.Background(), "Supports", connectErr)).Proto().GetDetails()
	require.Len(t, got, 3, "a detail the router cannot decode is forwarded, not dropped")

	assert.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", got[0].GetTypeUrl())
	assert.Equal(t, "type.googleapis.com/"+string(unknownName), got[1].GetTypeUrl())
	assert.Equal(t, unknown.Bytes(), got[1].GetValue(), "the raw bytes are untouched")
	assert.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", got[2].GetTypeUrl())
}
